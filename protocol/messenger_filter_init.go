package protocol

import (
	"crypto/ecdsa"
	stderrors "errors"
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/deprecation"
	"github.com/status-im/status-go/protocol/common/shard"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/transport"
)

// InitFilters analyzes chats and contacts in order to setup filters
// which are responsible for retrieving messages.
func (m *Messenger) InitFilters() error {
	// Seed the for color generation
	rand.Seed(time.Now().Unix())

	// Community requests will arrive in this pubsub topic
	if err := m.SubscribeToPubsubTopic(shard.DefaultNonProtectedPubsubTopic(), nil); err != nil {
		return err
	}

	filters, publicKeys, err := m.collectFiltersAndKeys()
	if err != nil {
		return err
	}

	_, err = m.transport.InitFilters(filters, publicKeys)
	return err
}

func (m *Messenger) collectFiltersAndKeys() ([]transport.FiltersToInitialize, []*ecdsa.PublicKey, error) {
	var wg sync.WaitGroup
	errCh := make(chan error, 5)
	filtersCh := make(chan []transport.FiltersToInitialize, 3)
	publicKeysCh := make(chan []*ecdsa.PublicKey, 2)

	wg.Add(5)
	go m.processJoinedCommunities(&wg, filtersCh, errCh)
	go m.processSpectatedCommunities(&wg, filtersCh, errCh)
	go m.processChats(&wg, filtersCh, publicKeysCh, errCh)
	go m.processContacts(&wg, publicKeysCh, errCh)
	go m.processControlledCommunities(&wg, errCh)

	wg.Wait()
	close(filtersCh)
	close(publicKeysCh)
	close(errCh)

	return m.collectResults(filtersCh, publicKeysCh, errCh)
}

func (m *Messenger) processJoinedCommunities(wg *sync.WaitGroup, filtersCh chan<- []transport.FiltersToInitialize, errCh chan<- error) {
	defer gocommon.LogOnPanic()
	defer wg.Done()

	joinedCommunities, err := m.communitiesManager.Joined()
	if err != nil {
		errCh <- err
		return
	}

	filtersToInit := m.processCommunitiesSettings(joinedCommunities)
	filtersCh <- filtersToInit
}

func (m *Messenger) processCommunitiesSettings(communities []*communities.Community) []transport.FiltersToInitialize {
	logger := m.logger.With(zap.String("site", "processCommunitiesSettings"))
	filtersToInit := make([]transport.FiltersToInitialize, 0, len(communities))

	for _, org := range communities {
		// the org advertise on the public topic derived by the pk
		filtersToInit = append(filtersToInit, m.DefaultFilters(org)...)

		if err := m.communitiesManager.EnsureCommunitySettings(org); err != nil {
			logger.Warn("failed to process community settings", zap.Error(err))
		}
	}

	return filtersToInit
}

func (m *Messenger) processSpectatedCommunities(wg *sync.WaitGroup, filtersCh chan<- []transport.FiltersToInitialize, errCh chan<- error) {
	defer gocommon.LogOnPanic()
	defer wg.Done()

	spectatedCommunities, err := m.communitiesManager.Spectated()
	if err != nil {
		errCh <- err
		return
	}

	filtersToInit := make([]transport.FiltersToInitialize, 0, len(spectatedCommunities))
	for _, org := range spectatedCommunities {
		filtersToInit = append(filtersToInit, m.DefaultFilters(org)...)
	}
	filtersCh <- filtersToInit
}

func (m *Messenger) processChats(wg *sync.WaitGroup, filtersCh chan<- []transport.FiltersToInitialize, publicKeysCh chan<- []*ecdsa.PublicKey, errCh chan<- error) {
	defer gocommon.LogOnPanic()
	defer wg.Done()

	// Get chat IDs and public keys from the existing chats.
	// TODO: Get only active chats by the query.
	chats, err := m.persistence.Chats()
	if err != nil {
		errCh <- err
		return
	}

	validChats := m.validateChats(chats)
	communitiesCache := make(map[string]*communities.Community)
	m.initChatsFirstMessageTimestamp(communitiesCache, validChats)

	filters, publicKeys, err := m.processValidChats(validChats, communitiesCache)
	if err != nil {
		errCh <- err
		return
	}

	filtersCh <- filters
	publicKeysCh <- publicKeys

	if err := m.processDeprecatedChats(); err != nil {
		errCh <- err
	}
}

func (m *Messenger) validateChats(chats []*Chat) []*Chat {
	logger := m.logger.With(zap.String("site", "validateChats"))
	var validChats []*Chat

	for _, chat := range chats {
		if err := chat.Validate(); err != nil {
			logger.Warn("failed to validate chat", zap.Error(err))
			continue
		}
		validChats = append(validChats, chat)
	}

	return validChats
}

func (m *Messenger) processValidChats(validChats []*Chat, communityInfo map[string]*communities.Community) ([]transport.FiltersToInitialize, []*ecdsa.PublicKey, error) {
	var filtersToInit []transport.FiltersToInitialize
	var publicKeys []*ecdsa.PublicKey

	for _, chat := range validChats {
		if !chat.Active || chat.Timeline() {
			m.allChats.Store(chat.ID, chat)
			continue
		}

		filters, pks, err := m.processSingleChat(chat, communityInfo)
		if err != nil {
			return nil, nil, err
		}

		filtersToInit = append(filtersToInit, filters...)
		publicKeys = append(publicKeys, pks...)
		m.allChats.Store(chat.ID, chat)
	}

	return filtersToInit, publicKeys, nil
}

func (m *Messenger) processSingleChat(chat *Chat, communityInfo map[string]*communities.Community) ([]transport.FiltersToInitialize, []*ecdsa.PublicKey, error) {
	var filters []transport.FiltersToInitialize
	var publicKeys []*ecdsa.PublicKey

	switch chat.ChatType {
	case ChatTypePublic, ChatTypeProfile:
		filters = append(filters, transport.FiltersToInitialize{ChatID: chat.ID})

	case ChatTypeCommunityChat:
		filter, err := m.processCommunityChat(chat, communityInfo)
		if err != nil {
			return nil, nil, err
		}
		filters = append(filters, filter)

	case ChatTypeOneToOne:
		pk, err := chat.PublicKey()
		if err != nil {
			return nil, nil, err
		}
		publicKeys = append(publicKeys, pk)

	case ChatTypePrivateGroupChat:
		pks, err := m.processPrivateGroupChat(chat)
		if err != nil {
			return nil, nil, err
		}
		publicKeys = append(publicKeys, pks...)

	default:
		return nil, nil, errors.New("invalid chat type")
	}

	return filters, publicKeys, nil
}

func (m *Messenger) processCommunityChat(chat *Chat, communityInfo map[string]*communities.Community) (transport.FiltersToInitialize, error) {
	community, ok := communityInfo[chat.CommunityID]
	if !ok {
		var err error
		community, err = m.communitiesManager.GetByIDString(chat.CommunityID)
		if err != nil {
			return transport.FiltersToInitialize{}, err
		}
		communityInfo[chat.CommunityID] = community
	}

	if chat.UnviewedMessagesCount > 0 || chat.UnviewedMentionsCount > 0 {
		// Make sure the unread count is 0 for the channels the user cannot view
		// It's possible that the users received messages to a channel before permissions were added
		if !community.CanView(&m.identity.PublicKey, chat.CommunityChatID()) {
			chat.UnviewedMessagesCount = 0
			chat.UnviewedMentionsCount = 0
		}
	}

	return transport.FiltersToInitialize{
		ChatID:      chat.ID,
		PubsubTopic: community.PubsubTopic(),
	}, nil
}

func (m *Messenger) processPrivateGroupChat(chat *Chat) ([]*ecdsa.PublicKey, error) {
	var publicKeys []*ecdsa.PublicKey
	for _, member := range chat.Members {
		publicKey, err := member.PublicKey()
		if err != nil {
			return nil, errors.Wrapf(err, "invalid public key for member %s in chat %s", member.ID, chat.Name)
		}
		publicKeys = append(publicKeys, publicKey)
	}
	return publicKeys, nil
}

func (m *Messenger) processDeprecatedChats() error {
	// Timeline and profile chats are deprecated.
	// This code can be removed after some reasonable time.

	// upsert timeline chat
	if !deprecation.ChatProfileDeprecated {
		if err := m.ensureTimelineChat(); err != nil {
			return err
		}
	}

	// upsert profile chat
	if !deprecation.ChatTimelineDeprecated {
		if err := m.ensureMyOwnProfileChat(); err != nil {
			return err
		}
	}

	return nil
}

func (m *Messenger) processContacts(wg *sync.WaitGroup, publicKeysCh chan<- []*ecdsa.PublicKey, errCh chan<- error) {
	defer gocommon.LogOnPanic()
	defer wg.Done()

	// Get chat IDs and public keys from the contacts.
	contacts, err := m.persistence.Contacts()
	if err != nil {
		errCh <- err
		return
	}

	var publicKeys []*ecdsa.PublicKey
	for idx, contact := range contacts {
		if err = m.updateContactImagesURL(contact); err != nil {
			errCh <- err
			return
		}
		m.allContacts.Store(contact.ID, contacts[idx])
		// We only need filters for contacts added by us and not blocked.
		if !contact.added() || contact.Blocked {
			continue
		}

		publicKey, err := contact.PublicKey()
		if err != nil {
			m.logger.Error("failed to get contact's public key", zap.Error(err))
			continue
		}
		publicKeys = append(publicKeys, publicKey)
	}
	publicKeysCh <- publicKeys
}

// processControlledCommunities Init filters for the communities we control
func (m *Messenger) processControlledCommunities(wg *sync.WaitGroup, errCh chan<- error) {
	defer gocommon.LogOnPanic()
	defer wg.Done()

	controlledCommunities, err := m.communitiesManager.Controlled()
	if err != nil {
		errCh <- err
		return
	}

	var communityFiltersToInitialize []transport.CommunityFilterToInitialize
	for _, c := range controlledCommunities {
		communityFiltersToInitialize = append(communityFiltersToInitialize, transport.CommunityFilterToInitialize{
			Shard:   c.Shard(),
			PrivKey: c.PrivateKey(),
		})
	}

	_, err = m.InitCommunityFilters(communityFiltersToInitialize)
	if err != nil {
		errCh <- err
	}
}

func (m *Messenger) collectResults(filtersCh <-chan []transport.FiltersToInitialize, publicKeysCh <-chan []*ecdsa.PublicKey, errCh <-chan error) ([]transport.FiltersToInitialize, []*ecdsa.PublicKey, error) {
	var errs []error
	for err := range errCh {
		m.logger.Error("error collecting filters and public keys", zap.Error(err))
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return nil, nil, stderrors.Join(errs...)
	}

	var allFilters []transport.FiltersToInitialize
	var allPublicKeys []*ecdsa.PublicKey

	for filters := range filtersCh {
		allFilters = append(allFilters, filters...)
	}

	for pks := range publicKeysCh {
		allPublicKeys = append(allPublicKeys, pks...)
	}

	return allFilters, allPublicKeys, nil
}
