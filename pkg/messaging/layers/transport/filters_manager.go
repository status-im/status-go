package transport

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	wakuv2 "github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

type RawFilter struct {
	FilterID string
	Topic    types.TopicType
	SymKeyID string
}

type FiltersService interface {
	AddKeyPair(key *ecdsa.PrivateKey) (string, error)
	DeleteKeyPair(keyID string) bool

	AddSymKeyDirect(key []byte) (string, error)
	AddSymKeyFromPassword(password string) (string, error)
	GetSymKey(id string) ([]byte, error)
	DeleteSymKey(id string) bool

	Subscribe(pubsubTopic string, contentTopics [][]byte) (string, error)
	Unsubscribe(ctx context.Context, id string) error
	UnsubscribeMany(ids []string) error
}

type FiltersManager struct {
	service     FiltersService
	persistence KeysPersistence
	privateKey  *ecdsa.PrivateKey
	keys        map[string][]byte // a cache of symmetric manager derived from passwords
	logger      *zap.Logger
	mutex       sync.Mutex
	filters     map[string]*Filter
	// asymKeys maps an asymmetric filter's FilterID to the private key used to
	// decode messages received on it. Symmetric keys are resolved on demand via
	// keysManager.RawSymKey(SymKeyID) (the same path the send side uses), so only
	// the asymmetric keys — which are not otherwise reachable from a persisted
	// Filter — are held here. Runtime-only; never persisted.
	asymKeys map[string]*ecdsa.PrivateKey
}

// NewFiltersManager returns a new filtersManager.
func NewFiltersManager(persistence KeysPersistence, service FiltersService, privateKey *ecdsa.PrivateKey, logger *zap.Logger) (*FiltersManager, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	keys, err := persistence.All()
	if err != nil {
		return nil, err
	}

	return &FiltersManager{
		privateKey:  privateKey,
		service:     service,
		persistence: persistence,
		keys:        keys,
		filters:     make(map[string]*Filter),
		asymKeys:    make(map[string]*ecdsa.PrivateKey),
		logger:      logger.With(zap.Namespace("filtersManager")),
	}, nil
}

// WatchersByTopic returns the filters interested in the given (pubsubTopic,
// contentTopic) pair. A filter with no explicit pubsub topic listens on the
// default shard, so it is normalized before comparison — preserving the
// (pubsub, content) routing the waku layer used. A single content topic can be
// shared by distinct chats (content topics are a 4-byte hash of the chat id),
// so the caller must still confirm the match by decoding with each candidate's
// own key.
func (f *FiltersManager) WatchersByTopic(pubsubTopic, contentTopic string) []*Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	var res []*Filter
	for _, filter := range f.filters {
		if filter.ContentTopic.ContentTopic() != contentTopic {
			continue
		}
		filterPubsubTopic := filter.PubsubTopic
		if filterPubsubTopic == "" {
			filterPubsubTopic = wakuv2.DefaultShardPubsubTopic()
		}
		if filterPubsubTopic == pubsubTopic {
			res = append(res, filter)
		}
	}
	return res
}

// AsymKey returns the asymmetric private key used to decode messages received
// on the filter with the given FilterID, if it is an asymmetric filter.
func (f *FiltersManager) AsymKey(filterID string) (*ecdsa.PrivateKey, bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	k, ok := f.asymKeys[filterID]
	return k, ok
}

func (f *FiltersManager) Init(
	filtersToInit []FiltersToInitialize,
	publicKeys []*ecdsa.PublicKey,
) ([]*Filter, error) {

	// Load our contact code.
	_, err := f.LoadContactCode(&f.privateKey.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load contact code")
	}

	// Load partitioned topic.
	_, err = f.loadMyPartitioned()
	if err != nil {
		return nil, err
	}

	// Add discovery topic.
	_, err = f.LoadDiscovery()
	if err != nil {
		return nil, err
	}

	// Add public, one-to-one and negotiated filters.
	for _, fi := range filtersToInit {
		_, err := f.LoadPublic(fi.ChatID, fi.PubsubTopic)
		if err != nil {
			return nil, err
		}
	}

	for _, publicKey := range publicKeys {
		_, err := f.LoadContactCode(publicKey)
		if err != nil {
			return nil, err
		}
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	var allFilters []*Filter
	for _, f := range f.filters {
		allFilters = append(allFilters, f)
	}
	return allFilters, nil
}

type FiltersToInitialize struct {
	ChatID      string
	PubsubTopic string
}

func (f *FiltersManager) InitPublicFilters(publicFiltersToInit []FiltersToInitialize) ([]*Filter, error) {
	var filters []*Filter
	// Add public, one-to-one and negotiated filters.
	for _, pf := range publicFiltersToInit {
		f, err := f.LoadPublic(pf.ChatID, pf.PubsubTopic)
		if err != nil {
			return nil, err
		}
		filters = append(filters, f)
	}
	return filters, nil
}

type CommunityFilterToInitialize struct {
	PrivKey *ecdsa.PrivateKey
}

func (f *FiltersManager) InitCommunityFilters(communityFiltersToInitialize []CommunityFilterToInitialize) ([]*Filter, error) {
	var filters []*Filter
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, communityFilter := range communityFiltersToInitialize {
		// to satisfy gosec: C601 checks
		cf := communityFilter
		if cf.PrivKey == nil {
			continue
		}

		topics := make([]string, 0)
		topics = append(topics, wakuv2.DefaultShardPubsubTopic())
		topics = append(topics, wakuv2.DefaultNonProtectedPubsubTopic())

		for _, pubsubTopic := range topics {
			pk := &cf.PrivKey.PublicKey
			identityStr := PublicKeyToStr(pk)
			rawFilter, err := f.addAsymmetric(identityStr, pubsubTopic, cf.PrivKey)
			if err != nil {
				f.logger.Debug("could not register community filter", zap.Error(err))
				return nil, err

			}
			filterID := identityStr + "-admin" + pubsubTopic
			filter := &Filter{
				ChatID:       filterID,
				FilterID:     rawFilter.FilterID,
				PubsubTopic:  pubsubTopic,
				ContentTopic: rawFilter.Topic,
				Identity:     identityStr,
				Listen:       true,
				OneToOne:     true,
			}

			f.filters[filterID] = filter

			f.logger.Debug("registering filter for", zap.String("chatID", filterID), zap.String("type", "community"), zap.String("pubsubTopic", pubsubTopic), zap.String("contentTopic", rawFilter.Topic.String()))

			filters = append(filters, filter)
		}
	}
	return filters, nil
}

func (f *FiltersManager) Reset(ctx context.Context) error {
	var filters []*Filter

	f.mutex.Lock()
	for _, f := range f.filters {
		filters = append(filters, f)
	}
	f.mutex.Unlock()

	return f.Remove(ctx, filters...)
}

func (f *FiltersManager) Filters() (result []*Filter) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, f := range f.filters {
		result = append(result, f)
	}

	return
}

func (f *FiltersManager) Filter(chatID string) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.filters[chatID]
}

// FilterByFilterID returns a Filter with a given Whisper filter ID.
func (f *FiltersManager) FilterByFilterID(filterID string) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	for _, f := range f.filters {
		if f.FilterID == filterID {
			return f
		}
	}
	return nil
}

func (f *FiltersManager) FilterByTopic(topic []byte) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	for _, f := range f.filters {
		if bytes.Equal(types.TopicTypeToByteArray(f.ContentTopic), topic) {
			return f
		}
	}
	return nil
}

// FiltersByIdentities returns an array of filters for given list of public keys
func (f *FiltersManager) FiltersByIdentities(identities []string) []*Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	identitiesMap := make(map[string]bool)

	for _, identity := range identities {
		identitiesMap[identity] = true
	}

	var filters []*Filter

	for _, filter := range f.filters {
		// Pre-pend 0x before comparing
		if identitiesMap["0x"+filter.Identity] {
			filters = append(filters, filter)
		}
	}
	return filters
}

// FilterByChatID returns a Filter for given chat id
func (f *FiltersManager) FilterByChatID(chatID string) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.filters[chatID]
}

// Remove removes all the filtersToRemove
func (f *FiltersManager) Remove(ctx context.Context, filtersToRemove ...*Filter) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	for _, filter := range filtersToRemove {
		if err := f.service.Unsubscribe(ctx, filter.FilterID); err != nil {
			return err
		}
		if filter.SymKeyID != "" {
			f.service.DeleteSymKey(filter.SymKeyID)
		}
		delete(f.asymKeys, filter.FilterID)
		for k, v := range f.filters {
			if filter.FilterID == v.FilterID {
				delete(f.filters, k)
			}
		}
	}

	return nil
}

// Remove remove all the filters associated with a chat/identity
func (f *FiltersManager) RemoveNoListenFilters() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	var filterIDs []string
	var filters []*Filter

	for _, v := range f.filters {
		if !v.Listen {
			filterIDs = append(filterIDs, v.FilterID)
			filters = append(filters, v)
		}
	}
	if err := f.service.UnsubscribeMany(filterIDs); err != nil {
		return err
	}

	for _, filter := range filters {
		if filter.SymKeyID != "" {
			f.service.DeleteSymKey(filter.SymKeyID)
		}
		delete(f.asymKeys, filter.FilterID)
		for k, v := range f.filters {
			if filter.FilterID == v.FilterID {
				delete(f.filters, k)
			}
		}
	}

	return nil
}

// RemoveFilterByChatID removes the filters associated with a chat/identity
func (f *FiltersManager) RemoveFilterByChatID(chatID string) (*Filter, error) {
	// TODO: remove subscriptions from waku2 if required. Might need to be implemented in transport

	toRemove := make([]*Filter, 0)
	f.mutex.Lock()
	for _, filter := range f.filters {
		if filter.ChatID == chatID {
			toRemove = append(toRemove, filter)
		}
	}
	f.mutex.Unlock()

	if len(toRemove) == 0 {
		return nil, nil
	}

	err := f.Remove(context.Background(), toRemove...)
	if err != nil {
		return nil, err
	}

	// TODO temporary so not changing the return type, otherwise we should return a slice
	return toRemove[0], nil
}

// LoadPartitioned creates a filter for a partitioned topic.
func (f *FiltersManager) LoadPartitioned(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen bool) (*Filter, error) {
	return f.loadPartitioned(publicKey, identity, listen, false)
}

// LoadEphemeral creates a filter for a partitioned/personal topic.
func (f *FiltersManager) LoadEphemeral(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen bool) (*Filter, error) {
	return f.loadPartitioned(publicKey, identity, listen, true)
}

// LoadPersonal creates a filter for a personal topic.
func (f *FiltersManager) LoadPersonal(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen bool) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	chatID := PersonalDiscoveryTopic(publicKey)
	if _, ok := f.filters[chatID]; ok {
		return f.filters[chatID], nil
	}

	// We set up a filter so we can publish,
	// but we discard envelopes if listen is false.
	filter, err := f.addAsymmetric(chatID, "", identity)
	if err != nil {
		f.logger.Debug("could not register personal topic filter", zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		FilterID:     filter.FilterID,
		ContentTopic: filter.Topic,
		Identity:     PublicKeyToStr(publicKey),
		Listen:       listen,
		OneToOne:     true,
	}

	f.filters[chatID] = chat

	f.logger.Debug("registering filter for", zap.String("chatID", chatID), zap.String("type", "personal"), zap.String("topic", filter.Topic.String()))

	return chat, nil

}

func (f *FiltersManager) loadMyPartitioned() (*Filter, error) {
	return f.loadPartitioned(&f.privateKey.PublicKey, f.privateKey, true, false)
}

func (f *FiltersManager) loadPartitioned(publicKey *ecdsa.PublicKey, identity *ecdsa.PrivateKey, listen, ephemeral bool) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	chatID := PartitionedTopic(publicKey)
	if _, ok := f.filters[chatID]; ok {
		return f.filters[chatID], nil
	}

	// We set up a filter so we can publish,
	// but we discard envelopes if listen is false.
	filter, err := f.addAsymmetric(chatID, "", identity)
	if err != nil {
		f.logger.Debug("could not register partitioned topic", zap.String("chatID", chatID), zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		FilterID:     filter.FilterID,
		ContentTopic: filter.Topic,
		Identity:     PublicKeyToStr(publicKey),
		Listen:       listen,
		Ephemeral:    ephemeral,
		OneToOne:     true,
	}

	f.filters[chatID] = chat

	f.logger.Debug("registering filter for", zap.String("chatID", chatID), zap.String("type", "partitioned"), zap.String("topic", filter.Topic.String()))

	return chat, nil
}

// LoadNegotiated loads a negotiated secret as a filter.
func (f *FiltersManager) LoadNegotiated(secret messagingtypes.NegotiatedSecret) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	chatID := NegotiatedTopic(secret.PublicKey)

	if _, ok := f.filters[chatID]; ok {
		return f.filters[chatID], nil
	}

	keyString := hex.EncodeToString(secret.Key)
	filter, err := f.addSymmetric(keyString, "")
	if err != nil {
		f.logger.Debug("could not register negotiated topic", zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		ContentTopic: filter.Topic,
		SymKeyID:     filter.SymKeyID,
		FilterID:     filter.FilterID,
		Identity:     PublicKeyToStr(secret.PublicKey),
		Negotiated:   true,
		Listen:       true,
		OneToOne:     true,
	}

	f.filters[chat.ChatID] = chat

	f.logger.Debug("registering filter for", zap.String("chatID", chatID), zap.String("type", "negotiated"), zap.String("topic", filter.Topic.String()))

	return chat, nil
}

// LoadDiscovery adds 1 discovery filter
// for the personal discovery topic.
func (f *FiltersManager) LoadDiscovery() ([]*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	personalDiscoveryTopic := PersonalDiscoveryTopic(&f.privateKey.PublicKey)

	// Check if filters are already loaded.
	var result []*Filter

	expectedTopicCount := 1

	if chat, ok := f.filters[personalDiscoveryTopic]; ok {
		result = append(result, chat)
	}

	if len(result) == expectedTopicCount {
		return result, nil
	}

	identityStr := PublicKeyToStr(&f.privateKey.PublicKey)

	// Load personal discovery
	personalDiscoveryChat := &Filter{
		ChatID:    personalDiscoveryTopic,
		Identity:  identityStr,
		Discovery: true,
		Listen:    true,
		OneToOne:  true,
	}

	discoveryResponse, err := f.addAsymmetric(personalDiscoveryChat.ChatID, personalDiscoveryChat.PubsubTopic, f.privateKey)
	if err != nil {
		f.logger.Debug("could not register discovery topic", zap.String("chatID", personalDiscoveryChat.ChatID), zap.Error(err))
		return nil, err
	}

	personalDiscoveryChat.ContentTopic = discoveryResponse.Topic
	personalDiscoveryChat.FilterID = discoveryResponse.FilterID

	f.filters[personalDiscoveryChat.ChatID] = personalDiscoveryChat

	f.logger.Debug("registering filter for", zap.String("chatID", personalDiscoveryChat.ChatID), zap.String("type", "discovery"), zap.String("topic", personalDiscoveryChat.ContentTopic.String()))

	return []*Filter{personalDiscoveryChat}, nil
}

func (f *FiltersManager) PersonalTopicFilter() *Filter {
	personalDiscoveryTopic := PersonalDiscoveryTopic(&f.privateKey.PublicKey)

	return f.filters[personalDiscoveryTopic]
}

// LoadPublic adds a filter for a public chat with specific pubsubTopic
func (f *FiltersManager) LoadPublic(chatID string, pubsubTopic string) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if chat, ok := f.filters[chatID]; ok {
		if chat.PubsubTopic != pubsubTopic {
			f.logger.Debug("updating pubsub topic for filter",
				zap.String("chatID", chatID),
				zap.String("type", "public"),
				zap.String("oldTopic", chat.PubsubTopic),
				zap.String("newTopic", pubsubTopic),
			)
			chat.PubsubTopic = pubsubTopic
			f.filters[chatID] = chat
		}
		return chat, nil
	}

	filterAndTopic, err := f.addSymmetric(chatID, pubsubTopic)
	if err != nil {
		f.logger.Debug("could not register public chat topic", zap.String("chatID", chatID), zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		FilterID:     filterAndTopic.FilterID,
		SymKeyID:     filterAndTopic.SymKeyID,
		ContentTopic: filterAndTopic.Topic,
		PubsubTopic:  pubsubTopic,
		Listen:       true,
		OneToOne:     false,
	}

	f.filters[chatID] = chat

	f.logger.Debug("registering filter for",
		zap.String("chatID", chatID),
		zap.String("type", "public"),
		zap.String("ContentTopic", filterAndTopic.Topic.String()),
		zap.String("PubsubTopic", pubsubTopic),
	)

	return chat, nil
}

// LoadContactCode creates a filter for the advertise topic for a given public key.
func (f *FiltersManager) LoadContactCode(pubKey *ecdsa.PublicKey) (*Filter, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	chatID := ContactCodeTopic(pubKey)

	if _, ok := f.filters[chatID]; ok {
		return f.filters[chatID], nil
	}

	contactCodeFilter, err := f.addSymmetric(chatID, "")
	if err != nil {
		f.logger.Debug("could not register contact code topic", zap.String("chatID", chatID), zap.Error(err))
		return nil, err
	}

	chat := &Filter{
		ChatID:       chatID,
		FilterID:     contactCodeFilter.FilterID,
		ContentTopic: contactCodeFilter.Topic,
		SymKeyID:     contactCodeFilter.SymKeyID,
		Identity:     PublicKeyToStr(pubKey),
		Listen:       true,
	}

	f.filters[chatID] = chat

	f.logger.Debug("registering filter for", zap.String("chatID", chatID), zap.String("type", "contact-code"), zap.String("topic", contactCodeFilter.Topic.String()))

	return chat, nil
}

// addSymmetric adds a symmetric key filter
func (f *FiltersManager) addSymmetric(chatID string, pubsubTopic string) (*RawFilter, error) {
	var symKeyID string
	var err error

	topic := ToTopic(chatID)
	topics := [][]byte{topic}

	symKey, ok := f.keys[chatID]
	if ok {
		symKeyID, err = f.service.AddSymKeyDirect(symKey)
		if err != nil {
			return nil, err
		}
	} else {
		symKeyID, err = f.service.AddSymKeyFromPassword(chatID)
		if err != nil {
			return nil, err
		}
		if symKey, err = f.service.GetSymKey(symKeyID); err != nil {
			return nil, err
		}
		f.keys[chatID] = symKey

		err = f.persistence.Add(chatID, symKey)
		if err != nil {
			return nil, err
		}
	}

	id, err := f.service.Subscribe(pubsubTopic, topics)
	if err != nil {
		return nil, err
	}

	return &RawFilter{
		FilterID: id,
		SymKeyID: symKeyID,
		Topic:    types.BytesToTopic(topic),
	}, nil
}

// addAsymmetricFilter adds a filter with our private key. Whether the filter
// actually receives messages is governed by its Listen flag: non-listening
// filters are skipped during routing in handleReceivedMessage.
func (f *FiltersManager) addAsymmetric(chatID string, pubsubTopic string, identity *ecdsa.PrivateKey) (*RawFilter, error) {
	topic := ToTopic(chatID)
	topics := [][]byte{topic}

	if _, err := f.service.AddKeyPair(identity); err != nil {
		return nil, err
	}

	id, err := f.service.Subscribe(pubsubTopic, topics)
	if err != nil {
		return nil, err
	}

	f.asymKeys[id] = identity

	return &RawFilter{FilterID: id, Topic: types.BytesToTopic(topic)}, nil
}

// GetNegotiated returns a negotiated chat given an identity
func (f *FiltersManager) GetNegotiated(identity *ecdsa.PublicKey) *Filter {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.filters[NegotiatedTopic(identity)]
}
