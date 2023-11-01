package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"golang.org/x/time/rate"

	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/meirf/gopart"

	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/communities/token"
	"github.com/status-im/status-go/protocol/discord"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/services/wallet/bigint"
	"github.com/status-im/status-go/signal"
	"github.com/status-im/status-go/transactions"
)

// 7 days interval
var messageArchiveInterval = 7 * 24 * time.Hour

// 1 day interval
var updateActiveMembersInterval = 24 * time.Hour

const discordTimestampLayout = "2006-01-02T15:04:05+00:00"

const (
	importSlowRate          = time.Second / 1
	importFastRate          = time.Second / 100
	importMessagesChunkSize = 10
	importInitialDelay      = time.Minute * 5
)

const (
	maxChunkSizeMessages = 1000
	maxChunkSizeBytes    = 1500000
)

func (m *Messenger) publishOrg(org *communities.Community) error {
	if org == nil {
		return nil
	}

	m.logger.Debug("publishing org", zap.String("org-id", org.IDString()), zap.Any("org", org))
	payload, err := org.MarshaledDescription()
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload: payload,
		Sender:  org.PrivateKey(),
		// we don't want to wrap in an encryption layer message
		SkipProtocolLayer: true,
		MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION,
		PubsubTopic:       org.PubsubTopic(), // TODO: confirm if it should be sent in community pubsub topic
	}
	_, err = m.sender.SendPublic(context.Background(), org.IDString(), rawMessage)
	return err
}

func (m *Messenger) publishCommunityEvents(community *communities.Community, msg *communities.CommunityEventsMessage) error {
	m.logger.Debug("publishing community events", zap.String("admin-id", common.PubkeyToHex(&m.identity.PublicKey)), zap.Any("event", msg))

	payload, err := msg.Marshal()
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload: payload,
		Sender:  m.identity,
		// we don't want to wrap in an encryption layer message
		SkipProtocolLayer: true,
		MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_EVENTS_MESSAGE,
		PubsubTopic:       community.PubsubTopic(), // TODO: confirm if it should be sent in community pubsub topic
	}

	// TODO: resend in case of failure?
	_, err = m.sender.SendPublic(context.Background(), types.EncodeHex(msg.CommunityID), rawMessage)
	return err
}

func (m *Messenger) publishCommunityEventsRejected(community *communities.Community, msg *communities.CommunityEventsMessage) error {
	if !community.IsControlNode() {
		return communities.ErrNotControlNode
	}
	m.logger.Debug("publishing community events rejected", zap.Any("event", msg))

	communityEventsMessage := msg.ToProtobuf()
	communityEventsMessageRejected := &protobuf.CommunityEventsMessageRejected{
		Msg: communityEventsMessage,
	}

	payload, err := proto.Marshal(communityEventsMessageRejected)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload: payload,
		Sender:  community.PrivateKey(),
		// we don't want to wrap in an encryption layer message
		SkipProtocolLayer: true,
		MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_EVENTS_MESSAGE_REJECTED,
		PubsubTopic:       community.PubsubTopic(), // TODO: confirm if it should be sent in community pubsub topic
	}

	// TODO: resend in case of failure?
	_, err = m.sender.SendPublic(context.Background(), types.EncodeHex(msg.CommunityID), rawMessage)
	return err
}

func (m *Messenger) publishCommunityPrivilegedMemberSyncMessage(msg *communities.CommunityPrivilegedMemberSyncMessage) error {

	m.logger.Debug("publishing privileged user sync message", zap.Any("event", msg))

	payload, err := proto.Marshal(msg.CommunityPrivilegedUserSyncMessage)
	if err != nil {
		return err
	}

	rawMessage := &common.RawMessage{
		Payload:           payload,
		Sender:            msg.CommunityPrivateKey, // if empty, sender private key will be used in SendPrivate
		SkipProtocolLayer: true,
		MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_PRIVILEGED_USER_SYNC_MESSAGE,
	}

	for _, receivers := range msg.Receivers {
		_, err = m.sender.SendPrivate(context.Background(), receivers, rawMessage)
	}

	return err
}

func (m *Messenger) handleCommunitiesHistoryArchivesSubscription(c chan *communities.Subscription) {

	go func() {
		for {
			select {
			case sub, more := <-c:
				if !more {
					return
				}

				if sub.CreatingHistoryArchivesSignal != nil {
					m.config.messengerSignalsHandler.CreatingHistoryArchives(sub.CreatingHistoryArchivesSignal.CommunityID)
				}

				if sub.HistoryArchivesCreatedSignal != nil {
					m.config.messengerSignalsHandler.HistoryArchivesCreated(
						sub.HistoryArchivesCreatedSignal.CommunityID,
						sub.HistoryArchivesCreatedSignal.From,
						sub.HistoryArchivesCreatedSignal.To,
					)
				}

				if sub.NoHistoryArchivesCreatedSignal != nil {
					m.config.messengerSignalsHandler.NoHistoryArchivesCreated(
						sub.NoHistoryArchivesCreatedSignal.CommunityID,
						sub.NoHistoryArchivesCreatedSignal.From,
						sub.NoHistoryArchivesCreatedSignal.To,
					)
				}

				if sub.HistoryArchivesSeedingSignal != nil {

					m.config.messengerSignalsHandler.HistoryArchivesSeeding(sub.HistoryArchivesSeedingSignal.CommunityID)

					c, err := m.communitiesManager.GetByIDString(sub.HistoryArchivesSeedingSignal.CommunityID)
					if err != nil {
						m.logger.Debug("failed to retrieve community by id string", zap.Error(err))
					}

					if c.IsControlNode() {
						err := m.dispatchMagnetlinkMessage(sub.HistoryArchivesSeedingSignal.CommunityID)
						if err != nil {
							m.logger.Debug("failed to dispatch magnetlink message", zap.Error(err))
						}
					}
				}

				if sub.HistoryArchivesUnseededSignal != nil {
					m.config.messengerSignalsHandler.HistoryArchivesUnseeded(sub.HistoryArchivesUnseededSignal.CommunityID)
				}

				if sub.HistoryArchiveDownloadedSignal != nil {
					m.config.messengerSignalsHandler.HistoryArchiveDownloaded(
						sub.HistoryArchiveDownloadedSignal.CommunityID,
						sub.HistoryArchiveDownloadedSignal.From,
						sub.HistoryArchiveDownloadedSignal.To,
					)
				}

				if sub.DownloadingHistoryArchivesFinishedSignal != nil {
					m.config.messengerSignalsHandler.DownloadingHistoryArchivesFinished(sub.DownloadingHistoryArchivesFinishedSignal.CommunityID)
				}

				if sub.DownloadingHistoryArchivesStartedSignal != nil {
					m.config.messengerSignalsHandler.DownloadingHistoryArchivesStarted(sub.DownloadingHistoryArchivesStartedSignal.CommunityID)
				}

				if sub.ImportingHistoryArchiveMessagesSignal != nil {
					m.config.messengerSignalsHandler.ImportingHistoryArchiveMessages(sub.ImportingHistoryArchiveMessagesSignal.CommunityID)
				}

			case <-m.quit:
				return
			}
		}
	}()
}

// handleCommunitiesSubscription handles events from communities
func (m *Messenger) handleCommunitiesSubscription(c chan *communities.Subscription) {
	var lastPublished int64
	// We check every 5 minutes if we need to publish
	ticker := time.NewTicker(5 * time.Minute)

	recentlyPublishedOrgs := func() map[string]*communities.Community {
		result := make(map[string]*communities.Community)

		controlledCommunities, err := m.communitiesManager.Controlled()
		if err != nil {
			m.logger.Warn("failed to retrieve orgs", zap.Error(err))
			return result
		}

		for _, org := range controlledCommunities {
			result[org.IDString()] = org
		}

		return result
	}()

	publishOrgAndDistributeEncryptionKeys := func(community *communities.Community) {
		err := m.publishOrg(community)
		if err != nil {
			m.logger.Warn("failed to publish org", zap.Error(err))
			return
		}
		m.logger.Debug("published org")

		recentlyPublishedOrg := recentlyPublishedOrgs[community.IDString()]

		// signal client with published community
		if m.config.messengerSignalsHandler != nil {
			if recentlyPublishedOrg == nil || community.Clock() > recentlyPublishedOrg.Clock() {
				response := &MessengerResponse{}
				response.AddCommunity(community)
				m.config.messengerSignalsHandler.MessengerResponse(response)
			}
		}

		// evaluate and distribute encryption keys (if any)
		encryptionKeyActions := communities.EvaluateCommunityEncryptionKeyActions(recentlyPublishedOrg, community)
		err = m.communitiesKeyDistributor.Distribute(community, encryptionKeyActions)
		if err != nil {
			m.logger.Warn("failed to distribute encryption keys", zap.Error(err))
		}

		recentlyPublishedOrgs[community.IDString()] = community.CreateDeepCopy()
	}

	go func() {
		for {
			select {
			case sub, more := <-c:
				if !more {
					return
				}
				if sub.Community != nil {
					publishOrgAndDistributeEncryptionKeys(sub.Community)
				}

				if sub.CommunityEventsMessage != nil {
					err := m.publishCommunityEvents(sub.Community, sub.CommunityEventsMessage)
					if err != nil {
						m.logger.Warn("failed to publish community events", zap.Error(err))
					}
				}

				if sub.CommunityEventsMessageInvalidClock != nil {
					err := m.publishCommunityEventsRejected(sub.CommunityEventsMessageInvalidClock.Community,
						sub.CommunityEventsMessageInvalidClock.CommunityEventsMessage)
					if err != nil {
						m.logger.Warn("failed to publish community events rejected", zap.Error(err))
					}
				}

				if sub.AcceptedRequestsToJoin != nil {
					for _, requestID := range sub.AcceptedRequestsToJoin {
						accept := &requests.AcceptRequestToJoinCommunity{
							ID: requestID,
						}
						_, err := m.AcceptRequestToJoinCommunity(accept)
						if err != nil {
							m.logger.Warn("failed to accept request to join ", zap.Error(err))
						}
					}
				}

				if sub.RejectedRequestsToJoin != nil {
					for _, requestID := range sub.RejectedRequestsToJoin {
						reject := &requests.DeclineRequestToJoinCommunity{
							ID: requestID,
						}
						_, err := m.DeclineRequestToJoinCommunity(reject)
						if err != nil {
							m.logger.Warn("failed to decline request to join ", zap.Error(err))
						}
					}
				}

				if sub.CommunityPrivilegedMemberSyncMessage != nil {
					if err := m.publishCommunityPrivilegedMemberSyncMessage(sub.CommunityPrivilegedMemberSyncMessage); err != nil {
						m.logger.Warn("failed to publish community private members sync message", zap.Error(err))
					}
				}
				if sub.TokenCommunityValidated != nil {
					state := m.buildMessageState()
					communityResponse := sub.TokenCommunityValidated

					err := m.handleCommunityResponse(state, communityResponse)
					if err != nil {
						m.logger.Error("failed to handle community response", zap.Error(err))
					}

					m.processCommunityChanges(state)

					response, err := m.saveDataAndPrepareResponse(state)
					if err != nil {
						m.logger.Error("failed to save data and prepare response")
					}

					// control node changed and we were kicked out. It now awaits our addresses
					if communityResponse.Changes.ControlNodeChanged != nil && communityResponse.Changes.MemberKicked {
						requestToJoin, err := m.sendSharedAddressToControlNode(communityResponse.Community.ControlNode(), communityResponse.Community)
						if err != nil {
							m.logger.Error("share address to control node failed", zap.String("id", types.EncodeHex(communityResponse.Community.ID())), zap.Error(err))
						} else {
							state.Response.RequestsToJoinCommunity = append(state.Response.RequestsToJoinCommunity, requestToJoin)
						}
					}

					if m.config.messengerSignalsHandler != nil {
						m.config.messengerSignalsHandler.MessengerResponse(response)
					}
				}

			case <-ticker.C:
				// If we are not online, we don't even try
				if !m.online() {
					continue
				}

				// If not enough time has passed since last advertisement, we skip this
				if time.Now().Unix()-lastPublished < communityAdvertiseIntervalSecond {
					continue
				}

				controlledCommunities, err := m.communitiesManager.Controlled()
				if err != nil {
					m.logger.Warn("failed to retrieve orgs", zap.Error(err))
				}

				for idx := range controlledCommunities {
					org := controlledCommunities[idx]
					_, beingImported := m.importingCommunities[org.IDString()]
					if !beingImported {
						publishOrgAndDistributeEncryptionKeys(org)
					}
				}

				// set lastPublished
				lastPublished = time.Now().Unix()

			case <-m.quit:
				return

			}
		}
	}()
}

func (m *Messenger) updateCommunitiesActiveMembersPeriodically() {
	communitiesLastUpdated := make(map[string]int64)

	// We check every 5 minutes if we need to update
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-ticker.C:
				controlledCommunities, err := m.communitiesManager.Controlled()
				if err != nil {
					m.logger.Error("failed to update community active members count", zap.Error(err))
				}

				for _, community := range controlledCommunities {
					lastUpdated, ok := communitiesLastUpdated[community.IDString()]
					if !ok {
						lastUpdated = 0
					}

					// If not enough time has passed since last update, we skip this
					if time.Now().Unix()-lastUpdated < int64(updateActiveMembersInterval.Seconds()) {
						continue
					}

					if err := m.updateCommunityActiveMembers(community.IDString()); err == nil {
						communitiesLastUpdated[community.IDString()] = time.Now().Unix()

						// Perf: ensure `updateCommunityActiveMembers` is not called few times in a row
						// Next communities will be handled in subsequent ticks
						break
					} else {
						m.logger.Error("failed to update community active members count", zap.Error(err))
					}
				}

			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) CheckCommunitiesToUnmute() (*MessengerResponse, error) {
	m.logger.Debug("watching communities to unmute")
	response := &MessengerResponse{}
	communities, err := m.communitiesManager.All()
	if err != nil {
		return nil, fmt.Errorf("couldn't get all communities: %v", err)
	}
	for _, community := range communities {
		communityMuteTill, err := time.Parse(time.RFC3339, community.MuteTill().Format(time.RFC3339))
		if err != nil {
			return nil, err
		}
		currTime, err := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		if err != nil {
			return nil, err
		}

		if currTime.After(communityMuteTill) && !communityMuteTill.Equal(time.Time{}) && community.Muted() {
			err := m.communitiesManager.SetMuted(community.ID(), false)
			if err != nil {
				m.logger.Info("CheckCommunitiesToUnmute err", zap.Any("Couldn't unmute community", err))
				break
			}

			err = m.MuteCommunityTill(community.ID(), time.Time{})
			if err != nil {
				m.logger.Info("MuteCommunityTill err", zap.Any("Could not set mute community till", err))
				break
			}

			unmutedCommunity, err := m.communitiesManager.GetByID(community.ID())
			if err != nil {
				return nil, err
			}
			response.AddCommunity(unmutedCommunity)

		}

	}

	return response, nil
}

func (m *Messenger) updateCommunityActiveMembers(communityID string) error {
	lastWeek := time.Now().AddDate(0, 0, -7).Unix()
	count, err := m.persistence.CountActiveChattersInCommunity(communityID, lastWeek)
	if err != nil {
		return err
	}

	if err = m.communitiesManager.SetCommunityActiveMembersCount(communityID, uint64(count)); err != nil {
		return err
	}

	m.logger.Debug("community active members updated", zap.String("communityID", communityID), zap.Uint("count", count))
	return nil
}

func (m *Messenger) Communities() ([]*communities.Community, error) {
	return m.communitiesManager.All()
}

func (m *Messenger) ControlledCommunities() ([]*communities.Community, error) {
	return m.communitiesManager.Controlled()
}

func (m *Messenger) JoinedCommunities() ([]*communities.Community, error) {
	return m.communitiesManager.Joined()
}

func (m *Messenger) SpectatedCommunities() ([]*communities.Community, error) {
	return m.communitiesManager.Spectated()
}

// Regularly gets list of curated communities and signals them to client
func (m *Messenger) startCuratedCommunitiesUpdateLoop() {
	logger := m.logger.Named("startCuratedCommunitiesUpdateLoop")

	const errorTimeout = 10 * time.Second
	const successTimeout = 120 * time.Second
	// TODO lower this back again once the real curated community contract is up
	// The current contract contains communities that are no longer accessible on waku
	const unknownCommunitiesFoundTimeout = 60 * time.Second

	type curatedCommunities struct {
		ContractCommunities         []string
		ContractFeaturedCommunities []string
		UnknownCommunities          []string
	}

	var mu = sync.RWMutex{}
	var c = curatedCommunities{}

	go func() {
		for {
			var timeTillNextUpdate time.Duration

			response, err := m.CuratedCommunities()

			if err != nil {
				timeTillNextUpdate = errorTimeout
			} else {
				mu.Lock()
				// Check if it's the same values we had
				if !reflect.DeepEqual(c.ContractCommunities, response.ContractCommunities) ||
					!reflect.DeepEqual(c.ContractFeaturedCommunities, response.ContractFeaturedCommunities) ||
					!reflect.DeepEqual(c.UnknownCommunities, response.UnknownCommunities) {
					// One of the communities is different, send the updated response
					m.config.messengerSignalsHandler.SendCuratedCommunitiesUpdate(response)

					// Update the values
					c.ContractCommunities = response.ContractCommunities
					c.ContractFeaturedCommunities = response.ContractFeaturedCommunities
					c.UnknownCommunities = response.UnknownCommunities
				}
				mu.Unlock()

				if len(response.UnknownCommunities) == 0 {
					//next update shouldn't happen soon
					timeTillNextUpdate = successTimeout
				} else {
					//unknown communities already requested from mailserver, so we wait just a bit before
					//next attempt to get their info
					timeTillNextUpdate = unknownCommunitiesFoundTimeout
				}
			}

			logger.Debug("Next curated communities update will happen in", zap.Duration("timeTillNextUpdate", timeTillNextUpdate))

			select {
			case <-time.After(timeTillNextUpdate):
			case <-m.quit:
				return
			}
		}
	}()
}

func (m *Messenger) CuratedCommunities() (*communities.KnownCommunitiesResponse, error) {
	if m.contractMaker == nil {
		m.logger.Warn("contract maker not initialized")
		return nil, errors.New("contract maker not initialized")
	}

	// Revert code to https://github.com/status-im/status-go/blob/e6a3f63ec7f2fa691878ed35f921413dc8acfc66/protocol/messenger_communities.go#L211-L226 once the curated communities contract is deployed to mainnet

	chainID := uint64(420) // Optimism Goerli

	directory, err := m.contractMaker.NewDirectory(chainID)
	if err != nil {
		return nil, err
	}
	// --- end delete

	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}

	curatedCommunities, err := directory.GetCommunities(callOpts)
	if err != nil {
		return nil, err
	}
	var communityIDs []types.HexBytes
	for _, c := range curatedCommunities {
		communityIDs = append(communityIDs, c)
	}

	response, err := m.communitiesManager.GetStoredDescriptionForCommunities(communityIDs)
	if err != nil {
		return nil, err
	}

	featuredCommunities, err := directory.GetFeaturedCommunities(callOpts)
	if err != nil {
		return nil, err
	}

	// TODO: The curated communities smart contract should also store the community shard cluster and index

	for _, c := range featuredCommunities {
		response.ContractFeaturedCommunities = append(response.ContractFeaturedCommunities, types.HexBytes(c).String())
		// TODO: use CommunityShard instead of communityID
		/*response.ContractFeaturedCommunities = append(response.ContractFeaturedCommunities, communities.CommunityShard{
			CommunityID: types.HexBytes(c).String(),
			// Shard:     c.Shard, // TODO: obtain this value
		})*/
	}

	// TODO: this loop is added just to not revert the change from requestCommunitiesFromMailserver
	// Once support for shards is added in the contract, just pass the `response.UnknownCommunities` directly to
	// the function

	var unknownCommunities []communities.CommunityShard
	for _, u := range response.UnknownCommunities {
		unknownCommunities = append(unknownCommunities, communities.CommunityShard{
			CommunityID: u,
		})
	}

	go m.requestCommunitiesFromMailserver(unknownCommunities)

	return response, nil
}

func (m *Messenger) initCommunityChats(community *communities.Community) ([]*Chat, error) {
	logger := m.logger.Named("initCommunityChats")

	publicFiltersToInit := community.DefaultFilters()

	chats := CreateCommunityChats(community, m.getTimesource())

	for _, chat := range chats {
		publicFiltersToInit = append(publicFiltersToInit, transport.FiltersToInitialize{ChatID: chat.ID, PubsubTopic: community.PubsubTopic()})

	}

	// Load transport filters
	filters, err := m.transport.InitPublicFilters(publicFiltersToInit)
	if err != nil {
		logger.Debug("m.transport.InitPublicFilters error", zap.Error(err))
		return nil, err
	}

	if community.IsControlNode() {
		// Init the community filter so we can receive messages on the community
		communityFilters, err := m.transport.InitCommunityFilters([]transport.CommunityFilterToInitialize{{
			Shard:   community.Shard().TransportShard(),
			PrivKey: community.PrivateKey(),
		}})

		if err != nil {
			return nil, err
		}
		filters = append(filters, communityFilters...)
	}

	willSync, err := m.scheduleSyncFilters(filters)
	if err != nil {
		logger.Debug("m.scheduleSyncFilters error", zap.Error(err))
		return nil, err
	}

	if !willSync {
		defaultSyncPeriod, err := m.settings.GetDefaultSyncPeriod()
		if err != nil {
			logger.Debug("m.settings.GetDefaultSyncPeriod error", zap.Error(err))
			return nil, err
		}

		timestamp := uint32(m.getTimesource().GetCurrentTime()/1000) - defaultSyncPeriod
		for idx := range chats {
			chats[idx].SyncedTo = timestamp
			chats[idx].SyncedFrom = timestamp
		}
	}

	if err = m.saveChats(chats); err != nil {
		logger.Debug("m.saveChats error", zap.Error(err))
		return nil, err
	}

	return chats, nil
}

func (m *Messenger) initCommunitySettings(communityID types.HexBytes) (*communities.CommunitySettings, error) {
	communitySettings, err := m.communitiesManager.GetCommunitySettingsByID(communityID)
	if err != nil {
		return nil, err
	}
	if communitySettings != nil {
		return communitySettings, nil
	}

	communitySettings = &communities.CommunitySettings{
		CommunityID:                  communityID.String(),
		HistoryArchiveSupportEnabled: true,
	}

	if err := m.communitiesManager.SaveCommunitySettings(*communitySettings); err != nil {
		return nil, err
	}

	return communitySettings, nil
}

func (m *Messenger) JoinCommunity(ctx context.Context, communityID types.HexBytes, forceJoin bool) (*MessengerResponse, error) {
	mr, err := m.joinCommunity(ctx, communityID, forceJoin)
	if err != nil {
		return nil, err
	}

	if com, ok := mr.communities[communityID.String()]; ok {
		err = m.syncCommunity(context.Background(), com, m.dispatchMessage)
		if err != nil {
			return nil, err
		}
	}

	return mr, nil
}

func (m *Messenger) subscribeToCommunityShard(communityID []byte, shard *common.Shard) error {
	if m.transport.WakuVersion() != 2 {
		return nil
	}

	// TODO: this should probably be moved completely to transport once pubsub topic logic is implemented
	pubsubTopic := transport.GetPubsubTopic(shard.TransportShard())

	privK, err := m.transport.RetrievePubsubTopicKey(pubsubTopic)
	if err != nil {
		return err
	}

	var pubK *ecdsa.PublicKey
	if privK != nil {
		pubK = &privK.PublicKey
	}

	return m.transport.SubscribeToPubsubTopic(pubsubTopic, pubK)
}

func (m *Messenger) joinCommunity(ctx context.Context, communityID types.HexBytes, forceJoin bool) (*MessengerResponse, error) {
	logger := m.logger.Named("joinCommunity")

	response := &MessengerResponse{}

	community, err := m.communitiesManager.JoinCommunity(communityID, forceJoin)
	if err != nil {
		logger.Debug("m.communitiesManager.JoinCommunity error", zap.Error(err))
		return nil, err
	}

	// chats and settings are already initialized for spectated communities
	if !community.Spectated() {
		chats, err := m.initCommunityChats(community)
		if err != nil {
			return nil, err
		}
		response.AddChats(chats)

		if _, err = m.initCommunitySettings(communityID); err != nil {
			return nil, err
		}

		if err = m.subscribeToCommunityShard(community.ID(), community.Shard()); err != nil {
			return nil, err
		}
	}

	communitySettings, err := m.communitiesManager.GetCommunitySettingsByID(communityID)
	if err != nil {
		return nil, err
	}

	response.AddCommunity(community)
	response.AddCommunitySettings(communitySettings)

	if err = m.reregisterForPushNotifications(); err != nil {
		return nil, err
	}

	if err = m.sendCurrentUserStatusToCommunity(ctx, community); err != nil {
		logger.Debug("m.sendCurrentUserStatusToCommunity error", zap.Error(err))
		return nil, err
	}

	if err = m.PublishIdentityImage(); err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) SpectateCommunity(communityID types.HexBytes) (*MessengerResponse, error) {
	logger := m.logger.Named("SpectateCommunity")

	response := &MessengerResponse{}

	community, err := m.communitiesManager.SpectateCommunity(communityID)
	if err != nil {
		logger.Debug("SpectateCommunity error", zap.Error(err))
		return nil, err
	}

	chats, err := m.initCommunityChats(community)
	if err != nil {
		return nil, err
	}
	response.AddChats(chats)

	settings, err := m.initCommunitySettings(communityID)
	if err != nil {
		return nil, err
	}
	response.AddCommunitySettings(settings)

	response.AddCommunity(community)

	if err = m.subscribeToCommunityShard(community.ID(), community.Shard()); err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) MuteDuration(mutedType requests.MutingVariation) (time.Time, error) {
	var MuteTill time.Time

	switch mutedType {
	case MuteTill1Min:
		MuteTill = time.Now().Add(MuteFor1MinDuration)
	case MuteFor15Min:
		MuteTill = time.Now().Add(MuteFor15MinsDuration)
	case MuteFor1Hr:
		MuteTill = time.Now().Add(MuteFor1HrsDuration)
	case MuteFor8Hr:
		MuteTill = time.Now().Add(MuteFor8HrsDuration)
	case MuteFor1Week:
		MuteTill = time.Now().Add(MuteFor1WeekDuration)
	default:
		MuteTill = time.Time{}
	}

	muteTillTimeRemoveMs, err := time.Parse(time.RFC3339, MuteTill.Format(time.RFC3339))
	if err != nil {
		return time.Time{}, err
	}

	return muteTillTimeRemoveMs, nil
}

func (m *Messenger) SetMuted(request *requests.MuteCommunity) error {
	if err := request.Validate(); err != nil {
		return err
	}

	if request.MutedType == Unmuted {
		return m.communitiesManager.SetMuted(request.CommunityID, false)
	}

	return m.communitiesManager.SetMuted(request.CommunityID, true)
}

func (m *Messenger) MuteCommunityTill(communityID []byte, muteTill time.Time) error {
	return m.communitiesManager.MuteCommunityTill(communityID, muteTill)
}

func (m *Messenger) MuteAllCommunityChats(request *requests.MuteCommunity) (time.Time, error) {
	return m.UpdateMuteCommunityStatus(request.CommunityID.String(), true, request.MutedType)
}

func (m *Messenger) UnMuteAllCommunityChats(communityID string) (time.Time, error) {
	return m.UpdateMuteCommunityStatus(communityID, false, Unmuted)
}

func (m *Messenger) UpdateMuteCommunityStatus(communityID string, muted bool, mutedType requests.MutingVariation) (time.Time, error) {
	community, err := m.communitiesManager.GetByIDString(communityID)
	if err != nil {
		return time.Time{}, err
	}

	request := &requests.MuteCommunity{
		CommunityID: community.ID(),
		MutedType:   mutedType,
	}

	err = m.SetMuted(request)
	if err != nil {
		return time.Time{}, err
	}

	muteTill, err := m.MuteDuration(mutedType)
	if err != nil {
		return time.Time{}, err
	}

	err = m.MuteCommunityTill(community.ID(), muteTill)

	for _, chatID := range community.CommunityChatsIDs() {
		if muted {
			_, err := m.MuteChat(&requests.MuteChat{ChatID: communityID + chatID, MutedType: mutedType})
			if err != nil {
				return time.Time{}, err
			}

		} else {
			err = m.UnmuteChat(communityID + chatID)
			if err != nil {
				return time.Time{}, err
			}

		}

		if err != nil {
			return time.Time{}, err
		}
	}

	return muteTill, err
}

func (m *Messenger) SetMutePropertyOnChatsByCategory(request *requests.MuteCategory, muted bool) error {
	if err := request.Validate(); err != nil {
		return err
	}
	community, err := m.communitiesManager.GetByIDString(request.CommunityID)
	if err != nil {
		return err
	}

	for _, chatID := range community.ChatsByCategoryID(request.CategoryID) {
		if muted {
			_, err = m.MuteChat(&requests.MuteChat{ChatID: request.CommunityID + chatID, MutedType: request.MutedType})
		} else {
			err = m.UnmuteChat(request.CommunityID + chatID)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// getAccountsToShare is used to get the wallet accounts to share either when requesting to join a community or when editing
// requestToJoinID can be empty when editing
func (m *Messenger) getAccountsToShare(addressesToReveal []string, airdropAddress string, communityID types.HexBytes, password string, requestToJoinID []byte) (map[gethcommon.Address]*protobuf.RevealedAccount, []gethcommon.Address, error) {
	walletAccounts, err := m.settings.GetActiveAccounts()
	if err != nil {
		return nil, nil, err
	}

	revealedAccounts := make(map[gethcommon.Address]*protobuf.RevealedAccount)
	revealedAddresses := make([]gethcommon.Address, 0)

	containsAddress := func(addresses []string, targetAddress string) bool {
		for _, address := range addresses {
			if types.HexToAddress(address) == types.HexToAddress(targetAddress) {
				return true
			}
		}
		return false
	}

	for i, walletAccount := range walletAccounts {
		if walletAccount.Chat || walletAccount.Type == accounts.AccountTypeWatch {
			continue
		}

		if len(addressesToReveal) > 0 && !containsAddress(addressesToReveal, walletAccount.Address.Hex()) {
			continue
		}

		verifiedAccount, err := m.accountsManager.GetVerifiedWalletAccount(m.settings, walletAccount.Address.Hex(), password)
		if err != nil {
			return nil, nil, err
		}

		messageToSign := types.EncodeHex(crypto.Keccak256(m.IdentityPublicKeyCompressed(), communityID, requestToJoinID))
		signParams := account.SignParams{
			Data:     messageToSign,
			Address:  verifiedAccount.Address.Hex(),
			Password: password,
		}
		signatureBytes, err := m.accountsManager.Sign(signParams, verifiedAccount)
		if err != nil {
			return nil, nil, err
		}

		revealedAddress := gethcommon.HexToAddress(verifiedAccount.Address.Hex())
		revealedAddresses = append(revealedAddresses, revealedAddress)
		address := verifiedAccount.Address.Hex()
		isAirdropAddress := types.HexToAddress(address) == types.HexToAddress(airdropAddress)
		if airdropAddress == "" {
			isAirdropAddress = i == 0
		}
		revealedAccounts[revealedAddress] = &protobuf.RevealedAccount{
			Address:          address,
			IsAirdropAddress: isAirdropAddress,
			Signature:        signatureBytes,
			ChainIds:         make([]uint64, 0),
		}
	}
	return revealedAccounts, revealedAddresses, nil
}

func (m *Messenger) RequestToJoinCommunity(request *requests.RequestToJoinCommunity) (*MessengerResponse, error) {
	logger := m.logger.Named("RequestToJoinCommunity")
	if err := request.Validate(); err != nil {
		logger.Debug("request failed to validate", zap.Error(err), zap.Any("request", request))
		return nil, err
	}

	if request.Password != "" {
		walletAccounts, err := m.settings.GetActiveAccounts()
		if err != nil {
			return nil, err
		}
		if len(walletAccounts) > 0 {
			_, err := m.accountsManager.GetVerifiedWalletAccount(m.settings, walletAccounts[0].Address.Hex(), request.Password)
			if err != nil {
				return nil, err
			}
		}
	}

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return nil, err
	}

	community, requestToJoin, err := m.communitiesManager.CreateRequestToJoin(&m.identity.PublicKey, request)
	if err != nil {
		return nil, err
	}

	requestToJoinProto := &protobuf.CommunityRequestToJoin{
		Clock:            requestToJoin.Clock,
		EnsName:          requestToJoin.ENSName,
		DisplayName:      displayName,
		CommunityId:      community.ID(),
		RevealedAccounts: make([]*protobuf.RevealedAccount, 0),
	}

	// find wallet accounts and attach wallet addresses and
	// signatures to request
	if request.Password != "" {
		revealedAccounts, revealedAddresses, err := m.getAccountsToShare(
			request.AddressesToReveal,
			request.AirdropAddress,
			request.CommunityID,
			request.Password,
			requestToJoin.ID,
		)
		if err != nil {
			return nil, err
		}

		response, err := m.communitiesManager.CheckPermissionToJoin(community.ID(), revealedAddresses)
		if err != nil {
			return nil, err
		}
		if !response.Satisfied {
			return nil, errors.New("permission to join not satisfied")
		}

		for _, accountAndChainIDs := range response.ValidCombinations {
			revealedAccounts[accountAndChainIDs.Address].ChainIds = accountAndChainIDs.ChainIDs
		}

		for _, revealedAccount := range revealedAccounts {
			requestToJoinProto.RevealedAccounts = append(requestToJoinProto.RevealedAccounts, revealedAccount)
			requestToJoin.RevealedAccounts = append(requestToJoinProto.RevealedAccounts, revealedAccount)
		}
	}

	community, _, err = m.communitiesManager.SaveRequestToJoinAndCommunity(requestToJoin, community)
	if err != nil {
		return nil, err
	}
	err = m.syncCommunity(context.Background(), community, m.dispatchMessage)
	if err != nil {
		return nil, err
	}

	payload, err := proto.Marshal(requestToJoinProto)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		Payload:           payload,
		CommunityID:       community.ID(),
		SkipProtocolLayer: true,
		MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN,
		PubsubTopic:       common.DefaultNonProtectedPubsubTopic(community.Shard()),
	}

	_, err = m.sender.SendCommunityMessage(context.Background(), rawMessage)
	if err != nil {
		return nil, err
	}

	// send request to join to privileged members
	if !community.AutoAccept() {
		privilegedMembers := community.GetFilteredPrivilegedMembers(map[string]struct{}{})

		for _, member := range privilegedMembers[protobuf.CommunityMember_ROLE_OWNER] {
			_, err := m.sender.SendPrivate(context.Background(), member, &rawMessage)
			if err != nil {
				return nil, err
			}
		}
		for _, member := range privilegedMembers[protobuf.CommunityMember_ROLE_TOKEN_MASTER] {
			_, err := m.sender.SendPrivate(context.Background(), member, &rawMessage)
			if err != nil {
				return nil, err
			}
		}

		// don't send revealed addresses to admins
		requestToJoinProto.RevealedAccounts = make([]*protobuf.RevealedAccount, 0)
		payload, err = proto.Marshal(requestToJoinProto)
		if err != nil {
			return nil, err
		}
		rawMessage.Payload = payload

		for _, member := range privilegedMembers[protobuf.CommunityMember_ROLE_ADMIN] {
			_, err := m.sender.SendPrivate(context.Background(), member, &rawMessage)
			if err != nil {
				return nil, err
			}
		}
	}

	response := &MessengerResponse{RequestsToJoinCommunity: []*communities.RequestToJoin{requestToJoin}}
	response.AddCommunity(community)

	// We send a push notification in the background
	go func() {
		if m.pushNotificationClient != nil {
			pks, err := community.CanManageUsersPublicKeys()
			if err != nil {
				m.logger.Error("failed to get pks", zap.Error(err))
				return
			}
			for _, publicKey := range pks {
				pkString := common.PubkeyToHex(publicKey)
				_, err = m.pushNotificationClient.SendNotification(publicKey, nil, requestToJoin.ID, pkString, protobuf.PushNotification_REQUEST_TO_JOIN_COMMUNITY)
				if err != nil {
					m.logger.Error("error sending notification", zap.Error(err))
					return
				}
			}
		}
	}()

	// Activity center notification
	notification := &ActivityCenterNotification{
		ID:               types.FromHex(requestToJoin.ID.String()),
		Type:             ActivityCenterNotificationTypeCommunityRequest,
		Timestamp:        m.getTimesource().GetCurrentTime(),
		CommunityID:      community.IDString(),
		MembershipStatus: ActivityCenterMembershipStatusPending,
		Read:             true,
		Deleted:          false,
		UpdatedAt:        m.GetCurrentTimeInMillis(),
	}

	err = m.addActivityCenterNotification(response, notification, nil)
	if err != nil {
		m.logger.Error("failed to save notification", zap.Error(err))
		return nil, err
	}

	return response, nil
}

func (m *Messenger) EditSharedAddressesForCommunity(request *requests.EditSharedAddresses) (*MessengerResponse, error) {
	logger := m.logger.Named("EditSharedAddressesForCommunity")
	if err := request.Validate(); err != nil {
		logger.Debug("request failed to validate", zap.Error(err), zap.Any("request", request))
		return nil, err
	}
	// verify wallet password is there
	if request.Password == "" {
		return nil, errors.New("password is necessary to use this function")
	}

	community, err := m.communitiesManager.GetByID(request.CommunityID)
	if err != nil {
		return nil, err
	}

	walletAccounts, err := m.settings.GetActiveAccounts()
	if err != nil {
		return nil, err
	}
	if len(walletAccounts) > 0 {
		_, err := m.accountsManager.GetVerifiedWalletAccount(m.settings, walletAccounts[0].Address.Hex(), request.Password)
		if err != nil {
			return nil, err
		}
	}

	if !community.HasMember(m.IdentityPublicKey()) {
		return nil, errors.New("not part of the community")
	}

	member := community.GetMember(m.IdentityPublicKey())

	requestToEditRevealedAccountsProto := &protobuf.CommunityEditSharedAddresses{
		Clock:            member.LastUpdateClock + 1,
		CommunityId:      community.ID(),
		RevealedAccounts: make([]*protobuf.RevealedAccount, 0),
	}
	// find wallet accounts and attach wallet addresses and
	// signatures to request
	revealedAccounts, revealedAddresses, err := m.getAccountsToShare(
		request.AddressesToReveal,
		request.AirdropAddress,
		request.CommunityID,
		request.Password,
		[]byte{},
	)
	if err != nil {
		return nil, err
	}

	checkPermissionResponse, err := m.communitiesManager.CheckPermissionToJoin(community.ID(), revealedAddresses)
	if err != nil {
		return nil, err
	}

	for _, accountAndChainIDs := range checkPermissionResponse.ValidCombinations {
		revealedAccounts[accountAndChainIDs.Address].ChainIds = accountAndChainIDs.ChainIDs
	}

	for _, revealedAccount := range revealedAccounts {
		requestToEditRevealedAccountsProto.RevealedAccounts = append(requestToEditRevealedAccountsProto.RevealedAccounts, revealedAccount)
	}

	requestID := communities.CalculateRequestID(common.PubkeyToHex(&m.identity.PublicKey), request.CommunityID)
	err = m.communitiesManager.RemoveRequestToJoinRevealedAddresses(requestID)
	if err != nil {
		return nil, err
	}
	err = m.communitiesManager.SaveRequestToJoinRevealedAddresses(requestID, requestToEditRevealedAccountsProto.RevealedAccounts)
	if err != nil {
		return nil, err
	}

	payload, err := proto.Marshal(requestToEditRevealedAccountsProto)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		Payload:           payload,
		CommunityID:       community.ID(),
		SkipProtocolLayer: true,
		MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_EDIT_SHARED_ADDRESSES,
		PubsubTopic:       community.PubsubTopic(), // TODO: confirm if it should be sent in community pubsub topic
	}

	_, err = m.sender.SendCommunityMessage(context.Background(), rawMessage)
	if err != nil {
		return nil, err
	}

	// send edit message also to TokenMasters and Owners
	skipMembers := make(map[string]struct{})
	skipMembers[common.PubkeyToHex(&m.identity.PublicKey)] = struct{}{}

	privilegedMembers := community.GetFilteredPrivilegedMembers(skipMembers)
	for role, members := range privilegedMembers {
		if len(members) == 0 || (role != protobuf.CommunityMember_ROLE_TOKEN_MASTER && role != protobuf.CommunityMember_ROLE_OWNER) {
			continue
		}
		for _, member := range members {
			_, err := m.sender.SendPrivate(context.Background(), member, &rawMessage)
			if err != nil {
				return nil, err
			}
		}
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)

	return response, nil
}

func (m *Messenger) GetRevealedAccounts(communityID types.HexBytes, memberPk string) ([]*protobuf.RevealedAccount, error) {
	return m.communitiesManager.GetRevealedAddresses(communityID, memberPk)
}

func (m *Messenger) GetRevealedAccountsForAllMembers(communityID types.HexBytes) (map[string][]*protobuf.RevealedAccount, error) {
	community, err := m.communitiesManager.GetByID(communityID)
	if err != nil {
		return nil, err
	}
	membersRevealedAccounts := map[string][]*protobuf.RevealedAccount{}
	for _, memberPubKey := range community.GetMemberPubkeys() {
		memberPubKeyStr := common.PubkeyToHex(memberPubKey)
		accounts, err := m.communitiesManager.GetRevealedAddresses(communityID, memberPubKeyStr)
		if err != nil {
			return nil, err
		}
		membersRevealedAccounts[memberPubKeyStr] = accounts
	}
	return membersRevealedAccounts, nil
}

func (m *Messenger) CreateCommunityCategory(request *requests.CreateCommunityCategory) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var response MessengerResponse
	_, changes, err := m.communitiesManager.CreateCategory(request, true)
	if err != nil {
		return nil, err
	}
	response.AddCommunity(changes.Community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	return &response, nil
}

func (m *Messenger) EditCommunityCategory(request *requests.EditCommunityCategory) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var response MessengerResponse
	community, changes, err := m.communitiesManager.EditCategory(request)
	if err != nil {
		return nil, err
	}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	return &response, nil
}

func (m *Messenger) ReorderCommunityCategories(request *requests.ReorderCommunityCategories) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var response MessengerResponse
	community, changes, err := m.communitiesManager.ReorderCategories(request)
	if err != nil {
		return nil, err
	}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	return &response, nil
}

func (m *Messenger) ReorderCommunityChat(request *requests.ReorderCommunityChat) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var response MessengerResponse
	community, changes, err := m.communitiesManager.ReorderChat(request)
	if err != nil {
		return nil, err
	}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	return &response, nil
}

func (m *Messenger) DeleteCommunityCategory(request *requests.DeleteCommunityCategory) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var response MessengerResponse
	community, changes, err := m.communitiesManager.DeleteCategory(request)
	if err != nil {
		return nil, err
	}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	return &response, nil
}

func (m *Messenger) CancelRequestToJoinCommunity(ctx context.Context, request *requests.CancelRequestToJoinCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	requestToJoin, community, err := m.communitiesManager.CancelRequestToJoin(request)
	if err != nil {
		return nil, err
	}

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return nil, err
	}

	cancelRequestToJoinProto := &protobuf.CommunityCancelRequestToJoin{
		Clock:       community.Clock(),
		EnsName:     requestToJoin.ENSName,
		DisplayName: displayName,
		CommunityId: community.ID(),
	}

	payload, err := proto.Marshal(cancelRequestToJoinProto)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		Payload:           payload,
		CommunityID:       community.ID(),
		SkipProtocolLayer: true,
		MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_CANCEL_REQUEST_TO_JOIN,
		PubsubTopic:       common.DefaultNonProtectedPubsubTopic(community.Shard()),
	}
	_, err = m.sender.SendCommunityMessage(context.Background(), rawMessage)

	if err != nil {
		return nil, err
	}

	if !community.AutoAccept() {
		// send cancelation to community admins also
		rawMessage.Payload = payload

		privilegedMembers := community.GetPrivilegedMembers()
		for _, privilegedMember := range privilegedMembers {
			_, err := m.sender.SendPrivate(context.Background(), privilegedMember, &rawMessage)
			if err != nil {
				return nil, err
			}
		}
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	response.RequestsToJoinCommunity = append(response.RequestsToJoinCommunity, requestToJoin)

	// delete activity center notification
	notification, err := m.persistence.GetActivityCenterNotificationByID(requestToJoin.ID)
	if err != nil {
		return nil, err
	}

	if notification != nil {
		updatedAt := m.GetCurrentTimeInMillis()
		notification.UpdatedAt = updatedAt
		err = m.persistence.DeleteActivityCenterNotificationByID(types.FromHex(requestToJoin.ID.String()), notification.UpdatedAt)
		if err != nil {
			m.logger.Error("failed to delete notification from Activity Center", zap.Error(err))
			return nil, err
		}

		// set notification as deleted, so that the client will remove the activity center notification from UI
		notification.Deleted = true
		err = m.syncActivityCenterDeletedByIDs(ctx, []types.HexBytes{notification.ID}, updatedAt)
		if err != nil {
			m.logger.Error("CancelRequestToJoinCommunity, failed to sync activity center notification as deleted", zap.Error(err))
			return nil, err
		}
		response.AddActivityCenterNotification(notification)
	}

	return response, nil
}

func (m *Messenger) acceptRequestToJoinCommunity(requestToJoin *communities.RequestToJoin) (*MessengerResponse, error) {
	community, err := m.communitiesManager.AcceptRequestToJoin(requestToJoin)
	if err != nil {
		return nil, err
	}

	if community.IsControlNode() {
		// If we are the control node, we send the response to the user
		pk, err := common.HexToPubkey(requestToJoin.PublicKey)
		if err != nil {
			return nil, err
		}

		grant, err := community.BuildGrant(pk, "")
		if err != nil {
			return nil, err
		}

		var key *ecdsa.PrivateKey
		if m.transport.WakuVersion() == 2 {
			key, err = m.transport.RetrievePubsubTopicKey(community.PubsubTopic())
			if err != nil {
				return nil, err
			}
		}

		requestToJoinResponseProto := &protobuf.CommunityRequestToJoinResponse{
			Clock:                    community.Clock(),
			Accepted:                 true,
			CommunityId:              community.ID(),
			Community:                community.Description(),
			Grant:                    grant,
			ProtectedTopicPrivateKey: crypto.FromECDSA(key),
			Shard:                    community.Shard().Protobuffer(),
		}

		if m.torrentClientReady() && m.communitiesManager.TorrentFileExists(community.IDString()) {
			magnetlink, err := m.communitiesManager.GetHistoryArchiveMagnetlink(community.ID())
			if err != nil {
				m.logger.Warn("couldn't get magnet link for community", zap.Error(err))
				return nil, err
			}
			requestToJoinResponseProto.MagnetUri = magnetlink
		}

		payload, err := proto.Marshal(requestToJoinResponseProto)
		if err != nil {
			return nil, err
		}

		rawMessage := &common.RawMessage{
			Payload:           payload,
			Sender:            community.PrivateKey(),
			SkipProtocolLayer: true,
			MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN_RESPONSE,
			PubsubTopic:       common.DefaultNonProtectedPubsubTopic(community.Shard()),
		}

		_, err = m.sender.SendPrivate(context.Background(), pk, rawMessage)
		if err != nil {
			return nil, err
		}
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	response.AddRequestToJoinCommunity(requestToJoin)

	// Update existing notification
	notification, err := m.persistence.GetActivityCenterNotificationByID(requestToJoin.ID)
	if err != nil {
		return nil, err
	}
	if notification != nil {
		notification.MembershipStatus = ActivityCenterMembershipStatusAccepted
		if community.HasPermissionToSendCommunityEvents() {
			notification.MembershipStatus = ActivityCenterMembershipStatusAcceptedPending
		}
		notification.Read = true
		notification.Accepted = true
		notification.UpdatedAt = m.GetCurrentTimeInMillis()

		err = m.addActivityCenterNotification(response, notification, m.syncActivityCenterCommunityRequestDecisionAdapter)
		if err != nil {
			m.logger.Error("failed to save notification", zap.Error(err))
			return nil, err
		}
	}

	return response, nil
}

func (m *Messenger) AcceptRequestToJoinCommunity(request *requests.AcceptRequestToJoinCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	requestToJoin, err := m.communitiesManager.GetRequestToJoin(request.ID)
	if err != nil {
		return nil, err
	}

	return m.acceptRequestToJoinCommunity(requestToJoin)
}

func (m *Messenger) declineRequestToJoinCommunity(requestToJoin *communities.RequestToJoin) (*MessengerResponse, error) {
	community, err := m.communitiesManager.DeclineRequestToJoin(requestToJoin)
	if err != nil {
		return nil, err
	}

	if community.IsControlNode() {
		// Notify privileged members that request to join was rejected
		// Send request to join without revealed addresses
		requestToJoin.RevealedAccounts = make([]*protobuf.RevealedAccount, 0)
		declinedRequestsToJoin := make(map[string]*protobuf.CommunityRequestToJoin)
		declinedRequestsToJoin[requestToJoin.PublicKey] = requestToJoin.ToCommunityRequestToJoinProtobuf()

		syncMsg := &protobuf.CommunityPrivilegedUserSyncMessage{
			Type:          protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_REJECT_REQUEST_TO_JOIN,
			CommunityId:   community.ID(),
			RequestToJoin: declinedRequestsToJoin,
		}

		payloadSyncMsg, err := proto.Marshal(syncMsg)
		if err != nil {
			return nil, err
		}

		rawSyncMessage := &common.RawMessage{
			Payload:           payloadSyncMsg,
			Sender:            community.PrivateKey(),
			SkipProtocolLayer: true,
			MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_PRIVILEGED_USER_SYNC_MESSAGE,
		}

		privilegedMembers := community.GetPrivilegedMembers()
		for _, privilegedMember := range privilegedMembers {
			if privilegedMember.Equal(&m.identity.PublicKey) {
				continue
			}
			_, err := m.sender.SendPrivate(context.Background(), privilegedMember, rawSyncMessage)
			if err != nil {
				return nil, err
			}
		}
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	response.AddRequestToJoinCommunity(requestToJoin)

	// Update existing notification
	notification, err := m.persistence.GetActivityCenterNotificationByID(requestToJoin.ID)
	if err != nil {
		return nil, err
	}
	if notification != nil {
		notification.MembershipStatus = ActivityCenterMembershipStatusDeclined
		if community.HasPermissionToSendCommunityEvents() {
			notification.MembershipStatus = ActivityCenterMembershipStatusDeclinedPending
		}
		notification.Read = true
		notification.Dismissed = true
		notification.UpdatedAt = m.GetCurrentTimeInMillis()

		err = m.addActivityCenterNotification(response, notification, m.syncActivityCenterCommunityRequestDecisionAdapter)
		if err != nil {
			m.logger.Error("failed to save notification", zap.Error(err))
			return nil, err
		}
	}

	return response, nil
}

func (m *Messenger) DeclineRequestToJoinCommunity(request *requests.DeclineRequestToJoinCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	requestToJoin, err := m.communitiesManager.GetRequestToJoin(request.ID)
	if err != nil {
		return nil, err
	}

	return m.declineRequestToJoinCommunity(requestToJoin)
}

func (m *Messenger) LeaveCommunity(communityID types.HexBytes) (*MessengerResponse, error) {
	_, err := m.persistence.DismissAllActivityCenterNotificationsFromCommunity(communityID.String(), m.GetCurrentTimeInMillis())
	if err != nil {
		return nil, err
	}

	mr, err := m.leaveCommunity(communityID)
	if err != nil {
		return nil, err
	}

	community, ok := mr.communities[communityID.String()]
	if !ok {
		return nil, communities.ErrOrgNotFound
	}

	err = m.communitiesManager.DeleteCommunitySettings(communityID)
	if err != nil {
		return nil, err
	}

	m.communitiesManager.StopHistoryArchiveTasksInterval(communityID)

	err = m.syncCommunity(context.Background(), community, m.dispatchMessage)
	if err != nil {
		return nil, err
	}

	if !community.IsControlNode() {
		requestToLeaveProto := &protobuf.CommunityRequestToLeave{
			Clock:       uint64(time.Now().Unix()),
			CommunityId: communityID,
		}

		payload, err := proto.Marshal(requestToLeaveProto)
		if err != nil {
			return nil, err
		}

		community, err := m.communitiesManager.GetByID(communityID)
		if err != nil {
			return nil, err
		}

		rawMessage := common.RawMessage{
			Payload:           payload,
			CommunityID:       communityID,
			SkipProtocolLayer: true,
			MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_LEAVE,
			PubsubTopic:       community.PubsubTopic(), // TODO: confirm if it should be sent in the community pubsub topic
		}
		_, err = m.sender.SendCommunityMessage(context.Background(), rawMessage)
		if err != nil {
			return nil, err
		}
	}

	return mr, nil
}

func (m *Messenger) leaveCommunity(communityID types.HexBytes) (*MessengerResponse, error) {
	response := &MessengerResponse{}

	community, err := m.communitiesManager.LeaveCommunity(communityID)
	if err != nil {
		return nil, err
	}

	// Make chat inactive
	for chatID := range community.Chats() {
		communityChatID := communityID.String() + chatID
		response.AddRemovedChat(communityChatID)

		_, err = m.deactivateChat(communityChatID, 0, false, false)
		if err != nil {
			return nil, err
		}
		_, err = m.transport.RemoveFilterByChatID(communityChatID)
		if err != nil {
			return nil, err
		}
	}

	_, err = m.transport.RemoveFilterByChatID(communityID.String())
	if err != nil {
		return nil, err
	}

	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) kickedOutOfCommunity(communityID types.HexBytes) (*MessengerResponse, error) {
	response := &MessengerResponse{}

	community, err := m.communitiesManager.KickedOutOfCommunity(communityID)
	if err != nil {
		return nil, err
	}

	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) CheckAndDeletePendingRequestToJoinCommunity(ctx context.Context, sendResponse bool) (*MessengerResponse, error) {
	sendSignal := false

	pendingRequestsToJoin, err := m.communitiesManager.PendingRequestsToJoin()
	if err != nil {
		m.logger.Error("failed to fetch pending request to join", zap.Error(err))
		return nil, err
	}

	if len(pendingRequestsToJoin) == 0 {
		return nil, nil
	}

	response := &MessengerResponse{}
	timeNow := uint64(time.Now().Unix())

	for _, requestToJoin := range pendingRequestsToJoin {
		requestTimeOutClock, err := communities.AddTimeoutToRequestToJoinClock(requestToJoin.Clock)
		if err != nil {
			return nil, err
		}

		if timeNow >= requestTimeOutClock {
			err := m.communitiesManager.DeletePendingRequestToJoin(requestToJoin)
			if err != nil {
				m.logger.Error("failed to delete pending request to join", zap.String("req-id", requestToJoin.ID.String()), zap.Error(err))
				return nil, err
			}

			requestToJoin.Deleted = true
			response.AddRequestToJoinCommunity(requestToJoin)

			notification, err := m.persistence.GetActivityCenterNotificationByID(requestToJoin.ID)
			if err != nil {
				m.logger.Error("failed to fetch pending request to join", zap.Error(err))
				return nil, err
			}

			if notification != nil {
				// Delete activity centre notification for community admin
				if notification.Type == ActivityCenterNotificationTypeCommunityMembershipRequest {
					response2, err := m.MarkActivityCenterNotificationsDeleted(ctx, []types.HexBytes{notification.ID}, m.GetCurrentTimeInMillis(), true)
					if err != nil {
						m.logger.Error("[CheckAndDeletePendingRequestToJoinCommunity] failed to mark notification as deleted", zap.Error(err))
						return nil, err
					}
					response.AddActivityCenterNotifications(response2.ActivityCenterNotifications())
					response.SetActivityCenterState(response2.ActivityCenterState())
				}
				// Update activity centre notification for requester
				if notification.Type == ActivityCenterNotificationTypeCommunityRequest {
					notification.MembershipStatus = ActivityCenterMembershipStatusIdle
					notification.Read = false
					notification.Deleted = false
					notification.UpdatedAt = m.GetCurrentTimeInMillis()
					err = m.addActivityCenterNotification(response, notification, m.syncActivityCenterUnreadByIDs)
					if err != nil {
						m.logger.Error("failed to update notification in activity center", zap.Error(err))
						return nil, err
					}
				}
			}

			sendSignal = true
		}
	}

	if sendSignal && !sendResponse {
		signal.SendNewMessages(response)
	}

	if sendResponse {
		return response, nil
	}

	return nil, nil
}

func (m *Messenger) CreateCommunityChat(communityID types.HexBytes, c *protobuf.CommunityChat) (*MessengerResponse, error) {
	var response MessengerResponse

	c.Identity.FirstMessageTimestamp = FirstMessageTimestampNoMessage
	changes, err := m.communitiesManager.CreateChat(communityID, c, true, "")
	if err != nil {
		return nil, err
	}
	response.AddCommunity(changes.Community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	var chats []*Chat
	var publicFiltersToInit []transport.FiltersToInitialize
	for chatID, chat := range changes.ChatsAdded {
		c := CreateCommunityChat(changes.Community.IDString(), chatID, chat, m.getTimesource())
		chats = append(chats, c)
		publicFiltersToInit = append(publicFiltersToInit, transport.FiltersToInitialize{ChatID: c.ID, PubsubTopic: changes.Community.PubsubTopic()})

		response.AddChat(c)
	}

	// Load filters
	filters, err := m.transport.InitPublicFilters(publicFiltersToInit)
	if err != nil {
		return nil, err
	}
	_, err = m.scheduleSyncFilters(filters)
	if err != nil {
		return nil, err
	}

	err = m.saveChats(chats)
	if err != nil {
		return nil, err
	}

	err = m.reregisterForPushNotifications()
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (m *Messenger) EditCommunityChat(communityID types.HexBytes, chatID string, c *protobuf.CommunityChat) (*MessengerResponse, error) {
	var response MessengerResponse
	community, changes, err := m.communitiesManager.EditChat(communityID, chatID, c)
	if err != nil {
		return nil, err
	}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	var chats []*Chat
	var publicFiltersToInit []transport.FiltersToInitialize
	for chatID, change := range changes.ChatsModified {
		c := CreateCommunityChat(community.IDString(), chatID, change.ChatModified, m.getTimesource())
		chats = append(chats, c)
		publicFiltersToInit = append(publicFiltersToInit, transport.FiltersToInitialize{ChatID: c.ID, PubsubTopic: community.PubsubTopic()})
		response.AddChat(c)
	}

	// Load filters
	filters, err := m.transport.InitPublicFilters(publicFiltersToInit)
	if err != nil {
		return nil, err
	}
	_, err = m.scheduleSyncFilters(filters)
	if err != nil {
		return nil, err
	}

	return &response, m.saveChats(chats)
}

func (m *Messenger) DeleteCommunityChat(communityID types.HexBytes, chatID string) (*MessengerResponse, error) {
	response := &MessengerResponse{}

	community, _, err := m.communitiesManager.DeleteChat(communityID, chatID)
	if err != nil {
		return nil, err
	}
	err = m.deleteChat(chatID)
	if err != nil {
		return nil, err
	}
	response.AddRemovedChat(chatID)

	_, err = m.transport.RemoveFilterByChatID(chatID)
	if err != nil {
		return nil, err
	}

	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) CreateCommunity(request *requests.CreateCommunity, createDefaultChannel bool) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	response := &MessengerResponse{}

	community, err := m.communitiesManager.CreateCommunity(request, true)
	if err != nil {
		return nil, err
	}

	communitySettings := communities.CommunitySettings{
		CommunityID:                  community.IDString(),
		HistoryArchiveSupportEnabled: request.HistoryArchiveSupportEnabled,
	}
	err = m.communitiesManager.SaveCommunitySettings(communitySettings)
	if err != nil {
		return nil, err
	}

	if err = m.subscribeToCommunityShard(community.ID(), community.Shard()); err != nil {
		return nil, err
	}

	// Init the community filter so we can receive messages on the community
	_, err = m.transport.InitCommunityFilters([]transport.CommunityFilterToInitialize{{
		Shard:   community.Shard().TransportShard(),
		PrivKey: community.PrivateKey(),
	}})
	if err != nil {
		return nil, err
	}

	// Init the default community filters
	_, err = m.transport.InitPublicFilters(community.DefaultFilters())
	if err != nil {
		return nil, err
	}

	if createDefaultChannel {
		chatResponse, err := m.CreateCommunityChat(community.ID(), &protobuf.CommunityChat{
			Identity: &protobuf.ChatIdentity{
				DisplayName:           "general",
				Description:           "General channel for the community",
				Color:                 community.Description().Identity.Color,
				FirstMessageTimestamp: FirstMessageTimestampNoMessage,
			},
			Permissions: &protobuf.CommunityPermissions{
				Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
			},
		})
		if err != nil {
			return nil, err
		}

		// updating community so it contains the general chat
		community = chatResponse.Communities()[0]
		response.AddChat(chatResponse.Chats()[0])
	}

	response.AddCommunity(community)
	response.AddCommunitySettings(&communitySettings)
	err = m.syncCommunity(context.Background(), community, m.dispatchMessage)
	if err != nil {
		return nil, err
	}

	if m.config.torrentConfig != nil && m.config.torrentConfig.Enabled && communitySettings.HistoryArchiveSupportEnabled {
		go m.communitiesManager.StartHistoryArchiveTasksInterval(community, messageArchiveInterval)
	}

	return response, nil
}

func (m *Messenger) SetCommunityShard(request *requests.SetCommunityShard) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, err := m.communitiesManager.SetShard(request.CommunityID, request.Shard)
	if err != nil {
		return nil, err
	}

	var topicPrivKey *ecdsa.PrivateKey
	if request.PrivateKey != nil {
		topicPrivKey, err = crypto.ToECDSA(*request.PrivateKey)
	} else {
		topicPrivKey, err = crypto.GenerateKey()
	}
	if err != nil {
		return nil, err
	}

	err = m.UpdateCommunityFilters(community, topicPrivKey)
	if err != nil {
		return nil, err
	}

	err = m.SendCommunityShardKey(community, community.GetMemberPubkeys())
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddProtectedTopic(&ProtectedTopic{
		CommunityID: community.IDString(),
		PubsubTopic: community.PubsubTopic(),
		PublicKey:   hexutil.Encode(crypto.FromECDSAPub(&topicPrivKey.PublicKey)),
	})

	return response, nil
}

func (m *Messenger) UpdateCommunityFilters(community *communities.Community, privKey *ecdsa.PrivateKey) error {
	if m.transport.WakuVersion() == 2 && privKey != nil {
		if err := m.transport.StorePubsubTopicKey(community.PubsubTopic(), privKey); err != nil {
			return err
		}
	}

	publicFiltersToInit := make([]transport.FiltersToInitialize, 0, len(community.DefaultFilters())+len(community.Chats()))

	publicFiltersToInit = append(publicFiltersToInit, community.DefaultFilters()...)

	for chatID := range community.Chats() {
		communityChatID := community.IDString() + chatID
		_, err := m.transport.RemoveFilterByChatID(communityChatID)
		if err != nil {
			return err
		}
		publicFiltersToInit = append(publicFiltersToInit, transport.FiltersToInitialize{ChatID: communityChatID, PubsubTopic: community.PubsubTopic()})
	}

	_, err := m.transport.InitPublicFilters(publicFiltersToInit)
	if err != nil {
		return err
	}

	// Init the community filter so we can receive messages on the community
	_, err = m.transport.InitCommunityFilters([]transport.CommunityFilterToInitialize{{
		Shard:   community.Shard().TransportShard(),
		PrivKey: community.PrivateKey(),
	}})
	if err != nil {
		return err
	}

	// Init the default community filters
	_, err = m.transport.InitPublicFilters(publicFiltersToInit)
	if err != nil {
		return err
	}

	if err = m.subscribeToCommunityShard(community.ID(), community.Shard()); err != nil {
		return err
	}

	return nil
}

func (m *Messenger) CreateCommunityTokenPermission(request *requests.CreateCommunityTokenPermission) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, changes, err := m.communitiesManager.CreateCommunityTokenPermission(request)
	if err != nil {
		return nil, err
	}

	if community.IsControlNode() {
		// check existing member permission once, then check periodically
		go func() {
			if err := m.communitiesManager.ReevaluateCommunityMembersPermissions(community); err != nil {
				m.logger.Debug("failed to check member permissions", zap.Error(err))
			}

			m.communitiesManager.ReevaluateMembersPeriodically(community.ID())
		}()
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	return response, nil
}

func (m *Messenger) EditCommunityTokenPermission(request *requests.EditCommunityTokenPermission) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, changes, err := m.communitiesManager.EditCommunityTokenPermission(request)
	if err != nil {
		return nil, err
	}

	// check if members still fulfill the token criteria of all
	// BECOME_MEMBER permissions and kick them if necessary
	//
	// We do this in a separate routine to not block this function
	if community.IsControlNode() {
		go func() {
			if err := m.communitiesManager.ReevaluateCommunityMembersPermissions(community); err != nil {
				m.logger.Debug("failed to check member permissions", zap.Error(err))
			}
		}()
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	return response, nil
}

func (m *Messenger) DeleteCommunityTokenPermission(request *requests.DeleteCommunityTokenPermission) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, changes, err := m.communitiesManager.DeleteCommunityTokenPermission(request)
	if err != nil {
		return nil, err
	}

	// check if members still fulfill the token criteria
	// We do this in a separate routine to not block this function
	if community.IsControlNode() {
		go func() {
			if err = m.communitiesManager.ReevaluateCommunityMembersPermissions(community); err != nil {
				m.logger.Debug("failed to check member permissions", zap.Error(err))
			}
		}()
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	return response, nil
}

func (m *Messenger) ReevaluateCommunityMembersPermissions(request *requests.ReevaluateCommunityMembersPermissions) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, err := m.communitiesManager.GetByID(request.CommunityID)
	if err != nil {
		return nil, err
	}

	if err = m.communitiesManager.ReevaluateCommunityMembersPermissions(community); err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)

	return response, nil
}

func (m *Messenger) EditCommunity(request *requests.EditCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, err := m.communitiesManager.EditCommunity(request)
	if err != nil {
		return nil, err
	}

	communitySettings := communities.CommunitySettings{
		CommunityID:                  community.IDString(),
		HistoryArchiveSupportEnabled: request.HistoryArchiveSupportEnabled,
	}
	err = m.communitiesManager.UpdateCommunitySettings(communitySettings)
	if err != nil {
		return nil, err
	}

	id := community.ID()

	if m.torrentClientReady() {
		if !communitySettings.HistoryArchiveSupportEnabled {
			m.communitiesManager.StopHistoryArchiveTasksInterval(id)
		} else if !m.communitiesManager.IsSeedingHistoryArchiveTorrent(id) {
			var communities []*communities.Community
			communities = append(communities, community)
			go m.InitHistoryArchiveTasks(communities)
		}
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	response.AddCommunitySettings(&communitySettings)
	err = m.SyncCommunitySettings(context.Background(), &communitySettings)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) RemovePrivateKey(id types.HexBytes) (*MessengerResponse, error) {
	community, err := m.communitiesManager.RemovePrivateKey(id)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)

	return response, nil
}

func (m *Messenger) ExportCommunity(id types.HexBytes) (*ecdsa.PrivateKey, error) {
	return m.communitiesManager.ExportCommunity(id)
}

func (m *Messenger) ImportCommunity(ctx context.Context, key *ecdsa.PrivateKey) (*MessengerResponse, error) {
	clock, _ := m.getLastClockWithRelatedChat()

	community, err := m.communitiesManager.ImportCommunity(key, clock)
	if err != nil {
		return nil, err
	}

	// Load filters
	_, err = m.transport.InitPublicFilters(community.DefaultFilters())
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	_, err = m.RequestCommunityInfoFromMailserver(community.IDString(), community.Shard(), false)
	if err != nil {
		// TODO In the future we should add a mechanism to re-apply next steps (adding owner, joining)
		// if there is no connection with mailserver. Otherwise changes will be overwritten.
		// Do not return error to make tests pass.
		m.logger.Error("Can't request community info from mailserver")
	}

	// We add ourselves
	community, err = m.communitiesManager.AddMemberOwnerToCommunity(community.ID(), &m.identity.PublicKey)
	if err != nil {
		return nil, err
	}

	response, err := m.JoinCommunity(ctx, community.ID(), true)
	if err != nil {
		return nil, err
	}

	// Notify other clients we are the control node now
	err = m.syncCommunity(context.Background(), community, m.dispatchMessage)
	if err != nil {
		return nil, err
	}

	if m.torrentClientReady() {
		var communities []*communities.Community
		communities = append(communities, community)
		go m.InitHistoryArchiveTasks(communities)
	}
	return response, nil
}

func (m *Messenger) GetCommunityByID(communityID types.HexBytes) (*communities.Community, error) {
	return m.communitiesManager.GetByID(communityID)
}

func (m *Messenger) ShareCommunity(request *requests.ShareCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	response := &MessengerResponse{}

	community, err := m.communitiesManager.GetByID(request.CommunityID)
	if err != nil {
		return nil, err
	}

	var messages []*common.Message
	for _, pk := range request.Users {
		message := common.NewMessage()
		message.ChatId = pk.String()
		message.CommunityID = request.CommunityID.String()
		message.Text = fmt.Sprintf("Community %s has been shared with you", community.Name())
		if request.InviteMessage != "" {
			message.Text = request.InviteMessage
		}
		messages = append(messages, message)
		r, err := m.CreateOneToOneChat(&requests.CreateOneToOneChat{ID: pk})
		if err != nil {
			return nil, err
		}

		if err := response.Merge(r); err != nil {
			return nil, err
		}
	}

	sendMessagesResponse, err := m.SendChatMessages(context.Background(), messages)
	if err != nil {
		return nil, err
	}

	if err := response.Merge(sendMessagesResponse); err != nil {
		return nil, err
	}

	return response, nil
}

func (m *Messenger) MyCanceledRequestsToJoin() ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.CanceledRequestsToJoinForUser(&m.identity.PublicKey)
}

func (m *Messenger) MyPendingRequestsToJoin() ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.PendingRequestsToJoinForUser(&m.identity.PublicKey)
}

func (m *Messenger) MyAwaitingAddressesRequestsToJoin() ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.AwaitingAddressesRequestsToJoinForUser(&m.identity.PublicKey)
}

func (m *Messenger) PendingRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.PendingRequestsToJoinForCommunity(id)
}

func (m *Messenger) AllPendingRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	// TODO: optimize and extract via one query
	pendingRequests, err := m.communitiesManager.PendingRequestsToJoinForCommunity(id)
	if err != nil {
		return nil, err
	}
	acceptedPendingRequests, err := m.communitiesManager.AcceptedPendingRequestsToJoinForCommunity(id)
	if err != nil {
		return nil, err
	}
	declinedPendingRequests, err := m.communitiesManager.DeclinedPendingRequestsToJoinForCommunity(id)
	if err != nil {
		return nil, err
	}

	ownershipChangedRequests, err := m.communitiesManager.RequestsToJoinForCommunityAwaitingAddresses(id)
	if err != nil {
		return nil, err
	}

	pendingRequests = append(pendingRequests, acceptedPendingRequests...)
	pendingRequests = append(pendingRequests, declinedPendingRequests...)
	pendingRequests = append(pendingRequests, ownershipChangedRequests...)

	return pendingRequests, nil
}

func (m *Messenger) DeclinedRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.DeclinedRequestsToJoinForCommunity(id)
}

func (m *Messenger) CanceledRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.CanceledRequestsToJoinForCommunity(id)
}

func (m *Messenger) AcceptedRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.AcceptedRequestsToJoinForCommunity(id)
}

func (m *Messenger) AcceptedPendingRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.AcceptedPendingRequestsToJoinForCommunity(id)
}

func (m *Messenger) DeclinedPendingRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.DeclinedPendingRequestsToJoinForCommunity(id)
}

func (m *Messenger) RemoveUserFromCommunity(id types.HexBytes, pkString string) (*MessengerResponse, error) {
	publicKey, err := common.HexToPubkey(pkString)
	if err != nil {
		return nil, err
	}

	community, err := m.communitiesManager.RemoveUserFromCommunity(id, publicKey)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) SendCommunityShardKey(community *communities.Community, pubkeys []*ecdsa.PublicKey) error {
	if m.transport.WakuVersion() != 2 {
		return nil
	}

	if !community.IsControlNode() {
		return nil
	}

	key, err := m.transport.RetrievePubsubTopicKey(community.PubsubTopic())
	if err != nil {
		return err
	}

	if key == nil {
		return nil // No community shard key available
	}

	communityShardKey := &protobuf.CommunityShardKey{
		Clock:       m.getTimesource().GetCurrentTime(),
		CommunityId: community.ID(),
		PrivateKey:  crypto.FromECDSA(key),
		Shard:       community.Shard().Protobuffer(),
	}

	encodedMessage, err := proto.Marshal(communityShardKey)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Recipients:          pubkeys,
		ResendAutomatically: true,
		MessageType:         protobuf.ApplicationMetadataMessage_COMMUNITY_SHARD_KEY,
		Payload:             encodedMessage,
	}

	_, err = m.sender.SendPubsubTopicKey(context.Background(), &rawMessage)

	return err
}

func (m *Messenger) UnbanUserFromCommunity(request *requests.UnbanUserFromCommunity) (*MessengerResponse, error) {
	community, err := m.communitiesManager.UnbanUserFromCommunity(request)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) BanUserFromCommunity(ctx context.Context, request *requests.BanUserFromCommunity) (*MessengerResponse, error) {
	community, err := m.communitiesManager.BanUserFromCommunity(request)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response, err = m.DeclineAllPendingGroupInvitesFromUser(ctx, response, request.User.String())
	if err != nil {
		return nil, err
	}

	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) AddRoleToMember(request *requests.AddRoleToMember) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	community, err := m.communitiesManager.AddRoleToMember(request)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) RemoveRoleFromMember(request *requests.RemoveRoleFromMember) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	community, err := m.communitiesManager.RemoveRoleFromMember(request)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) findCommunityInfoFromDB(communityID string) (*communities.Community, error) {
	id, err := hexutil.Decode(communityID)
	if err != nil {
		return nil, err
	}

	var community *communities.Community
	community, err = m.GetCommunityByID(id)
	if err != nil {
		return nil, err
	}
	return community, nil
}

func (m *Messenger) GetCommunityIDFromKey(communityKey string) string {
	// Check if the key is a private key. strip the 0x at the start
	privateKey, err := crypto.HexToECDSA(communityKey[2:])
	communityID := ""
	if err != nil {
		// Not a private key, use the public key
		communityID = communityKey
	} else {
		// It is a privateKey
		communityID = types.HexBytes(crypto.CompressPubkey(&privateKey.PublicKey)).String()
	}
	return communityID
}

// RequestCommunityInfoFromMailserver installs filter for community and requests its details
// from mailserver. It waits until it has the community before returning it.
// If useDatabase is true, it searches for community in database and does not request mailserver.
func (m *Messenger) RequestCommunityInfoFromMailserver(privateOrPublicKey string, shard *common.Shard, useDatabase bool) (*communities.Community, error) {
	communityID := m.GetCommunityIDFromKey(privateOrPublicKey)
	if useDatabase {
		community, err := m.findCommunityInfoFromDB(communityID)
		if err != nil {
			return nil, err
		}
		if community != nil {
			return community, nil
		}
	}

	return m.requestCommunityInfoFromMailserver(communityID, shard, true)
}

// RequestCommunityInfoFromMailserverAsync installs filter for community and requests its details
// from mailserver. When response received it will be passed through signals handler
func (m *Messenger) RequestCommunityInfoFromMailserverAsync(privateOrPublicKey string, shard *common.Shard) error {
	communityID := m.GetCommunityIDFromKey(privateOrPublicKey)

	community, err := m.findCommunityInfoFromDB(communityID)
	if err != nil {
		return err
	}
	if community != nil {
		m.config.messengerSignalsHandler.CommunityInfoFound(community)
		return nil
	}
	_, err = m.requestCommunityInfoFromMailserver(communityID, shard, false)
	return err
}

// RequestCommunityInfoFromMailserver installs filter for community and requests its details
// from mailserver. When response received it will be passed through signals handler
func (m *Messenger) requestCommunityInfoFromMailserver(communityID string, shard *common.Shard, waitForResponse bool) (*communities.Community, error) {

	m.logger.Info("requesting community info", zap.String("communityID", communityID), zap.Any("shard", shard))

	m.requestedCommunitiesLock.Lock()
	defer m.requestedCommunitiesLock.Unlock()

	if _, ok := m.requestedCommunities[communityID]; ok {
		return nil, nil
	}

	//If filter wasn't installed we create it and remember for deinstalling after
	//response received
	filter := m.transport.FilterByChatID(communityID)
	if filter == nil {
		filters, err := m.transport.InitPublicFilters([]transport.FiltersToInitialize{{
			ChatID:      communityID,
			PubsubTopic: transport.GetPubsubTopic(shard.TransportShard()),
		}})
		if err != nil {
			return nil, fmt.Errorf("Can't install filter for community: %v", err)
		}
		if len(filters) != 1 {
			return nil, fmt.Errorf("Unexpected amount of filters created")
		}
		filter = filters[0]
		m.requestedCommunities[communityID] = filter
	} else {
		//we don't remember filter id associated with community because it was already installed
		m.requestedCommunities[communityID] = nil
	}

	defer m.forgetCommunityRequest(communityID)

	to := uint32(m.transport.GetCurrentTime() / 1000)
	from := to - oneMonthInSeconds

	_, err := m.performMailserverRequest(func() (*MessengerResponse, error) {
		batch := MailserverBatch{
			From:        from,
			To:          to,
			Topics:      []types.TopicType{filter.ContentTopic},
			PubsubTopic: filter.PubsubTopic,
		}
		m.logger.Info("requesting historic", zap.Any("batch", batch))
		err := m.processMailserverBatch(batch)
		return nil, err
	})

	if err != nil {
		return nil, err
	}

	if !waitForResponse {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for {
		select {
		case <-time.After(200 * time.Millisecond):
			//send signal to client that message status updated
			community, err := m.communitiesManager.GetByIDString(communityID)
			if err != nil {
				return nil, err
			}
			if community != nil && community.Name() != "" && community.DescriptionText() != "" {
				return community, nil
			}

		case <-ctx.Done():
			m.logger.Error("failed to request community info", zap.String("communityID", communityID), zap.Error(ctx.Err()))
			return nil, fmt.Errorf("failed to request community info for id '%s' from mailserver: %w", communityID, ctx.Err())
		}
	}
}

// RequestCommunityInfoFromMailserver installs filter for community and requests its details
// from mailserver. When response received it will be passed through signals handler
func (m *Messenger) requestCommunitiesFromMailserver(communities []communities.CommunityShard) {
	m.requestedCommunitiesLock.Lock()
	defer m.requestedCommunitiesLock.Unlock()

	// we group topics by PubsubTopic
	groupedTopics := map[string]map[types.TopicType]struct{}{}

	for _, c := range communities {
		if _, ok := m.requestedCommunities[c.CommunityID]; ok {
			continue
		}

		//If filter wasn't installed we create it and remember for deinstalling after
		//response received
		filter := m.transport.FilterByChatID(c.CommunityID)
		if filter == nil {
			filters, err := m.transport.InitPublicFilters([]transport.FiltersToInitialize{{
				ChatID:      c.CommunityID,
				PubsubTopic: transport.GetPubsubTopic(c.Shard.TransportShard()),
			}})
			if err != nil {
				m.logger.Error("Can't install filter for community", zap.Error(err))
				continue
			}
			if len(filters) != 1 {
				m.logger.Error("Unexpected amount of filters created")
				continue
			}
			filter = filters[0]
			m.requestedCommunities[c.CommunityID] = filter
		} else {
			//we don't remember filter id associated with community because it was already installed
			m.requestedCommunities[c.CommunityID] = nil
		}

		if _, ok := groupedTopics[filter.PubsubTopic]; !ok {
			groupedTopics[filter.PubsubTopic] = map[types.TopicType]struct{}{}
		}

		groupedTopics[filter.PubsubTopic][filter.ContentTopic] = struct{}{}
	}

	defer func() {
		for _, c := range communities {
			m.forgetCommunityRequest(c.CommunityID)
		}
	}()

	to := uint32(m.transport.GetCurrentTime() / 1000)
	from := to - oneMonthInSeconds

	wg := sync.WaitGroup{}

	for pubsubTopic, contentTopics := range groupedTopics {
		wg.Add(1)
		go func(pubsubTopic string, contentTopics map[types.TopicType]struct{}) {
			batch := MailserverBatch{
				From:        from,
				To:          to,
				Topics:      maps.Keys(contentTopics),
				PubsubTopic: pubsubTopic,
			}
			_, err := m.performMailserverRequest(func() (*MessengerResponse, error) {
				m.logger.Info("requesting historic", zap.Any("batch", batch))
				err := m.processMailserverBatch(batch)
				return nil, err
			})
			if err != nil {
				m.logger.Error("error performing mailserver request", zap.Any("batch", batch), zap.Error(err))
			}
			wg.Done()
		}(pubsubTopic, contentTopics)
	}

	wg.Wait()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	fetching := true
	for fetching {
		select {
		case <-time.After(200 * time.Millisecond):
			allLoaded := true
			for _, c := range communities {
				community, err := m.communitiesManager.GetByIDString(c.CommunityID)
				if err != nil {
					m.logger.Error("Error loading community", zap.Error(err))
					break
				}

				if community == nil || community.Name() == "" || community.DescriptionText() == "" {
					allLoaded = false
					break
				}
			}

			if allLoaded {
				fetching = false
			}

		case <-ctx.Done():
			fetching = false
		}
	}

}

// forgetCommunityRequest removes community from requested ones and removes filter
func (m *Messenger) forgetCommunityRequest(communityID string) {
	m.logger.Info("forgetting community request", zap.String("communityID", communityID))

	filter, ok := m.requestedCommunities[communityID]
	if !ok {
		return
	}

	if filter != nil {
		err := m.transport.RemoveFilters([]*transport.Filter{filter})
		if err != nil {
			m.logger.Warn("cant remove filter", zap.Error(err))
		}
	}

	delete(m.requestedCommunities, communityID)
}

// passStoredCommunityInfoToSignalHandler calls signal handler with community info
func (m *Messenger) passStoredCommunityInfoToSignalHandler(communityID string) {
	if m.config.messengerSignalsHandler == nil {
		return
	}

	//send signal to client that message status updated
	community, err := m.communitiesManager.GetByIDString(communityID)
	if community == nil {
		return
	}

	if err != nil {
		m.logger.Warn("cant get community and pass it to signal handler", zap.Error(err))
		return
	}

	//if there is no info helpful for client, we don't post it
	if community.Name() == "" && community.DescriptionText() == "" && community.MembersCount() == 0 {
		return
	}

	m.config.messengerSignalsHandler.CommunityInfoFound(community)
	m.forgetCommunityRequest(communityID)
}

// handleCommunityDescription handles an community description
func (m *Messenger) handleCommunityDescription(state *ReceivedMessageState, signer *ecdsa.PublicKey, description *protobuf.CommunityDescription, rawPayload []byte) error {
	communityResponse, err := m.communitiesManager.HandleCommunityDescriptionMessage(signer, description, rawPayload, nil)
	if err != nil {
		return err
	}

	// If response is nil, but not error, it will be processed async
	if communityResponse == nil {
		return nil
	}

	return m.handleCommunityResponse(state, communityResponse)
}

func (m *Messenger) handleCommunityResponse(state *ReceivedMessageState, communityResponse *communities.CommunityResponse) error {
	community := communityResponse.Community

	state.Response.AddCommunity(community)
	state.Response.CommunityChanges = append(state.Response.CommunityChanges, communityResponse.Changes)
	state.Response.AddRequestsToJoinCommunity(communityResponse.RequestsToJoin)

	// If we haven't joined/spectated the org, nothing to do
	if !community.Joined() && !community.Spectated() {
		return nil
	}

	removedChatIDs := make([]string, 0)
	for id := range communityResponse.Changes.ChatsRemoved {
		chatID := community.IDString() + id
		_, ok := state.AllChats.Load(chatID)
		if ok {
			removedChatIDs = append(removedChatIDs, chatID)
			state.AllChats.Delete(chatID)
			err := m.DeleteChat(chatID)
			if err != nil {
				m.logger.Error("couldn't delete chat", zap.Error(err))
			}
		}
	}

	// Update relevant chats names and add new ones
	// Currently removal is not supported
	chats := CreateCommunityChats(community, state.Timesource)
	var publicFiltersToInit []transport.FiltersToInitialize
	for i, chat := range chats {

		oldChat, ok := state.AllChats.Load(chat.ID)
		if !ok {
			// Beware, don't use the reference in the range (i.e chat) as it's a shallow copy
			state.AllChats.Store(chat.ID, chats[i])

			state.Response.AddChat(chat)
			publicFiltersToInit = append(publicFiltersToInit, transport.FiltersToInitialize{
				ChatID:      chat.ID,
				PubsubTopic: community.PubsubTopic(),
			})
			// Update name, currently is the only field is mutable
		} else if oldChat.Name != chat.Name ||
			oldChat.Description != chat.Description ||
			oldChat.Emoji != chat.Emoji ||
			oldChat.Color != chat.Color ||
			oldChat.UpdateFirstMessageTimestamp(chat.FirstMessageTimestamp) {
			oldChat.Name = chat.Name
			oldChat.Description = chat.Description
			oldChat.Emoji = chat.Emoji
			oldChat.Color = chat.Color
			// TODO(samyoul) remove storing of an updated reference pointer?
			state.AllChats.Store(chat.ID, oldChat)
			state.Response.AddChat(chat)
		}
	}

	for _, chatID := range removedChatIDs {
		_, err := m.transport.RemoveFilterByChatID(chatID)
		if err != nil {
			m.logger.Error("couldn't remove filter", zap.Error(err))
		}
	}

	// Load transport filters
	filters, err := m.transport.InitPublicFilters(publicFiltersToInit)
	if err != nil {
		return err
	}
	_, err = m.scheduleSyncFilters(filters)
	if err != nil {
		return err
	}

	for _, requestToJoin := range communityResponse.RequestsToJoin {
		// Activity Center notification
		notification, err := m.persistence.GetActivityCenterNotificationByID(requestToJoin.ID)
		if err != nil {
			return err
		}

		if notification != nil {
			notification.MembershipStatus = ActivityCenterMembershipStatusAccepted
			switch requestToJoin.State {
			case communities.RequestToJoinStateDeclined:
				notification.MembershipStatus = ActivityCenterMembershipStatusDeclined
			case communities.RequestToJoinStateAccepted:
				notification.MembershipStatus = ActivityCenterMembershipStatusAccepted
			case communities.RequestToJoinStateAcceptedPending:
				notification.MembershipStatus = ActivityCenterMembershipStatusAcceptedPending
			case communities.RequestToJoinStateDeclinedPending:
				notification.MembershipStatus = ActivityCenterMembershipStatusDeclinedPending
			case communities.RequestToJoinStateAwaitingAddresses:
				notification.MembershipStatus = ActivityCenterMembershipOwnershipChanged
			default:
				notification.MembershipStatus = ActivityCenterMembershipStatusPending

			}

			notification.Read = true
			notification.Accepted = true
			notification.UpdatedAt = m.GetCurrentTimeInMillis()

			err = m.addActivityCenterNotification(state.Response, notification, nil)
			if err != nil {
				m.logger.Error("failed to save notification", zap.Error(err))
				return err
			}
		}
	}

	return nil
}

func (m *Messenger) HandleCommunityEventsMessage(state *ReceivedMessageState, message *protobuf.CommunityEventsMessage, statusMessage *v1protocol.StatusMessage) error {
	signer := state.CurrentMessageState.PublicKey
	communityResponse, err := m.communitiesManager.HandleCommunityEventsMessage(signer, message)
	if err != nil {
		return err
	}

	return m.handleCommunityResponse(state, communityResponse)
}

// Re-sends rejected events, if any.
func (m *Messenger) HandleCommunityEventsMessageRejected(state *ReceivedMessageState, message *protobuf.CommunityEventsMessageRejected, statusMessage *v1protocol.StatusMessage) error {
	signer := state.CurrentMessageState.PublicKey
	reapplyEventsMessage, err := m.communitiesManager.HandleCommunityEventsMessageRejected(signer, message)
	if err != nil {
		return err
	}
	if reapplyEventsMessage == nil {
		return nil
	}

	community, err := m.communitiesManager.GetByID(reapplyEventsMessage.CommunityID)
	if err != nil {
		return err
	}

	err = m.publishCommunityEvents(community, reapplyEventsMessage)
	if err != nil {
		return err
	}

	return nil
}

// HandleCommunityShardKey handles the private keys for the community shards
func (m *Messenger) HandleCommunityShardKey(state *ReceivedMessageState, message *protobuf.CommunityShardKey, statusMessage *v1protocol.StatusMessage) error {
	// TODO: @cammellos: This is risky, it does not seem to support out of order messages
	// (say that the community changes shards twice, last one wins, but we don't check clock
	// etc)

	// TODO: @cammellos: getbyid returns nil if the community is not in the db, so we need to handle it
	community, err := m.communitiesManager.GetByID(message.CommunityId)
	if err != nil {
		return err
	}

	// If we haven't joined the community, nothing to do
	if !community.Joined() {
		return nil
	}

	signer := state.CurrentMessageState.PublicKey
	if signer == nil {
		return errors.New("signer can't be nil")
	}

	if !community.IsMemberOwner(signer) {
		return communities.ErrNotAuthorized
	}

	return m.handleCommunityShardAndFiltersFromProto(community, common.ShardFromProtobuff(message.Shard), message.PrivateKey)
}

func (m *Messenger) handleCommunityShardAndFiltersFromProto(community *communities.Community, shard *common.Shard, privateKeyBytes []byte) error {
	err := m.communitiesManager.UpdateShard(community, shard)
	if err != nil {
		return err
	}

	var privKey *ecdsa.PrivateKey = nil
	if privateKeyBytes != nil {
		privKey, err = crypto.ToECDSA(privateKeyBytes)
		if err != nil {
			return err
		}
	}

	err = m.UpdateCommunityFilters(community, privKey)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) handleCommunityPrivilegedUserSyncMessage(state *ReceivedMessageState, signer *ecdsa.PublicKey, message *protobuf.CommunityPrivilegedUserSyncMessage) error {
	if signer == nil {
		return errors.New("signer can't be nil")
	}

	community, err := m.communitiesManager.GetByID(message.CommunityId)
	if err != nil {
		return err
	}

	// Currently this type of msg coming from the control node.
	// If it will change in the future, check that events types starting from
	// CONTROL_NODE were sent by a control node
	isControlNodeMsg := common.IsPubKeyEqual(community.PublicKey(), signer)
	if !isControlNodeMsg {
		return errors.New("accepted/requested to join sync messages can be send only by the control node")
	}

	if community == nil {
		return errors.New("community not found")
	}

	err = m.communitiesManager.ValidateCommunityPrivilegedUserSyncMessage(message)
	if err != nil {
		return err
	}

	switch message.Type {
	case protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ACCEPT_REQUEST_TO_JOIN:
		fallthrough
	case protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_REJECT_REQUEST_TO_JOIN:
		requestsToJoin, err := m.communitiesManager.HandleRequestToJoinPrivilegedUserSyncMessage(message, community.ID())
		if err != nil {
			return nil
		}
		state.Response.AddRequestsToJoinCommunity(requestsToJoin)

	case protobuf.CommunityPrivilegedUserSyncMessage_CONTROL_NODE_ALL_SYNC_REQUESTS_TO_JOIN:
		requestsToJoin, err := m.communitiesManager.HandleSyncAllRequestToJoinForNewPrivilegedMember(message, community.ID())
		if err != nil {
			return nil
		}
		state.Response.AddRequestsToJoinCommunity(requestsToJoin)
	}

	return nil
}

func (m *Messenger) HandleCommunityPrivilegedUserSyncMessage(state *ReceivedMessageState, message *protobuf.CommunityPrivilegedUserSyncMessage, statusMessage *v1protocol.StatusMessage) error {
	signer := state.CurrentMessageState.PublicKey
	return m.handleCommunityPrivilegedUserSyncMessage(state, signer, message)
}

func (m *Messenger) sendSharedAddressToControlNode(receiver *ecdsa.PublicKey, community *communities.Community) (*communities.RequestToJoin, error) {
	if receiver == nil {
		return nil, errors.New("receiver can't be nil")
	}

	if community == nil {
		return nil, communities.ErrOrgNotFound
	}

	m.logger.Info("share address to the new owner ", zap.String("community id", community.IDString()))

	pk := common.PubkeyToHex(&m.identity.PublicKey)

	requestToJoin, err := m.communitiesManager.GetCommunityRequestToJoinWithRevealedAddresses(pk, community.ID())
	if err != nil {
		return nil, err
	}

	requestToJoin.Clock = uint64(time.Now().Unix())
	requestToJoin.State = communities.RequestToJoinStateAwaitingAddresses
	payload, err := proto.Marshal(requestToJoin.ToCommunityRequestToJoinProtobuf())
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		Payload:           payload,
		CommunityID:       community.ID(),
		SkipProtocolLayer: true,
		MessageType:       protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN,
		PubsubTopic:       community.PubsubTopic(), // TODO: confirm if it should be sent in community pubsub topic
	}

	if err = m.communitiesManager.SaveRequestToJoin(requestToJoin); err != nil {
		return nil, err
	}

	_, err = m.sender.SendPrivate(context.Background(), receiver, &rawMessage)
	return requestToJoin, err
}

func (m *Messenger) HandleSyncInstallationCommunity(messageState *ReceivedMessageState, syncCommunity *protobuf.SyncInstallationCommunity, statusMessage *v1protocol.StatusMessage) error {
	return m.handleSyncInstallationCommunity(messageState, syncCommunity, nil)
}

func (m *Messenger) handleSyncInstallationCommunity(messageState *ReceivedMessageState, syncCommunity *protobuf.SyncInstallationCommunity, statusMessage *v1protocol.StatusMessage) error {
	logger := m.logger.Named("handleSyncCommunity")

	// Should handle community
	shouldHandle, err := m.communitiesManager.ShouldHandleSyncCommunity(syncCommunity)
	if err != nil {
		logger.Debug("m.communitiesManager.ShouldHandleSyncCommunity error", zap.Error(err))
		return err
	}
	logger.Debug("ShouldHandleSyncCommunity result", zap.Bool("shouldHandle", shouldHandle))
	if !shouldHandle {
		return nil
	}

	// Handle community keys
	if len(syncCommunity.EncryptionKeys) != 0 {
		//  We pass nil,nil as private key/public key as they won't be encrypted
		_, err := m.encryptor.HandleHashRatchetKeys(syncCommunity.Id, syncCommunity.EncryptionKeys, nil, nil)
		if err != nil {
			return err
		}
	}

	// Handle any community requests to join.
	// MUST BE HANDLED BEFORE DESCRIPTION!
	pending := false
	for _, rtj := range syncCommunity.RequestsToJoin {
		req := new(communities.RequestToJoin)
		req.InitFromSyncProtobuf(rtj)

		if req.State == communities.RequestToJoinStatePending {
			pending = true
		}

		err = m.communitiesManager.SaveRequestToJoin(req)
		if err != nil && err != communities.ErrOldRequestToJoin {
			logger.Debug("m.communitiesManager.SaveRequestToJoin error", zap.Error(err))
			return err
		}
	}
	logger.Debug("community requests to join pending state", zap.Bool("pending", pending))

	// Don't use the public key of the private key, uncompress the community id
	orgPubKey, err := crypto.DecompressPubkey(syncCommunity.Id)
	if err != nil {
		logger.Debug("crypto.DecompressPubkey error", zap.Error(err))
		return err
	}
	logger.Debug("crypto.DecompressPubkey result", zap.Any("orgPubKey", orgPubKey))

	var amm protobuf.ApplicationMetadataMessage
	err = proto.Unmarshal(syncCommunity.Description, &amm)
	if err != nil {
		logger.Debug("proto.Unmarshal protobuf.ApplicationMetadataMessage error", zap.Error(err))
		return err
	}

	var cd protobuf.CommunityDescription
	err = proto.Unmarshal(amm.Payload, &cd)
	if err != nil {
		logger.Debug("proto.Unmarshal protobuf.CommunityDescription error", zap.Error(err))
		return err
	}

	err = m.handleCommunityDescription(messageState, orgPubKey, &cd, syncCommunity.Description)
	if err != nil {
		logger.Debug("m.handleCommunityDescription error", zap.Error(err))
		return err
	}

	if syncCommunity.Settings != nil {
		err = m.HandleSyncCommunitySettings(messageState, syncCommunity.Settings, nil)
		if err != nil {
			logger.Debug("m.handleSyncCommunitySettings error", zap.Error(err))
			return err
		}
	}

	if syncCommunity.ControlNode != nil {
		err = m.communitiesManager.SetSyncControlNode(syncCommunity.Id, syncCommunity.ControlNode)
		if err != nil {
			logger.Debug("m.SetSyncControlNode", zap.Error(err))
			return err
		}
	}

	savedCommunity, err := m.communitiesManager.GetByID(syncCommunity.Id)
	if err != nil {
		return err
	}

	if err := m.handleCommunityTokensMetadataByPrivilegedMembers(savedCommunity); err != nil {
		logger.Debug("m.handleCommunityTokensMetadataByPrivilegedMembers", zap.Error(err))
		return err
	}

	// if we are not waiting for approval, join or leave the community
	if !pending {
		var mr *MessengerResponse
		if syncCommunity.Joined {
			mr, err = m.joinCommunity(context.Background(), syncCommunity.Id, false)
			if err != nil && err != communities.ErrOrgAlreadyJoined {
				logger.Debug("m.joinCommunity error", zap.Error(err))
				return err
			}
		} else {
			mr, err = m.leaveCommunity(syncCommunity.Id)
			if err != nil {
				logger.Debug("m.leaveCommunity error", zap.Error(err))
				return err
			}
		}
		if mr != nil {
			err = messageState.Response.Merge(mr)
			if err != nil {
				logger.Debug("messageState.Response.Merge error", zap.Error(err))
				return err
			}
		}
	}

	// update the clock value
	err = m.communitiesManager.SetSyncClock(syncCommunity.Id, syncCommunity.Clock)
	if err != nil {
		logger.Debug("m.communitiesManager.SetSyncClock", zap.Error(err))
		return err
	}

	return nil
}

func (m *Messenger) HandleSyncCommunitySettings(messageState *ReceivedMessageState, syncCommunitySettings *protobuf.SyncCommunitySettings, statusMessage *v1protocol.StatusMessage) error {
	shouldHandle, err := m.communitiesManager.ShouldHandleSyncCommunitySettings(syncCommunitySettings)
	if err != nil {
		m.logger.Debug("m.communitiesManager.ShouldHandleSyncCommunitySettings error", zap.Error(err))
		return err
	}
	m.logger.Debug("ShouldHandleSyncCommunity result", zap.Bool("shouldHandle", shouldHandle))
	if !shouldHandle {
		return nil
	}

	communitySettings, err := m.communitiesManager.HandleSyncCommunitySettings(syncCommunitySettings)
	if err != nil {
		return err
	}

	messageState.Response.AddCommunitySettings(communitySettings)
	return nil
}

func (m *Messenger) handleCommunityTokensMetadataByPrivilegedMembers(community *communities.Community) error {
	return m.communitiesManager.HandleCommunityTokensMetadataByPrivilegedMembers(community)
}

func (m *Messenger) InitHistoryArchiveTasks(communities []*communities.Community) {

	m.communitiesManager.LogStdout("initializing history archive tasks")

	for _, c := range communities {

		if c.Joined() {
			settings, err := m.communitiesManager.GetCommunitySettingsByID(c.ID())
			if err != nil {
				m.communitiesManager.LogStdout("failed to get community settings", zap.Error(err))
				continue
			}
			if !settings.HistoryArchiveSupportEnabled {
				m.communitiesManager.LogStdout("history archive support disabled for community", zap.String("id", c.IDString()))
				continue
			}

			// Check if there's already a torrent file for this community and seed it
			if m.communitiesManager.TorrentFileExists(c.IDString()) {
				err = m.communitiesManager.SeedHistoryArchiveTorrent(c.ID())
				if err != nil {
					m.communitiesManager.LogStdout("failed to seed history archive", zap.Error(err))
				}
			}

			filters, err := m.communitiesManager.GetCommunityChatsFilters(c.ID())
			if err != nil {
				m.communitiesManager.LogStdout("failed to get community chats filters for community", zap.Error(err))
				continue
			}

			if len(filters) == 0 {
				m.communitiesManager.LogStdout("no filters or chats for this community starting interval", zap.String("id", c.IDString()))
				go m.communitiesManager.StartHistoryArchiveTasksInterval(c, messageArchiveInterval)
				continue
			}

			topics := []types.TopicType{}

			for _, filter := range filters {
				topics = append(topics, filter.ContentTopic)
			}

			// First we need to know the timestamp of the latest waku message
			// we've received for this community, so we can request messages we've
			// possibly missed since then
			latestWakuMessageTimestamp, err := m.communitiesManager.GetLatestWakuMessageTimestamp(topics)
			if err != nil {
				m.communitiesManager.LogStdout("failed to get Latest waku message timestamp", zap.Error(err))
				continue
			}

			if latestWakuMessageTimestamp == 0 {
				// This means we don't have any waku messages for this community
				// yet, either because no messages were sent in the community so far,
				// or because messages haven't reached this node
				//
				// In this case we default to requesting messages from the store nodes
				// for the past 30 days
				latestWakuMessageTimestamp = uint64(time.Now().AddDate(0, 0, -30).Unix())
			}

			// Request possibly missed waku messages for community
			_, err = m.syncFiltersFrom(filters, uint32(latestWakuMessageTimestamp))
			if err != nil {
				m.communitiesManager.LogStdout("failed to request missing messages", zap.Error(err))
				continue
			}

			// We figure out the end date of the last created archive and schedule
			// the interval for creating future archives
			// If the last end date is at least `interval` ago, we create an archive immediately first
			lastArchiveEndDateTimestamp, err := m.communitiesManager.GetHistoryArchivePartitionStartTimestamp(c.ID())
			if err != nil {
				m.communitiesManager.LogStdout("failed to get archive partition start timestamp", zap.Error(err))
				continue
			}

			to := time.Now()
			lastArchiveEndDate := time.Unix(int64(lastArchiveEndDateTimestamp), 0)
			durationSinceLastArchive := to.Sub(lastArchiveEndDate)

			if lastArchiveEndDateTimestamp == 0 {
				// No prior messages to be archived, so we just kick off the archive creation loop
				// for future archives
				go m.communitiesManager.StartHistoryArchiveTasksInterval(c, messageArchiveInterval)
			} else if durationSinceLastArchive < messageArchiveInterval {
				// Last archive is less than `interval` old, wait until `interval` is complete,
				// then create archive and kick off archive creation loop for future archives
				// Seed current archive in the meantime
				err := m.communitiesManager.SeedHistoryArchiveTorrent(c.ID())
				if err != nil {
					m.communitiesManager.LogStdout("failed to seed history archive", zap.Error(err))
				}
				timeToNextInterval := messageArchiveInterval - durationSinceLastArchive

				m.communitiesManager.LogStdout("starting history archive tasks interval in", zap.Any("timeLeft", timeToNextInterval))
				time.AfterFunc(timeToNextInterval, func() {
					err := m.communitiesManager.CreateAndSeedHistoryArchive(c.ID(), topics, lastArchiveEndDate, to.Add(timeToNextInterval), messageArchiveInterval, c.Encrypted())
					if err != nil {
						m.communitiesManager.LogStdout("failed to get create and seed history archive", zap.Error(err))
					}
					go m.communitiesManager.StartHistoryArchiveTasksInterval(c, messageArchiveInterval)
				})
			} else {
				// Looks like the last archive was generated more than `interval`
				// ago, so lets create a new archive now and then schedule the archive
				// creation loop
				err := m.communitiesManager.CreateAndSeedHistoryArchive(c.ID(), topics, lastArchiveEndDate, to, messageArchiveInterval, c.Encrypted())
				if err != nil {
					m.communitiesManager.LogStdout("failed to get create and seed history archive", zap.Error(err))
				}

				go m.communitiesManager.StartHistoryArchiveTasksInterval(c, messageArchiveInterval)
			}
		}
	}
}

func (m *Messenger) enableHistoryArchivesImportAfterDelay() {
	go func() {
		time.Sleep(importInitialDelay)
		m.importDelayer.once.Do(func() {
			close(m.importDelayer.wait)
		})
	}()
}

func (m *Messenger) checkIfIMemberOfCommunity(communityID types.HexBytes) error {
	community, err := m.communitiesManager.GetByID(communityID)
	if err != nil {
		m.communitiesManager.LogStdout("couldn't get community to import archives", zap.Error(err))
		return err
	}

	if !community.HasMember(&m.identity.PublicKey) {
		m.communitiesManager.LogStdout("can't import archives when user not a member of community")
		return ErrUserNotMember
	}

	return nil
}

func (m *Messenger) resumeHistoryArchivesImport(communityID types.HexBytes) error {
	archiveIDsToImport, err := m.communitiesManager.GetMessageArchiveIDsToImport(communityID)
	if err != nil {
		return err
	}

	if len(archiveIDsToImport) == 0 {
		return nil
	}

	err = m.checkIfIMemberOfCommunity(communityID)
	if err != nil {
		return err
	}

	currentTask := m.communitiesManager.GetHistoryArchiveDownloadTask(communityID.String())
	// no need to resume imports if there's already a task ongoing
	if currentTask != nil {
		return nil
	}

	// Create new task
	task := &communities.HistoryArchiveDownloadTask{
		CancelChan: make(chan struct{}),
		Waiter:     *new(sync.WaitGroup),
		Cancelled:  false,
	}

	m.communitiesManager.AddHistoryArchiveDownloadTask(communityID.String(), task)

	// this wait groups tracks the ongoing task for a particular community
	task.Waiter.Add(1)

	go func() {
		defer task.Waiter.Done()
		err := m.importHistoryArchives(communityID, task.CancelChan)
		if err != nil {
			m.communitiesManager.LogStdout("failed to import history archives", zap.Error(err))
		}
		m.config.messengerSignalsHandler.DownloadingHistoryArchivesFinished(types.EncodeHex(communityID))
	}()
	return nil
}

func (m *Messenger) SpeedupArchivesImport() {
	m.importRateLimiter.SetLimit(rate.Every(importFastRate))
}

func (m *Messenger) SlowdownArchivesImport() {
	m.importRateLimiter.SetLimit(rate.Every(importSlowRate))
}

func (m *Messenger) importHistoryArchives(communityID types.HexBytes, cancel chan struct{}) error {
	importTicker := time.NewTicker(100 * time.Millisecond)
	defer importTicker.Stop()

	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		<-cancel
		cancelFunc()
	}()

	// don't proceed until initial import delay has passed
	select {
	case <-m.importDelayer.wait:
	case <-ctx.Done():
		return nil
	}

importMessageArchivesLoop:
	for {
		select {
		case <-ctx.Done():
			m.communitiesManager.LogStdout("interrupted importing history archive messages")
			return nil
		case <-importTicker.C:
			err := m.checkIfIMemberOfCommunity(communityID)
			if err != nil {
				break importMessageArchivesLoop
			}
			archiveIDsToImport, err := m.communitiesManager.GetMessageArchiveIDsToImport(communityID)
			if err != nil {
				m.communitiesManager.LogStdout("couldn't get message archive IDs to import", zap.Error(err))
				return err
			}

			if len(archiveIDsToImport) == 0 {
				m.communitiesManager.LogStdout("no message archives to import")
				break importMessageArchivesLoop
			}

			m.communitiesManager.LogStdout(fmt.Sprintf("importing message archive, %d left", len(archiveIDsToImport)))

			// only process one archive at a time, so in case of cancel we don't
			// wait for all archives to be processed first
			downloadedArchiveID := archiveIDsToImport[0]

			archiveMessages, err := m.communitiesManager.ExtractMessagesFromHistoryArchive(communityID, downloadedArchiveID)
			if err != nil {
				m.communitiesManager.LogStdout("failed to extract history archive messages", zap.Error(err))
				continue
			}

			m.config.messengerSignalsHandler.ImportingHistoryArchiveMessages(types.EncodeHex(communityID))

			for _, messagesChunk := range chunkSlice(archiveMessages, importMessagesChunkSize) {
				if err := m.importRateLimiter.Wait(ctx); err != nil {
					if !errors.Is(err, context.Canceled) {
						m.communitiesManager.LogStdout("rate limiter error when handling archive messages", zap.Error(err))
					}
					continue importMessageArchivesLoop
				}

				response, err := m.handleArchiveMessages(messagesChunk)
				if err != nil {
					m.communitiesManager.LogStdout("failed to handle archive messages", zap.Error(err))
					continue importMessageArchivesLoop
				}

				if !response.IsEmpty() {
					notifications := response.Notifications()
					response.ClearNotifications()
					signal.SendNewMessages(response)
					localnotifications.PushMessages(notifications)
				}
			}

			err = m.communitiesManager.SetMessageArchiveIDImported(communityID, downloadedArchiveID, true)
			if err != nil {
				m.communitiesManager.LogStdout("failed to mark history message archive as imported", zap.Error(err))
				continue
			}
		}
	}
	return nil
}

func (m *Messenger) dispatchMagnetlinkMessage(communityID string) error {

	community, err := m.communitiesManager.GetByIDString(communityID)
	if err != nil {
		return err
	}

	magnetlink, err := m.communitiesManager.GetHistoryArchiveMagnetlink(community.ID())
	if err != nil {
		return err
	}

	magnetLinkMessage := &protobuf.CommunityMessageArchiveMagnetlink{
		Clock:     m.getTimesource().GetCurrentTime(),
		MagnetUri: magnetlink,
	}

	encodedMessage, err := proto.Marshal(magnetLinkMessage)
	if err != nil {
		return err
	}

	chatID := community.MagnetlinkMessageChannelID()
	rawMessage := common.RawMessage{
		LocalChatID:          chatID,
		Sender:               community.PrivateKey(),
		Payload:              encodedMessage,
		MessageType:          protobuf.ApplicationMetadataMessage_COMMUNITY_MESSAGE_ARCHIVE_MAGNETLINK,
		SkipGroupMessageWrap: true,
		PubsubTopic:          community.PubsubTopic(),
	}

	_, err = m.sender.SendPublic(context.Background(), chatID, rawMessage)
	if err != nil {
		return err
	}

	err = m.communitiesManager.UpdateCommunityDescriptionMagnetlinkMessageClock(community.ID(), magnetLinkMessage.Clock)
	if err != nil {
		return err
	}
	return m.communitiesManager.UpdateMagnetlinkMessageClock(community.ID(), magnetLinkMessage.Clock)
}

func (m *Messenger) EnableCommunityHistoryArchiveProtocol() error {
	nodeConfig, err := m.settings.GetNodeConfig()
	if err != nil {
		return err
	}

	if nodeConfig.TorrentConfig.Enabled {
		return nil
	}

	nodeConfig.TorrentConfig.Enabled = true
	err = m.settings.SaveSetting("node-config", nodeConfig)
	if err != nil {
		return err
	}

	m.config.torrentConfig = &nodeConfig.TorrentConfig
	m.communitiesManager.SetTorrentConfig(&nodeConfig.TorrentConfig)
	err = m.communitiesManager.StartTorrentClient()
	if err != nil {
		return err
	}

	controlledCommunities, err := m.communitiesManager.Controlled()
	if err != nil {
		return err
	}

	if len(controlledCommunities) > 0 {
		go m.InitHistoryArchiveTasks(controlledCommunities)
	}
	m.config.messengerSignalsHandler.HistoryArchivesProtocolEnabled()
	return nil
}

func (m *Messenger) DisableCommunityHistoryArchiveProtocol() error {

	nodeConfig, err := m.settings.GetNodeConfig()
	if err != nil {
		return err
	}
	if !nodeConfig.TorrentConfig.Enabled {
		return nil
	}

	m.communitiesManager.StopTorrentClient()

	nodeConfig.TorrentConfig.Enabled = false
	err = m.settings.SaveSetting("node-config", nodeConfig)
	m.config.torrentConfig = &nodeConfig.TorrentConfig
	m.communitiesManager.SetTorrentConfig(&nodeConfig.TorrentConfig)
	if err != nil {
		return err
	}
	m.config.messengerSignalsHandler.HistoryArchivesProtocolDisabled()
	return nil
}

func (m *Messenger) GetCommunitiesSettings() ([]communities.CommunitySettings, error) {
	settings, err := m.communitiesManager.GetCommunitiesSettings()
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (m *Messenger) SyncCommunitySettings(ctx context.Context, settings *communities.CommunitySettings) error {

	if !m.hasPairedDevices() {
		return nil
	}

	clock, chat := m.getLastClockWithRelatedChat()

	syncMessage := &protobuf.SyncCommunitySettings{
		Clock:                        clock,
		CommunityId:                  settings.CommunityID,
		HistoryArchiveSupportEnabled: settings.HistoryArchiveSupportEnabled,
	}
	encodedMessage, err := proto.Marshal(syncMessage)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_COMMUNITY_SETTINGS,
		ResendAutomatically: true,
	})
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) ExtractDiscordDataFromImportFiles(filesToImport []string) (*discord.ExtractedData, map[string]*discord.ImportError) {

	extractedData := &discord.ExtractedData{
		Categories:             map[string]*discord.Category{},
		ExportedData:           make([]*discord.ExportedData, 0),
		OldestMessageTimestamp: 0,
		MessageCount:           0,
	}

	errors := map[string]*discord.ImportError{}

	for _, fileToImport := range filesToImport {
		filePath := strings.Replace(fileToImport, "file://", "", -1)

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			errors[fileToImport] = discord.Error(err.Error())
			continue
		}

		fileSize := fileInfo.Size()
		if fileSize > discord.MaxImportFileSizeBytes {
			errors[fileToImport] = discord.Error(discord.ErrImportFileTooBig.Error())
			continue
		}

		bytes, err := os.ReadFile(filePath)
		if err != nil {
			errors[fileToImport] = discord.Error(err.Error())
			continue
		}

		var discordExportedData discord.ExportedData

		err = json.Unmarshal(bytes, &discordExportedData)
		if err != nil {
			errors[fileToImport] = discord.Error(err.Error())
			continue
		}

		if len(discordExportedData.Messages) == 0 {
			errors[fileToImport] = discord.Error(discord.ErrNoMessageData.Error())
			continue
		}

		discordExportedData.Channel.FilePath = filePath
		categoryID := discordExportedData.Channel.CategoryID

		discordCategory := discord.Category{
			ID:   categoryID,
			Name: discordExportedData.Channel.CategoryName,
		}

		_, ok := extractedData.Categories[categoryID]
		if !ok {
			extractedData.Categories[categoryID] = &discordCategory
		}

		extractedData.MessageCount = extractedData.MessageCount + discordExportedData.MessageCount
		extractedData.ExportedData = append(extractedData.ExportedData, &discordExportedData)

		if len(discordExportedData.Messages) > 0 {
			msgTime, err := time.Parse(discordTimestampLayout, discordExportedData.Messages[0].Timestamp)
			if err != nil {
				m.logger.Error("failed to parse discord message timestamp", zap.Error(err))
				continue
			}

			if extractedData.OldestMessageTimestamp == 0 || int(msgTime.Unix()) <= extractedData.OldestMessageTimestamp {
				// Exported discord channel data already comes with `messages` being
				// sorted, starting with the oldest, so we can safely rely on the first
				// message
				extractedData.OldestMessageTimestamp = int(msgTime.Unix())
			}
		}
	}
	return extractedData, errors
}

func (m *Messenger) ExtractDiscordChannelsAndCategories(filesToImport []string) (*MessengerResponse, map[string]*discord.ImportError) {

	response := &MessengerResponse{}

	extractedData, errs := m.ExtractDiscordDataFromImportFiles(filesToImport)

	for _, category := range extractedData.Categories {
		response.AddDiscordCategory(category)
	}
	for _, export := range extractedData.ExportedData {
		response.AddDiscordChannel(&export.Channel)
	}
	if extractedData.OldestMessageTimestamp != 0 {
		response.DiscordOldestMessageTimestamp = extractedData.OldestMessageTimestamp
	}

	return response, errs
}

func (m *Messenger) RequestExtractDiscordChannelsAndCategories(filesToImport []string) {
	go func() {
		response, errors := m.ExtractDiscordChannelsAndCategories(filesToImport)
		m.config.messengerSignalsHandler.DiscordCategoriesAndChannelsExtracted(
			response.DiscordCategories,
			response.DiscordChannels,
			int64(response.DiscordOldestMessageTimestamp),
			errors)
	}()
}

func (m *Messenger) saveDiscordAuthorIfNotExists(discordAuthor *protobuf.DiscordMessageAuthor) *discord.ImportError {
	exists, err := m.persistence.HasDiscordMessageAuthor(discordAuthor.GetId())
	if err != nil {
		m.logger.Error("failed to check if message author exists in database", zap.Error(err))
		return discord.Error(err.Error())
	}

	if !exists {
		err := m.persistence.SaveDiscordMessageAuthor(discordAuthor)
		if err != nil {
			return discord.Error(err.Error())
		}
	}

	return nil
}

func (m *Messenger) convertDiscordMessageTimeStamp(discordMessage *protobuf.DiscordMessage, timestamp time.Time) *discord.ImportError {
	discordMessage.Timestamp = fmt.Sprintf("%d", timestamp.Unix())

	if discordMessage.TimestampEdited != "" {
		timestampEdited, err := time.Parse(discordTimestampLayout, discordMessage.TimestampEdited)
		if err != nil {
			m.logger.Error("failed to parse discord message timestamp", zap.Error(err))
			return discord.Warning(err.Error())
		}
		// Convert timestamp to unix timestamp
		discordMessage.TimestampEdited = fmt.Sprintf("%d", timestampEdited.Unix())
	}

	return nil
}

func (m *Messenger) createPinMessageFromDiscordMessage(message *common.Message, pinnedMessageID string, channelID string, community *communities.Community) (*common.PinMessage, *discord.ImportError) {
	pinMessage := protobuf.PinMessage{
		Clock:       message.WhisperTimestamp,
		MessageId:   pinnedMessageID,
		ChatId:      message.LocalChatID,
		MessageType: protobuf.MessageType_COMMUNITY_CHAT,
		Pinned:      true,
	}

	encodedPayload, err := proto.Marshal(&pinMessage)
	if err != nil {
		m.logger.Error("failed to parse marshal pin message", zap.Error(err))
		return nil, discord.Warning(err.Error())
	}

	wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, protobuf.ApplicationMetadataMessage_PIN_MESSAGE, community.PrivateKey())
	if err != nil {
		m.logger.Error("failed to wrap pin message", zap.Error(err))
		return nil, discord.Warning(err.Error())
	}

	pinMessageToSave := &common.PinMessage{
		ID:               types.EncodeHex(v1protocol.MessageID(&community.PrivateKey().PublicKey, wrappedPayload)),
		PinMessage:       &pinMessage,
		LocalChatID:      channelID,
		From:             message.From,
		SigPubKey:        message.SigPubKey,
		WhisperTimestamp: message.WhisperTimestamp,
	}

	return pinMessageToSave, nil
}

func (m *Messenger) generateSystemPinnedMessage(pinMessage *common.PinMessage, channel *Chat, clockAndTimestamp uint64, pinnedMessageID string) (*common.Message, *discord.ImportError) {
	id, err := generatePinMessageNotificationID(&m.identity.PublicKey, pinMessage, channel)
	if err != nil {
		m.logger.Warn("failed to generate pin message notification ID",
			zap.String("PinMessageId", pinMessage.ID))
		return nil, discord.Warning(err.Error())
	}
	systemMessage := &common.Message{
		ChatMessage: &protobuf.ChatMessage{
			Clock:       pinMessage.Clock,
			Timestamp:   clockAndTimestamp,
			ChatId:      channel.ID,
			MessageType: pinMessage.MessageType,
			ResponseTo:  pinnedMessageID,
			ContentType: protobuf.ChatMessage_SYSTEM_MESSAGE_PINNED_MESSAGE,
		},
		WhisperTimestamp: clockAndTimestamp,
		ID:               id,
		LocalChatID:      channel.ID,
		From:             pinMessage.From,
		Seen:             true,
	}

	return systemMessage, nil
}

func (m *Messenger) processDiscordMessages(discordChannel *discord.ExportedData,
	channel *Chat,
	importProgress *discord.ImportProgress,
	progressUpdates chan *discord.ImportProgress,
	fromDate int64,
	community *communities.Community) (
	map[string]*common.Message,
	[]*common.PinMessage,
	map[string]*protobuf.DiscordMessageAuthor,
	[]*protobuf.DiscordMessageAttachment) {

	messagesToSave := make(map[string]*common.Message, 0)
	pinMessagesToSave := make([]*common.PinMessage, 0)
	authorProfilesToSave := make(map[string]*protobuf.DiscordMessageAuthor, 0)
	messageAttachmentsToDownload := make([]*protobuf.DiscordMessageAttachment, 0)

	for _, discordMessage := range discordChannel.Messages {

		timestamp, err := time.Parse(discordTimestampLayout, discordMessage.Timestamp)
		if err != nil {
			m.logger.Error("failed to parse discord message timestamp", zap.Error(err))
			importProgress.AddTaskError(discord.ImportMessagesTask, discord.Warning(err.Error()))
			progressUpdates <- importProgress
			continue
		}

		if timestamp.Unix() < fromDate {
			progressUpdates <- importProgress
			continue
		}

		importErr := m.saveDiscordAuthorIfNotExists(discordMessage.Author)
		if importErr != nil {
			importProgress.AddTaskError(discord.ImportMessagesTask, importErr)
			progressUpdates <- importProgress
			continue
		}

		hasPayload, err := m.persistence.HasDiscordMessageAuthorImagePayload(discordMessage.Author.GetId())
		if err != nil {
			m.logger.Error("failed to check if message avatar payload exists in database", zap.Error(err))
			importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
			progressUpdates <- importProgress
			continue
		}

		if !hasPayload {
			authorProfilesToSave[discordMessage.Author.Id] = discordMessage.Author
		}

		// Convert timestamp to unix timestamp
		importErr = m.convertDiscordMessageTimeStamp(discordMessage, timestamp)
		if importErr != nil {
			importProgress.AddTaskError(discord.ImportMessagesTask, importErr)
			progressUpdates <- importProgress
			continue
		}

		for i := range discordMessage.Attachments {
			discordMessage.Attachments[i].MessageId = discordMessage.Id
		}
		messageAttachmentsToDownload = append(messageAttachmentsToDownload, discordMessage.Attachments...)

		clockAndTimestamp := uint64(timestamp.Unix()) * 1000
		communityPubKey := community.PrivateKey().PublicKey

		chatMessage := protobuf.ChatMessage{
			Timestamp:   clockAndTimestamp,
			MessageType: protobuf.MessageType_COMMUNITY_CHAT,
			ContentType: protobuf.ChatMessage_DISCORD_MESSAGE,
			Clock:       clockAndTimestamp,
			ChatId:      channel.ID,
			Payload: &protobuf.ChatMessage_DiscordMessage{
				DiscordMessage: discordMessage,
			},
		}

		// Handle message replies
		if discordMessage.Type == string(discord.MessageTypeReply) && discordMessage.Reference != nil {
			repliedMessageID := community.IDString() + discordMessage.Reference.MessageId
			if _, exists := messagesToSave[repliedMessageID]; exists {
				chatMessage.ResponseTo = repliedMessageID
			}
		}

		messageToSave := &common.Message{
			ID:               community.IDString() + discordMessage.Id,
			WhisperTimestamp: clockAndTimestamp,
			From:             types.EncodeHex(crypto.FromECDSAPub(&communityPubKey)),
			Seen:             true,
			LocalChatID:      channel.ID,
			SigPubKey:        &communityPubKey,
			CommunityID:      community.IDString(),
			ChatMessage:      &chatMessage,
		}

		err = messageToSave.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
		if err != nil {
			m.logger.Error("failed to prepare message content", zap.Error(err))
			importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
			progressUpdates <- importProgress
			continue
		}

		// Handle pin messages
		if discordMessage.Type == string(discord.MessageTypeChannelPinned) && discordMessage.Reference != nil {

			pinnedMessageID := community.IDString() + discordMessage.Reference.MessageId
			_, exists := messagesToSave[pinnedMessageID]
			if exists {
				pinMessageToSave, importErr := m.createPinMessageFromDiscordMessage(messageToSave, pinnedMessageID, channel.ID, community)
				if importErr != nil {
					importProgress.AddTaskError(discord.ImportMessagesTask, importErr)
					progressUpdates <- importProgress
					continue
				}

				pinMessagesToSave = append(pinMessagesToSave, pinMessageToSave)

				// Generate SystemMessagePinnedMessage
				systemMessage, importErr := m.generateSystemPinnedMessage(pinMessageToSave, channel, clockAndTimestamp, pinnedMessageID)
				if importErr != nil {
					importProgress.AddTaskError(discord.ImportMessagesTask, importErr)
					progressUpdates <- importProgress
					continue
				}

				messagesToSave[systemMessage.ID] = systemMessage
			}
		} else {
			messagesToSave[messageToSave.ID] = messageToSave
		}
	}

	return messagesToSave, pinMessagesToSave, authorProfilesToSave, messageAttachmentsToDownload
}

func (m *Messenger) RequestImportDiscordChannel(request *requests.ImportDiscordChannel) {
	go func() {
		totalImportChunkCount := len(request.FilesToImport)

		progressUpdates := make(chan *discord.ImportProgress)

		done := make(chan struct{})
		cancel := make(chan string)

		var newChat *Chat

		m.startPublishImportProgressInterval(progressUpdates, cancel, done)

		importProgress := &discord.ImportProgress{}
		importProgress.Init(totalImportChunkCount, []discord.ImportTask{
			discord.ChannelsCreationTask,
			discord.ImportMessagesTask,
			discord.DownloadAssetsTask,
		})

		importProgress.CommunityID = request.DiscordChannelID
		importProgress.CommunityName = request.Name
		// initial progress immediately

		if err := request.Validate(); err != nil {
			errmsg := fmt.Sprintf("Request validation failed: '%s'", err.Error())
			importProgress.AddTaskError(discord.ChannelsCreationTask, discord.Error(errmsg))
			importProgress.StopTask(discord.ChannelsCreationTask)
			progressUpdates <- importProgress
			cancel <- request.DiscordChannelID
			return
		}

		// Here's 3 steps: Find the corrent channel in files, get the community and create the channel
		progressValue := calculateProgress(3, 1, (float32(1) / 3))

		m.publishImportProgress(importProgress)

		community, err := m.GetCommunityByID(request.CommunityID)

		if err != nil {
			errmsg := fmt.Sprintf("Couldn't get the community '%s': '%s'", request.CommunityID, err.Error())
			importProgress.AddTaskError(discord.ChannelsCreationTask, discord.Error(errmsg))
			importProgress.StopTask(discord.ChannelsCreationTask)
			progressUpdates <- importProgress
			cancel <- request.DiscordChannelID
			return
		}

		if community == nil {
			errmsg := fmt.Sprintf("Couldn't get the community by id: '%s'", request.CommunityID)
			importProgress.AddTaskError(discord.ChannelsCreationTask, discord.Error(errmsg))
			importProgress.StopTask(discord.ChannelsCreationTask)
			progressUpdates <- importProgress
			cancel <- request.DiscordChannelID
			return
		}

		importProgress.UpdateTaskProgress(discord.ChannelsCreationTask, progressValue)
		progressUpdates <- importProgress

		for i, importFile := range request.FilesToImport {
			m.importingChannels[request.DiscordChannelID] = false

			exportData, errs := m.ExtractDiscordDataFromImportFiles([]string{importFile})
			if len(errs) > 0 {
				for _, err := range errs {
					importProgress.AddTaskError(discord.ChannelsCreationTask, err)
				}
				importProgress.StopTask(discord.ChannelsCreationTask)
				progressUpdates <- importProgress
				cancel <- request.DiscordChannelID
				return
			}

			var channel *discord.ExportedData

			for _, ch := range exportData.ExportedData {
				if ch.Channel.ID == request.DiscordChannelID {
					channel = ch
				}
			}

			if channel == nil {
				if i < len(request.FilesToImport)-1 {
					// skip this file
					continue
				} else if i == len(request.FilesToImport)-1 {
					errmsg := fmt.Sprintf("Couldn't find the target channel id in files: '%s'", request.DiscordChannelID)
					importProgress.AddTaskError(discord.ChannelsCreationTask, discord.Error(errmsg))
					importProgress.StopTask(discord.ChannelsCreationTask)
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID
					return
				}
			}
			progressValue := calculateProgress(3, 2, (float32(2) / 3))

			importProgress.UpdateTaskProgress(discord.ChannelsCreationTask, progressValue)
			progressUpdates <- importProgress

			if len(channel.Channel.ID) == 0 {
				// skip this file and try to find in the next file
				continue
			}
			exists := false

			for _, chatID := range community.ChatIDs() {
				if strings.HasSuffix(chatID, request.DiscordChannelID) {
					exists = true
					break
				}
			}

			if !exists {

				communityChat := &protobuf.CommunityChat{
					Permissions: &protobuf.CommunityPermissions{
						Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
					},
					Identity: &protobuf.ChatIdentity{
						DisplayName: request.Name,
						Emoji:       request.Emoji,
						Description: request.Description,
						Color:       request.Color,
					},
					CategoryId: "",
				}

				changes, err := m.communitiesManager.CreateChat(request.CommunityID, communityChat, false, channel.Channel.ID)
				if err != nil {
					errmsg := err.Error()
					if errors.Is(err, communities.ErrInvalidCommunityDescriptionDuplicatedName) {
						errmsg = fmt.Sprintf("Couldn't create channel '%s': %s", communityChat.Identity.DisplayName, err.Error())
					}

					importProgress.AddTaskError(discord.ChannelsCreationTask, discord.Error(errmsg))
					importProgress.StopTask(discord.ChannelsCreationTask)
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID
					return
				}

				community = changes.Community
				for chatID, chat := range changes.ChatsAdded {
					newChat = CreateCommunityChat(request.CommunityID.String(), chatID, chat, m.getTimesource())
				}

				progressValue = calculateProgress(3, 3, (float32(3) / 3))

				importProgress.UpdateTaskProgress(discord.ChannelsCreationTask, progressValue)
				progressUpdates <- importProgress
			}

			if m.DiscordImportChannelMarkedAsCancelled(request.DiscordChannelID) {
				importProgress.StopTask(discord.ImportMessagesTask)
				progressUpdates <- importProgress
				cancel <- request.DiscordChannelID
				return
			}

			messagesToSave, pinMessagesToSave, authorProfilesToSave, messageAttachmentsToDownload :=
				m.processDiscordMessages(channel, newChat, importProgress, progressUpdates, request.From, community)

			var discordMessages []*protobuf.DiscordMessage
			for _, msg := range messagesToSave {
				if msg.ChatMessage.ContentType == protobuf.ChatMessage_DISCORD_MESSAGE {
					discordMessages = append(discordMessages, msg.GetDiscordMessage())
				}
			}

			// We save these messages in chunks, so we don't block the database
			// for a longer period of time
			discordMessageChunks := chunkSlice(discordMessages, maxChunkSizeMessages)
			chunksCount := len(discordMessageChunks)

			for ii, msgs := range discordMessageChunks {
				m.communitiesManager.LogStdout(fmt.Sprintf("saving %d/%d chunk with %d discord messages", ii+1, chunksCount, len(msgs)))
				err := m.persistence.SaveDiscordMessages(msgs)
				if err != nil {
					m.cleanUpImportChannel(request.CommunityID.String(), newChat.ID)
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID

					return
				}

				if m.DiscordImportChannelMarkedAsCancelled(request.DiscordChannelID) {
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID
					return
				}

				// We're multiplying `chunksCount` by `0.25` so we leave 25% for additional save operations
				// 0.5 are the previous 50% of progress
				currentCount := ii + 1
				progressValue := calculateProgress(i+1, totalImportChunkCount, 0.5+(float32(currentCount)/float32(chunksCount))*0.25)
				importProgress.UpdateTaskProgress(discord.ImportMessagesTask, progressValue)
				progressUpdates <- importProgress

				// We slow down the saving of message chunks to keep the database responsive
				if currentCount < chunksCount {
					time.Sleep(2 * time.Second)
				}
			}

			// Get slice of all values in `messagesToSave` map
			var messages = make([]*common.Message, 0, len(messagesToSave))
			for _, msg := range messagesToSave {
				messages = append(messages, msg)
			}

			// Same as above, we save these messages in chunks so we don't block
			// the database for a longer period of time
			messageChunks := chunkSlice(messages, maxChunkSizeMessages)
			chunksCount = len(messageChunks)

			for ii, msgs := range messageChunks {
				m.communitiesManager.LogStdout(fmt.Sprintf("saving %d/%d chunk with %d app messages", ii+1, chunksCount, len(msgs)))
				err := m.persistence.SaveMessages(msgs)
				if err != nil {
					m.cleanUpImportChannel(request.CommunityID.String(), request.DiscordChannelID)
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID

					return
				}

				if m.DiscordImportChannelMarkedAsCancelled(request.DiscordChannelID) {
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID
					return
				}

				// 0.75 are the previous 75% of progress, hence we multiply our chunk progress
				// by 0.25
				currentCount := ii + 1
				progressValue := calculateProgress(i+1, totalImportChunkCount, 0.75+(float32(currentCount)/float32(chunksCount))*0.25)
				// progressValue := 0.75 + ((float32(currentCount) / float32(chunksCount)) * 0.25)
				importProgress.UpdateTaskProgress(discord.ImportMessagesTask, progressValue)
				progressUpdates <- importProgress

				// We slow down the saving of message chunks to keep the database responsive
				if currentCount < chunksCount {
					time.Sleep(2 * time.Second)
				}
			}

			pinMessageChunks := chunkSlice(pinMessagesToSave, maxChunkSizeMessages)
			for _, pinMsgs := range pinMessageChunks {
				err := m.persistence.SavePinMessages(pinMsgs)
				if err != nil {
					m.cleanUpImportChannel(request.CommunityID.String(), request.DiscordChannelID)
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID

					return
				}

				if m.DiscordImportChannelMarkedAsCancelled(request.DiscordChannelID) {
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID
					cancel <- request.DiscordChannelID

					return
				}
			}

			totalAssetsCount := len(messageAttachmentsToDownload) + len(authorProfilesToSave)
			var assetCounter discord.AssetCounter

			var wg sync.WaitGroup

			for id, author := range authorProfilesToSave {
				wg.Add(1)
				go func(id string, author *protobuf.DiscordMessageAuthor) {
					defer wg.Done()

					m.communitiesManager.LogStdout(fmt.Sprintf("downloading asset %d/%d", assetCounter.Value()+1, totalAssetsCount))
					imagePayload, err := discord.DownloadAvatarAsset(author.AvatarUrl)
					if err != nil {
						errmsg := fmt.Sprintf("Couldn't download profile avatar '%s': %s", author.AvatarUrl, err.Error())
						importProgress.AddTaskError(
							discord.DownloadAssetsTask,
							discord.Warning(errmsg),
						)
						progressUpdates <- importProgress
						cancel <- request.DiscordChannelID

						return
					}

					err = m.persistence.UpdateDiscordMessageAuthorImage(author.Id, imagePayload)
					if err != nil {
						importProgress.AddTaskError(discord.DownloadAssetsTask, discord.Warning(err.Error()))
						progressUpdates <- importProgress
						cancel <- request.DiscordChannelID

						return
					}

					author.AvatarImagePayload = imagePayload
					authorProfilesToSave[id] = author

					if m.DiscordImportMarkedAsCancelled(request.DiscordChannelID) {
						importProgress.StopTask(discord.DownloadAssetsTask)
						progressUpdates <- importProgress
						cancel <- request.DiscordChannelID
						return
					}

					assetCounter.Increase()
					progressValue := calculateProgress(i+1, totalImportChunkCount, (float32(assetCounter.Value())/float32(totalAssetsCount))*0.5)
					importProgress.UpdateTaskProgress(discord.DownloadAssetsTask, progressValue)
					progressUpdates <- importProgress

				}(id, author)
			}
			wg.Wait()

			if m.DiscordImportChannelMarkedAsCancelled(request.DiscordChannelID) {
				importProgress.StopTask(discord.DownloadAssetsTask)
				progressUpdates <- importProgress
				cancel <- request.DiscordChannelID
				return
			}

			for idxRange := range gopart.Partition(len(messageAttachmentsToDownload), 100) {
				attachments := messageAttachmentsToDownload[idxRange.Low:idxRange.High]
				wg.Add(1)
				go func(attachments []*protobuf.DiscordMessageAttachment) {
					defer wg.Done()
					for ii, attachment := range attachments {

						m.communitiesManager.LogStdout(fmt.Sprintf("downloading asset %d/%d", assetCounter.Value()+1, totalAssetsCount))

						assetPayload, contentType, err := discord.DownloadAsset(attachment.Url)
						if err != nil {
							errmsg := fmt.Sprintf("Couldn't download message attachment '%s': %s", attachment.Url, err.Error())
							importProgress.AddTaskError(
								discord.DownloadAssetsTask,
								discord.Warning(errmsg),
							)
							progressUpdates <- importProgress
							continue
						}

						attachment.Payload = assetPayload
						attachment.ContentType = contentType
						messageAttachmentsToDownload[ii] = attachment

						if m.DiscordImportChannelMarkedAsCancelled(request.DiscordChannelID) {
							importProgress.StopTask(discord.DownloadAssetsTask)
							progressUpdates <- importProgress
							cancel <- request.DiscordChannelID
							return
						}

						assetCounter.Increase()
						progressValue := calculateProgress(i+1, totalImportChunkCount, (float32(assetCounter.Value())/float32(totalAssetsCount))*0.5)
						importProgress.UpdateTaskProgress(discord.DownloadAssetsTask, progressValue)
						progressUpdates <- importProgress
					}
				}(attachments)
			}
			wg.Wait()

			if m.DiscordImportChannelMarkedAsCancelled(request.DiscordChannelID) {
				importProgress.StopTask(discord.DownloadAssetsTask)
				progressUpdates <- importProgress
				cancel <- request.DiscordChannelID
				return
			}

			attachmentChunks := chunkAttachmentsByByteSize(messageAttachmentsToDownload, maxChunkSizeBytes)
			chunksCount = len(attachmentChunks)

			for ii, attachments := range attachmentChunks {
				m.communitiesManager.LogStdout(fmt.Sprintf("saving %d/%d chunk with %d discord message attachments", ii+1, chunksCount, len(attachments)))
				err := m.persistence.SaveDiscordMessageAttachments(attachments)
				if err != nil {
					m.cleanUpImportChannel(request.CommunityID.String(), request.DiscordChannelID)
					importProgress.AddTaskError(discord.DownloadAssetsTask, discord.Error(err.Error()))
					importProgress.Stop()
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID

					return
				}

				if m.DiscordImportChannelMarkedAsCancelled(request.DiscordChannelID) {
					importProgress.StopTask(discord.DownloadAssetsTask)
					progressUpdates <- importProgress
					cancel <- request.DiscordChannelID
					return
				}

				// 0.5 are the previous 50% of progress, hence we multiply our chunk progress
				// by 0.5
				currentCount := ii + 1
				progressValue := calculateProgress(i+1, totalImportChunkCount, 0.5+(float32(currentCount)/float32(chunksCount))*0.5)
				importProgress.UpdateTaskProgress(discord.DownloadAssetsTask, progressValue)
				progressUpdates <- importProgress

				// We slow down the saving of attachment chunks to keep the database responsive
				if currentCount < chunksCount {
					time.Sleep(2 * time.Second)
				}
			}

			if len(attachmentChunks) == 0 {
				progressValue := calculateProgress(i+1, totalImportChunkCount, 1.0)
				importProgress.UpdateTaskProgress(discord.DownloadAssetsTask, progressValue)
			}

			_, err := m.transport.JoinPublic(newChat.ID)
			if err != nil {
				m.logger.Error("failed to load filter for chat", zap.Error(err))
				continue
			}

			wakuChatMessages, err := m.chatMessagesToWakuMessages(messages, community)
			if err != nil {
				m.logger.Error("failed to convert chat messages into waku messages", zap.Error(err))
				continue
			}

			wakuPinMessages, err := m.pinMessagesToWakuMessages(pinMessagesToSave, community)
			if err != nil {
				m.logger.Error("failed to convert pin messages into waku messages", zap.Error(err))
				continue
			}

			wakuMessages := append(wakuChatMessages, wakuPinMessages...)

			topics, err := m.communitiesManager.GetCommunityChatsTopics(request.CommunityID)
			if err != nil {
				m.logger.Error("failed to get community chat topics", zap.Error(err))
				continue
			}

			startDate := time.Unix(int64(exportData.OldestMessageTimestamp), 0)
			endDate := time.Now()

			_, err = m.communitiesManager.CreateHistoryArchiveTorrentFromMessages(
				request.CommunityID,
				wakuMessages,
				topics,
				startDate,
				endDate,
				messageArchiveInterval,
				community.Encrypted(),
			)
			if err != nil {
				m.logger.Error("failed to create history archive torrent", zap.Error(err))
				continue
			}
			communitySettings, err := m.communitiesManager.GetCommunitySettingsByID(request.CommunityID)
			if err != nil {
				m.logger.Error("Failed to get community settings", zap.Error(err))
				continue
			}
			if m.torrentClientReady() && communitySettings.HistoryArchiveSupportEnabled {

				err = m.communitiesManager.SeedHistoryArchiveTorrent(request.CommunityID)
				if err != nil {
					m.logger.Error("failed to seed history archive", zap.Error(err))
				}
				go m.communitiesManager.StartHistoryArchiveTasksInterval(community, messageArchiveInterval)
			}
		}
		if m.DiscordImportChannelMarkedAsCancelled(request.DiscordChannelID) {
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			cancel <- request.DiscordChannelID
			return
		}

		// Chats need to be saved after the community has been published,
		// hence we make this part of the `InitCommunityTask`
		err = m.saveChat(newChat)

		if err != nil {
			m.cleanUpImportChannel(request.CommunityID.String(), request.DiscordChannelID)
			importProgress.AddTaskError(discord.InitCommunityTask, discord.Error(err.Error()))
			importProgress.Stop()
			progressUpdates <- importProgress
			cancel <- request.DiscordChannelID
			return
		}

		m.config.messengerSignalsHandler.DiscordCommunityImportFinished(request.DiscordChannelID)
		close(done)
	}()
}

func (m *Messenger) RequestImportDiscordCommunity(request *requests.ImportDiscordCommunity) {
	go func() {

		totalImportChunkCount := len(request.FilesToImport)

		progressUpdates := make(chan *discord.ImportProgress)
		done := make(chan struct{})
		cancel := make(chan string)
		m.startPublishImportProgressInterval(progressUpdates, cancel, done)

		importProgress := &discord.ImportProgress{}
		importProgress.Init(totalImportChunkCount, []discord.ImportTask{
			discord.CommunityCreationTask,
			discord.ChannelsCreationTask,
			discord.ImportMessagesTask,
			discord.DownloadAssetsTask,
			discord.InitCommunityTask,
		})
		importProgress.CommunityName = request.Name

		// initial progress immediately
		m.publishImportProgress(importProgress)

		createCommunityRequest := request.ToCreateCommunityRequest()

		// We're calling `CreateCommunity` on `communitiesManager` directly, instead of
		// using the `Messenger` API, so we get more control over when we set up filters,
		// the community is published and data is being synced (we don't want the community
		// to show up in clients while the import is in progress)
		discordCommunity, err := m.communitiesManager.CreateCommunity(createCommunityRequest, false)
		if err != nil {
			importProgress.AddTaskError(discord.CommunityCreationTask, discord.Error(err.Error()))
			importProgress.StopTask(discord.CommunityCreationTask)
			progressUpdates <- importProgress
			return
		}

		communitySettings := communities.CommunitySettings{
			CommunityID:                  discordCommunity.IDString(),
			HistoryArchiveSupportEnabled: true,
		}
		err = m.communitiesManager.SaveCommunitySettings(communitySettings)
		if err != nil {
			m.cleanUpImport(discordCommunity.IDString())
			importProgress.AddTaskError(discord.CommunityCreationTask, discord.Error(err.Error()))
			importProgress.StopTask(discord.CommunityCreationTask)
			progressUpdates <- importProgress
			return
		}

		communityID := discordCommunity.IDString()

		// marking import as not cancelled
		m.importingCommunities[communityID] = false
		importProgress.CommunityID = communityID
		importProgress.CommunityImages = make(map[string]images.IdentityImage)

		imgs := discordCommunity.Images()
		for t, i := range imgs {
			importProgress.CommunityImages[t] = images.IdentityImage{Name: t, Payload: i.Payload}
		}

		importProgress.UpdateTaskProgress(discord.CommunityCreationTask, 1)
		progressUpdates <- importProgress

		if m.DiscordImportMarkedAsCancelled(communityID) {
			importProgress.StopTask(discord.CommunityCreationTask)
			progressUpdates <- importProgress
			cancel <- communityID
			return
		}

		var chatsToSave []*Chat
		createdChats := make(map[string]*Chat, 0)
		processedChannelIds := make(map[string]string, 0)
		processedCategoriesIds := make(map[string]string, 0)

		for i, importFile := range request.FilesToImport {

			exportData, errs := m.ExtractDiscordDataFromImportFiles([]string{importFile})
			if len(errs) > 0 {
				for _, err := range errs {
					importProgress.AddTaskError(discord.CommunityCreationTask, err)
				}
				progressUpdates <- importProgress
				return
			}
			totalChannelsCount := len(exportData.ExportedData)
			totalMessageCount := exportData.MessageCount

			if totalChannelsCount == 0 || totalMessageCount == 0 {
				importError := discord.Error(fmt.Errorf("No channel to import messages from in file: %s", importFile).Error())
				if totalMessageCount == 0 {
					importError.Message = fmt.Errorf("No messages to import in file: %s", importFile).Error()
				}
				importProgress.AddTaskError(discord.ChannelsCreationTask, importError)
				progressUpdates <- importProgress
				continue
			}

			importProgress.CurrentChunk = i + 1

			// We actually only ever receive a single category
			// from `exportData` but since it's a map, we still have to
			// iterate over it to access its values
			for _, category := range exportData.Categories {

				categories := discordCommunity.Categories()
				exists := false
				for catID := range categories {
					if strings.HasSuffix(catID, category.ID) {
						exists = true
						break
					}
				}

				if !exists {
					createCommunityCategoryRequest := &requests.CreateCommunityCategory{
						CommunityID:  discordCommunity.ID(),
						CategoryName: category.Name,
						ThirdPartyID: category.ID,
						ChatIDs:      make([]string, 0),
					}
					// We call `CreateCategory` on `communitiesManager` directly so we can control
					// whether or not the community update should be published (it should not until the
					// import has finished)
					communityWithCategories, changes, err := m.communitiesManager.CreateCategory(createCommunityCategoryRequest, false)
					if err != nil {
						m.cleanUpImport(communityID)
						importProgress.AddTaskError(discord.CommunityCreationTask, discord.Error(err.Error()))
						importProgress.StopTask(discord.CommunityCreationTask)
						progressUpdates <- importProgress
						return
					}
					discordCommunity = communityWithCategories
					// This looks like we keep overriding the same field but there's
					// only one `CategoriesAdded` change at this point.
					for _, addedCategory := range changes.CategoriesAdded {
						processedCategoriesIds[category.ID] = addedCategory.CategoryId
					}
				}
			}

			progressValue := calculateProgress(i+1, totalImportChunkCount, (float32(1) / 2))
			importProgress.UpdateTaskProgress(discord.ChannelsCreationTask, progressValue)

			progressUpdates <- importProgress

			if m.DiscordImportMarkedAsCancelled(communityID) {
				importProgress.StopTask(discord.CommunityCreationTask)
				progressUpdates <- importProgress
				cancel <- communityID
				return
			}

			messagesToSave := make(map[string]*common.Message, 0)
			pinMessagesToSave := make([]*common.PinMessage, 0)
			authorProfilesToSave := make(map[string]*protobuf.DiscordMessageAuthor, 0)
			messageAttachmentsToDownload := make([]*protobuf.DiscordMessageAttachment, 0)

			// Save to access the first item here as we process
			// exported data by files which only holds a single channel
			channel := exportData.ExportedData[0]
			chatIDs := discordCommunity.ChatIDs()

			exists := false
			for _, chatID := range chatIDs {
				if strings.HasSuffix(chatID, channel.Channel.ID) {
					exists = true
					break
				}
			}

			if !exists {
				communityChat := &protobuf.CommunityChat{
					Permissions: &protobuf.CommunityPermissions{
						Access: protobuf.CommunityPermissions_AUTO_ACCEPT,
					},
					Identity: &protobuf.ChatIdentity{
						DisplayName: channel.Channel.Name,
						Emoji:       "",
						Description: channel.Channel.Description,
						Color:       discordCommunity.Color(),
					},
					CategoryId: processedCategoriesIds[channel.Channel.CategoryID],
				}

				// We call `CreateChat` on `communitiesManager` directly to get more control
				// over whether we want to publish the updated community description.
				changes, err := m.communitiesManager.CreateChat(discordCommunity.ID(), communityChat, false, channel.Channel.ID)
				if err != nil {
					m.cleanUpImport(communityID)
					errmsg := err.Error()
					if errors.Is(err, communities.ErrInvalidCommunityDescriptionDuplicatedName) {
						errmsg = fmt.Sprintf("Couldn't create channel '%s': %s", communityChat.Identity.DisplayName, err.Error())
					}
					importProgress.AddTaskError(discord.ChannelsCreationTask, discord.Error(errmsg))
					importProgress.StopTask(discord.ChannelsCreationTask)
					progressUpdates <- importProgress
					return
				}
				discordCommunity = changes.Community

				// This looks like we keep overriding the chat id value
				// as we iterate over `ChatsAdded`, however at this point we
				// know there was only a single such change (and it's a map)
				for chatID, chat := range changes.ChatsAdded {
					c := CreateCommunityChat(communityID, chatID, chat, m.getTimesource())
					createdChats[c.ID] = c
					chatsToSave = append(chatsToSave, c)
					processedChannelIds[channel.Channel.ID] = c.ID
				}
			}

			progressValue = calculateProgress(i+1, totalImportChunkCount, 1)
			importProgress.UpdateTaskProgress(discord.ChannelsCreationTask, progressValue)
			progressUpdates <- importProgress

			for ii, discordMessage := range channel.Messages {

				timestamp, err := time.Parse(discordTimestampLayout, discordMessage.Timestamp)
				if err != nil {
					m.logger.Error("failed to parse discord message timestamp", zap.Error(err))
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Warning(err.Error()))
					progressUpdates <- importProgress
					continue
				}

				if timestamp.Unix() < request.From {
					progressUpdates <- importProgress
					continue
				}

				exists, err := m.persistence.HasDiscordMessageAuthor(discordMessage.Author.GetId())
				if err != nil {
					m.logger.Error("failed to check if message author exists in database", zap.Error(err))
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
					progressUpdates <- importProgress
					continue
				}

				if !exists {
					err := m.persistence.SaveDiscordMessageAuthor(discordMessage.Author)
					if err != nil {
						importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
						progressUpdates <- importProgress
						continue
					}
				}

				hasPayload, err := m.persistence.HasDiscordMessageAuthorImagePayload(discordMessage.Author.GetId())
				if err != nil {
					m.logger.Error("failed to check if message avatar payload exists in database", zap.Error(err))
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
					progressUpdates <- importProgress
					continue
				}

				if !hasPayload {
					authorProfilesToSave[discordMessage.Author.Id] = discordMessage.Author
				}

				// Convert timestamp to unix timestamp
				discordMessage.Timestamp = fmt.Sprintf("%d", timestamp.Unix())

				if discordMessage.TimestampEdited != "" {
					timestampEdited, err := time.Parse(discordTimestampLayout, discordMessage.TimestampEdited)
					if err != nil {
						m.logger.Error("failed to parse discord message timestamp", zap.Error(err))
						importProgress.AddTaskError(discord.ImportMessagesTask, discord.Warning(err.Error()))
						progressUpdates <- importProgress
						continue
					}
					// Convert timestamp to unix timestamp
					discordMessage.TimestampEdited = fmt.Sprintf("%d", timestampEdited.Unix())
				}

				for i := range discordMessage.Attachments {
					discordMessage.Attachments[i].MessageId = discordMessage.Id
				}
				messageAttachmentsToDownload = append(messageAttachmentsToDownload, discordMessage.Attachments...)

				clockAndTimestamp := uint64(timestamp.Unix()) * 1000
				communityPubKey := discordCommunity.PrivateKey().PublicKey

				chatMessage := protobuf.ChatMessage{
					Timestamp:   clockAndTimestamp,
					MessageType: protobuf.MessageType_COMMUNITY_CHAT,
					ContentType: protobuf.ChatMessage_DISCORD_MESSAGE,
					Clock:       clockAndTimestamp,
					ChatId:      processedChannelIds[channel.Channel.ID],
					Payload: &protobuf.ChatMessage_DiscordMessage{
						DiscordMessage: discordMessage,
					},
				}

				// Handle message replies
				if discordMessage.Type == string(discord.MessageTypeReply) && discordMessage.Reference != nil {
					repliedMessageID := communityID + discordMessage.Reference.MessageId
					if _, exists := messagesToSave[repliedMessageID]; exists {
						chatMessage.ResponseTo = repliedMessageID
					}
				}

				messageToSave := &common.Message{
					ID:               communityID + discordMessage.Id,
					WhisperTimestamp: clockAndTimestamp,
					From:             types.EncodeHex(crypto.FromECDSAPub(&communityPubKey)),
					Seen:             true,
					LocalChatID:      processedChannelIds[channel.Channel.ID],
					SigPubKey:        &communityPubKey,
					CommunityID:      communityID,
					ChatMessage:      &chatMessage,
				}

				err = messageToSave.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
				if err != nil {
					m.logger.Error("failed to prepare message content", zap.Error(err))
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
					progressUpdates <- importProgress
					continue
				}

				// Handle pin messages
				if discordMessage.Type == string(discord.MessageTypeChannelPinned) && discordMessage.Reference != nil {

					pinnedMessageID := communityID + discordMessage.Reference.MessageId
					_, exists := messagesToSave[pinnedMessageID]
					if exists {
						pinMessage := protobuf.PinMessage{
							Clock:       messageToSave.WhisperTimestamp,
							MessageId:   pinnedMessageID,
							ChatId:      messageToSave.LocalChatID,
							MessageType: protobuf.MessageType_COMMUNITY_CHAT,
							Pinned:      true,
						}

						encodedPayload, err := proto.Marshal(&pinMessage)
						if err != nil {
							m.logger.Error("failed to parse marshal pin message", zap.Error(err))
							importProgress.AddTaskError(discord.ImportMessagesTask, discord.Warning(err.Error()))
							progressUpdates <- importProgress
							continue
						}

						wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, protobuf.ApplicationMetadataMessage_PIN_MESSAGE, discordCommunity.PrivateKey())
						if err != nil {
							m.logger.Error("failed to wrap pin message", zap.Error(err))
							importProgress.AddTaskError(discord.ImportMessagesTask, discord.Warning(err.Error()))
							progressUpdates <- importProgress
							continue
						}

						pinMessageToSave := common.PinMessage{
							ID:               types.EncodeHex(v1protocol.MessageID(&communityPubKey, wrappedPayload)),
							PinMessage:       &pinMessage,
							LocalChatID:      processedChannelIds[channel.Channel.ID],
							From:             messageToSave.From,
							SigPubKey:        messageToSave.SigPubKey,
							WhisperTimestamp: messageToSave.WhisperTimestamp,
						}

						pinMessagesToSave = append(pinMessagesToSave, &pinMessageToSave)

						// Generate SystemMessagePinnedMessage

						chat, ok := createdChats[pinMessageToSave.LocalChatID]
						if !ok {
							err := errors.New("failed to get chat for pin message")
							m.logger.Warn(err.Error(),
								zap.String("PinMessageId", pinMessageToSave.ID),
								zap.String("ChatID", pinMessageToSave.LocalChatID))
							importProgress.AddTaskError(discord.ImportMessagesTask, discord.Warning(err.Error()))
							progressUpdates <- importProgress
							continue
						}

						id, err := generatePinMessageNotificationID(&m.identity.PublicKey, &pinMessageToSave, chat)
						if err != nil {
							m.logger.Warn("failed to generate pin message notification ID",
								zap.String("PinMessageId", pinMessageToSave.ID))
							importProgress.AddTaskError(discord.ImportMessagesTask, discord.Warning(err.Error()))
							progressUpdates <- importProgress
							continue
						}
						systemMessage := &common.Message{
							ChatMessage: &protobuf.ChatMessage{
								Clock:       pinMessageToSave.Clock,
								Timestamp:   clockAndTimestamp,
								ChatId:      chat.ID,
								MessageType: pinMessageToSave.MessageType,
								ResponseTo:  pinMessage.MessageId,
								ContentType: protobuf.ChatMessage_SYSTEM_MESSAGE_PINNED_MESSAGE,
							},
							WhisperTimestamp: clockAndTimestamp,
							ID:               id,
							LocalChatID:      chat.ID,
							From:             messageToSave.From,
							Seen:             true,
						}

						messagesToSave[systemMessage.ID] = systemMessage
					}
				} else {
					messagesToSave[messageToSave.ID] = messageToSave
				}

				progressValue := calculateProgress(i+1, totalImportChunkCount, float32(ii+1)/float32(len(channel.Messages))*0.5)
				importProgress.UpdateTaskProgress(discord.ImportMessagesTask, progressValue)
				progressUpdates <- importProgress
			}

			if m.DiscordImportMarkedAsCancelled(communityID) {
				importProgress.StopTask(discord.ImportMessagesTask)
				progressUpdates <- importProgress
				cancel <- communityID
				return
			}

			var discordMessages []*protobuf.DiscordMessage
			for _, msg := range messagesToSave {
				if msg.ChatMessage.ContentType == protobuf.ChatMessage_DISCORD_MESSAGE {
					discordMessages = append(discordMessages, msg.GetDiscordMessage())
				}
			}

			// We save these messages in chunks, so we don't block the database
			// for a longer period of time
			discordMessageChunks := chunkSlice(discordMessages, maxChunkSizeMessages)
			chunksCount := len(discordMessageChunks)

			for ii, msgs := range discordMessageChunks {
				m.communitiesManager.LogStdout(fmt.Sprintf("saving %d/%d chunk with %d discord messages", ii+1, chunksCount, len(msgs)))
				err = m.persistence.SaveDiscordMessages(msgs)
				if err != nil {
					m.cleanUpImport(communityID)
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					return
				}

				if m.DiscordImportMarkedAsCancelled(communityID) {
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					cancel <- communityID
					return
				}

				// We're multiplying `chunksCount` by `0.25` so we leave 25% for additional save operations
				// 0.5 are the previous 50% of progress
				currentCount := ii + 1
				progressValue := calculateProgress(i+1, totalImportChunkCount, 0.5+(float32(currentCount)/float32(chunksCount))*0.25)
				importProgress.UpdateTaskProgress(discord.ImportMessagesTask, progressValue)
				progressUpdates <- importProgress

				// We slow down the saving of message chunks to keep the database responsive
				if currentCount < chunksCount {
					time.Sleep(2 * time.Second)
				}
			}

			// Get slice of all values in `messagesToSave` map

			var messages = make([]*common.Message, 0, len(messagesToSave))
			for _, msg := range messagesToSave {
				messages = append(messages, msg)
			}

			// Same as above, we save these messages in chunks so we don't block
			// the database for a longer period of time
			messageChunks := chunkSlice(messages, maxChunkSizeMessages)
			chunksCount = len(messageChunks)

			for ii, msgs := range messageChunks {
				m.communitiesManager.LogStdout(fmt.Sprintf("saving %d/%d chunk with %d app messages", ii+1, chunksCount, len(msgs)))
				err = m.persistence.SaveMessages(msgs)
				if err != nil {
					m.cleanUpImport(communityID)
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					return
				}

				if m.DiscordImportMarkedAsCancelled(communityID) {
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					cancel <- communityID
					return
				}

				// 0.75 are the previous 75% of progress, hence we multiply our chunk progress
				// by 0.25
				currentCount := ii + 1
				progressValue := calculateProgress(i+1, totalImportChunkCount, 0.75+(float32(currentCount)/float32(chunksCount))*0.25)
				// progressValue := 0.75 + ((float32(currentCount) / float32(chunksCount)) * 0.25)
				importProgress.UpdateTaskProgress(discord.ImportMessagesTask, progressValue)
				progressUpdates <- importProgress

				// We slow down the saving of message chunks to keep the database responsive
				if currentCount < chunksCount {
					time.Sleep(2 * time.Second)
				}
			}

			pinMessageChunks := chunkSlice(pinMessagesToSave, maxChunkSizeMessages)
			for _, pinMsgs := range pinMessageChunks {
				err = m.persistence.SavePinMessages(pinMsgs)
				if err != nil {
					m.cleanUpImport(communityID)
					importProgress.AddTaskError(discord.ImportMessagesTask, discord.Error(err.Error()))
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					return
				}

				if m.DiscordImportMarkedAsCancelled(communityID) {
					importProgress.StopTask(discord.ImportMessagesTask)
					progressUpdates <- importProgress
					cancel <- communityID
					return
				}
			}

			totalAssetsCount := len(messageAttachmentsToDownload) + len(authorProfilesToSave)
			var assetCounter discord.AssetCounter

			var wg sync.WaitGroup

			for id, author := range authorProfilesToSave {
				wg.Add(1)
				go func(id string, author *protobuf.DiscordMessageAuthor) {
					defer wg.Done()

					m.communitiesManager.LogStdout(fmt.Sprintf("downloading asset %d/%d", assetCounter.Value()+1, totalAssetsCount))
					imagePayload, err := discord.DownloadAvatarAsset(author.AvatarUrl)
					if err != nil {
						errmsg := fmt.Sprintf("Couldn't download profile avatar '%s': %s", author.AvatarUrl, err.Error())
						importProgress.AddTaskError(
							discord.DownloadAssetsTask,
							discord.Warning(errmsg),
						)
						progressUpdates <- importProgress
						return
					}

					err = m.persistence.UpdateDiscordMessageAuthorImage(author.Id, imagePayload)
					if err != nil {
						importProgress.AddTaskError(discord.DownloadAssetsTask, discord.Warning(err.Error()))
						progressUpdates <- importProgress
						return
					}

					author.AvatarImagePayload = imagePayload
					authorProfilesToSave[id] = author

					if m.DiscordImportMarkedAsCancelled(discordCommunity.IDString()) {
						importProgress.StopTask(discord.DownloadAssetsTask)
						progressUpdates <- importProgress
						cancel <- discordCommunity.IDString()
						return
					}

					assetCounter.Increase()
					progressValue := calculateProgress(i+1, totalImportChunkCount, (float32(assetCounter.Value())/float32(totalAssetsCount))*0.5)
					importProgress.UpdateTaskProgress(discord.DownloadAssetsTask, progressValue)
					progressUpdates <- importProgress

				}(id, author)
			}
			wg.Wait()

			if m.DiscordImportMarkedAsCancelled(communityID) {
				importProgress.StopTask(discord.DownloadAssetsTask)
				progressUpdates <- importProgress
				cancel <- communityID
				return
			}

			for idxRange := range gopart.Partition(len(messageAttachmentsToDownload), 100) {
				attachments := messageAttachmentsToDownload[idxRange.Low:idxRange.High]
				wg.Add(1)
				go func(attachments []*protobuf.DiscordMessageAttachment) {
					defer wg.Done()
					for ii, attachment := range attachments {

						m.communitiesManager.LogStdout(fmt.Sprintf("downloading asset %d/%d", assetCounter.Value()+1, totalAssetsCount))

						assetPayload, contentType, err := discord.DownloadAsset(attachment.Url)
						if err != nil {
							errmsg := fmt.Sprintf("Couldn't download message attachment '%s': %s", attachment.Url, err.Error())
							importProgress.AddTaskError(
								discord.DownloadAssetsTask,
								discord.Warning(errmsg),
							)
							progressUpdates <- importProgress
							continue
						}

						attachment.Payload = assetPayload
						attachment.ContentType = contentType
						messageAttachmentsToDownload[ii] = attachment

						if m.DiscordImportMarkedAsCancelled(communityID) {
							importProgress.StopTask(discord.DownloadAssetsTask)
							progressUpdates <- importProgress
							cancel <- communityID
							return
						}

						assetCounter.Increase()
						progressValue := calculateProgress(i+1, totalImportChunkCount, (float32(assetCounter.Value())/float32(totalAssetsCount))*0.5)
						importProgress.UpdateTaskProgress(discord.DownloadAssetsTask, progressValue)
						progressUpdates <- importProgress
					}
				}(attachments)
			}
			wg.Wait()

			if m.DiscordImportMarkedAsCancelled(communityID) {
				importProgress.StopTask(discord.DownloadAssetsTask)
				progressUpdates <- importProgress
				cancel <- communityID
				return
			}

			attachmentChunks := chunkAttachmentsByByteSize(messageAttachmentsToDownload, maxChunkSizeBytes)
			chunksCount = len(attachmentChunks)

			for ii, attachments := range attachmentChunks {
				m.communitiesManager.LogStdout(fmt.Sprintf("saving %d/%d chunk with %d discord message attachments", ii+1, chunksCount, len(attachments)))
				err = m.persistence.SaveDiscordMessageAttachments(attachments)
				if err != nil {
					m.cleanUpImport(communityID)
					importProgress.AddTaskError(discord.DownloadAssetsTask, discord.Error(err.Error()))
					importProgress.Stop()
					progressUpdates <- importProgress
					return
				}

				if m.DiscordImportMarkedAsCancelled(communityID) {
					importProgress.StopTask(discord.DownloadAssetsTask)
					progressUpdates <- importProgress
					cancel <- communityID
					return
				}

				// 0.5 are the previous 50% of progress, hence we multiply our chunk progress
				// by 0.5
				currentCount := ii + 1
				progressValue := calculateProgress(i+1, totalImportChunkCount, 0.5+(float32(currentCount)/float32(chunksCount))*0.5)
				importProgress.UpdateTaskProgress(discord.DownloadAssetsTask, progressValue)
				progressUpdates <- importProgress

				// We slow down the saving of attachment chunks to keep the database responsive
				if currentCount < chunksCount {
					time.Sleep(2 * time.Second)
				}
			}

			if len(attachmentChunks) == 0 {
				progressValue := calculateProgress(i+1, totalImportChunkCount, 1.0)
				importProgress.UpdateTaskProgress(discord.DownloadAssetsTask, progressValue)
			}

			_, err := m.transport.JoinPublic(processedChannelIds[channel.Channel.ID])
			if err != nil {
				m.logger.Error("failed to load filter for chat", zap.Error(err))
				continue
			}

			wakuChatMessages, err := m.chatMessagesToWakuMessages(messages, discordCommunity)
			if err != nil {
				m.logger.Error("failed to convert chat messages into waku messages", zap.Error(err))
				continue
			}

			wakuPinMessages, err := m.pinMessagesToWakuMessages(pinMessagesToSave, discordCommunity)
			if err != nil {
				m.logger.Error("failed to convert pin messages into waku messages", zap.Error(err))
				continue
			}

			wakuMessages := append(wakuChatMessages, wakuPinMessages...)

			topics, err := m.communitiesManager.GetCommunityChatsTopics(discordCommunity.ID())
			if err != nil {
				m.logger.Error("failed to get community chat topics", zap.Error(err))
				continue
			}

			startDate := time.Unix(int64(exportData.OldestMessageTimestamp), 0)
			endDate := time.Now()

			_, err = m.communitiesManager.CreateHistoryArchiveTorrentFromMessages(
				discordCommunity.ID(),
				wakuMessages,
				topics,
				startDate,
				endDate,
				messageArchiveInterval,
				discordCommunity.Encrypted(),
			)
			if err != nil {
				m.logger.Error("failed to create history archive torrent", zap.Error(err))
				continue
			}

			if m.torrentClientReady() && communitySettings.HistoryArchiveSupportEnabled {

				err = m.communitiesManager.SeedHistoryArchiveTorrent(discordCommunity.ID())
				if err != nil {
					m.logger.Error("failed to seed history archive", zap.Error(err))
				}
				go m.communitiesManager.StartHistoryArchiveTasksInterval(discordCommunity, messageArchiveInterval)
			}
		}

		err = m.publishOrg(discordCommunity)
		if err != nil {
			m.cleanUpImport(communityID)
			importProgress.AddTaskError(discord.InitCommunityTask, discord.Error(err.Error()))
			importProgress.Stop()
			progressUpdates <- importProgress
			return
		}

		if m.DiscordImportMarkedAsCancelled(communityID) {
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			cancel <- communityID
			return
		}

		// Chats need to be saved after the community has been published,
		// hence we make this part of the `InitCommunityTask`
		err = m.saveChats(chatsToSave)
		if err != nil {
			m.cleanUpImport(communityID)
			importProgress.AddTaskError(discord.InitCommunityTask, discord.Error(err.Error()))
			importProgress.Stop()
			progressUpdates <- importProgress
			return
		}

		importProgress.UpdateTaskProgress(discord.InitCommunityTask, 0.15)
		progressUpdates <- importProgress

		if m.DiscordImportMarkedAsCancelled(communityID) {
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			cancel <- communityID
			return
		}

		// Init the community filter so we can receive messages on the community
		_, err = m.transport.InitCommunityFilters([]transport.CommunityFilterToInitialize{{
			Shard:   discordCommunity.Shard().TransportShard(),
			PrivKey: discordCommunity.PrivateKey(),
		}})
		if err != nil {
			m.cleanUpImport(communityID)
			importProgress.AddTaskError(discord.InitCommunityTask, discord.Error(err.Error()))
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			return
		}
		importProgress.UpdateTaskProgress(discord.InitCommunityTask, 0.25)
		progressUpdates <- importProgress

		if m.DiscordImportMarkedAsCancelled(communityID) {
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			cancel <- communityID
			return
		}

		_, err = m.transport.InitPublicFilters(discordCommunity.DefaultFilters())
		if err != nil {
			m.cleanUpImport(communityID)
			importProgress.AddTaskError(discord.InitCommunityTask, discord.Error(err.Error()))
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			return
		}

		importProgress.UpdateTaskProgress(discord.InitCommunityTask, 0.5)
		progressUpdates <- importProgress

		if m.DiscordImportMarkedAsCancelled(communityID) {
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			cancel <- communityID
			return
		}

		filters := m.transport.Filters()
		_, err = m.scheduleSyncFilters(filters)
		if err != nil {
			m.cleanUpImport(communityID)
			importProgress.AddTaskError(discord.InitCommunityTask, discord.Error(err.Error()))
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			return
		}
		importProgress.UpdateTaskProgress(discord.InitCommunityTask, 0.75)
		progressUpdates <- importProgress

		if m.DiscordImportMarkedAsCancelled(communityID) {
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			cancel <- communityID
			return
		}

		err = m.reregisterForPushNotifications()
		if err != nil {
			m.cleanUpImport(communityID)
			importProgress.AddTaskError(discord.InitCommunityTask, discord.Error(err.Error()))
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			return
		}
		importProgress.UpdateTaskProgress(discord.InitCommunityTask, 1)
		progressUpdates <- importProgress

		if m.DiscordImportMarkedAsCancelled(communityID) {
			importProgress.StopTask(discord.InitCommunityTask)
			progressUpdates <- importProgress
			cancel <- communityID
			return
		}

		m.config.messengerSignalsHandler.DiscordCommunityImportFinished(communityID)
		close(done)
	}()
}

func calculateProgress(i int, t int, currentProgress float32) float32 {
	current := float32(1) / float32(t) * currentProgress
	if i > 1 {
		return float32(i-1)/float32(t) + current
	}
	return current
}

func (m *Messenger) MarkDiscordCommunityImportAsCancelled(communityID string) {
	m.importingCommunities[communityID] = true
}

func (m *Messenger) MarkDiscordChannelImportAsCancelled(channelID string) {
	m.importingChannels[channelID] = true
}

func (m *Messenger) DiscordImportMarkedAsCancelled(communityID string) bool {
	cancelled, exists := m.importingCommunities[communityID]
	return exists && cancelled
}

func (m *Messenger) DiscordImportChannelMarkedAsCancelled(channelID string) bool {
	cancelled, exists := m.importingChannels[channelID]
	return exists && cancelled
}

func (m *Messenger) cleanUpImports() {
	for id := range m.importingCommunities {
		m.cleanUpImport(id)
	}
}

func (m *Messenger) cleanUpImport(communityID string) {
	community, err := m.communitiesManager.GetByIDString(communityID)
	if err != nil {
		m.logger.Error("clean up failed, couldn't delete community", zap.Error(err))
		return
	}
	deleteErr := m.communitiesManager.DeleteCommunity(community.ID())
	if deleteErr != nil {
		m.logger.Error("clean up failed, couldn't delete community", zap.Error(deleteErr))
	}
	deleteErr = m.persistence.DeleteMessagesByCommunityID(community.IDString())
	if deleteErr != nil {
		m.logger.Error("clean up failed, couldn't delete community messages", zap.Error(deleteErr))
	}
}

func (m *Messenger) cleanUpImportChannel(communityID string, channelID string) {
	community, err := m.communitiesManager.GetByIDString(communityID)
	if err != nil {
		m.logger.Error("clean up failed, couldn't delete community", zap.Error(err))
		return
	}

	_, err = community.DeleteChat(channelID)
	if err != nil {
		m.logger.Error("clean up failed, couldn't delete community chat", zap.Error(err))
		return
	}

	err = m.persistence.DeleteMessagesByChatID(channelID)
	if err != nil {
		m.logger.Error("clean up failed, couldn't delete community chat messages", zap.Error(err))
		return
	}
}

func (m *Messenger) publishImportProgress(progress *discord.ImportProgress) {
	m.config.messengerSignalsHandler.DiscordCommunityImportProgress(progress)
}

func (m *Messenger) startPublishImportProgressInterval(c chan *discord.ImportProgress, cancel chan string, done chan struct{}) {

	var currentProgress *discord.ImportProgress

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if currentProgress != nil {
					m.publishImportProgress(currentProgress)
					if currentProgress.Stopped {
						return
					}
				}
			case progressUpdate := <-c:
				currentProgress = progressUpdate
			case <-done:
				if currentProgress != nil {
					m.publishImportProgress(currentProgress)
				}
				return
			case communityID := <-cancel:
				if currentProgress != nil {
					m.publishImportProgress(currentProgress)
				}
				m.cleanUpImport(communityID)
				m.config.messengerSignalsHandler.DiscordCommunityImportCancelled(communityID)
				return
			case <-m.quit:
				m.cleanUpImports()
				return
			}
		}
	}()
}

func (m *Messenger) pinMessagesToWakuMessages(pinMessages []*common.PinMessage, c *communities.Community) ([]*types.Message, error) {
	wakuMessages := make([]*types.Message, 0)
	for _, msg := range pinMessages {

		filter := m.transport.FilterByChatID(msg.LocalChatID)
		encodedPayload, err := proto.Marshal(msg.GetProtobuf())
		if err != nil {
			return nil, err
		}
		wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, protobuf.ApplicationMetadataMessage_PIN_MESSAGE, c.PrivateKey())
		if err != nil {
			return nil, err
		}

		hash := crypto.Keccak256Hash(append([]byte(c.IDString()), wrappedPayload...))
		wakuMessage := &types.Message{
			Sig:          crypto.FromECDSAPub(&c.PrivateKey().PublicKey),
			Timestamp:    uint32(msg.WhisperTimestamp / 1000),
			Topic:        filter.ContentTopic,
			Payload:      wrappedPayload,
			Padding:      []byte{1},
			Hash:         hash[:],
			ThirdPartyID: msg.ID, // CommunityID + DiscordMessageID
		}
		wakuMessages = append(wakuMessages, wakuMessage)
	}

	return wakuMessages, nil
}

func (m *Messenger) torrentClientReady() bool {
	// Simply checking for `torrentConfig.Enabled` isn't enough
	// as there's a possiblity that the torrent client couldn't
	// be instantiated (for example in case of port conflicts)
	return m.config.torrentConfig != nil &&
		m.config.torrentConfig.Enabled &&
		m.communitiesManager.TorrentClientStarted()
}

func (m *Messenger) chatMessagesToWakuMessages(chatMessages []*common.Message, c *communities.Community) ([]*types.Message, error) {
	wakuMessages := make([]*types.Message, 0)
	for _, msg := range chatMessages {

		filter := m.transport.FilterByChatID(msg.LocalChatID)
		encodedPayload, err := proto.Marshal(msg.GetProtobuf())
		if err != nil {
			return nil, err
		}

		wrappedPayload, err := v1protocol.WrapMessageV1(encodedPayload, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE, c.PrivateKey())
		if err != nil {
			return nil, err
		}

		hash := crypto.Keccak256Hash([]byte(msg.ID))
		wakuMessage := &types.Message{
			Sig:          crypto.FromECDSAPub(&c.PrivateKey().PublicKey),
			Timestamp:    uint32(msg.WhisperTimestamp / 1000),
			Topic:        filter.ContentTopic,
			Payload:      wrappedPayload,
			Padding:      []byte{1},
			Hash:         hash[:],
			ThirdPartyID: msg.ID, // CommunityID + DiscordMessageID
		}
		wakuMessages = append(wakuMessages, wakuMessage)
	}

	return wakuMessages, nil
}

func (m *Messenger) GetCommunityToken(communityID string, chainID int, address string) (*token.CommunityToken, error) {
	return m.communitiesManager.GetCommunityToken(communityID, chainID, address)
}

func (m *Messenger) GetCommunityTokens(communityID string) ([]*token.CommunityToken, error) {
	return m.communitiesManager.GetCommunityTokens(communityID)
}

func (m *Messenger) GetAllCommunityTokens() ([]*token.CommunityToken, error) {
	return m.communitiesManager.GetAllCommunityTokens()
}

func (m *Messenger) SaveCommunityToken(token *token.CommunityToken, croppedImage *images.CroppedImage) (*token.CommunityToken, error) {
	return m.communitiesManager.SaveCommunityToken(token, croppedImage)
}

func (m *Messenger) AddCommunityToken(communityID string, chainID int, address string) error {
	communityToken, err := m.communitiesManager.GetCommunityToken(communityID, chainID, address)
	if err != nil {
		return err
	}

	clock, _ := m.getLastClockWithRelatedChat()
	community, err := m.communitiesManager.AddCommunityToken(communityToken, clock)
	if err != nil {
		return err
	}

	err = m.syncCommunity(context.Background(), community, m.dispatchMessage)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) UpdateCommunityTokenState(chainID int, contractAddress string, deployState token.DeployState) error {
	return m.communitiesManager.UpdateCommunityTokenState(chainID, contractAddress, deployState)
}

func (m *Messenger) UpdateCommunityTokenAddress(chainID int, oldContractAddress string, newContractAddress string) error {
	return m.communitiesManager.UpdateCommunityTokenAddress(chainID, oldContractAddress, newContractAddress)
}

func (m *Messenger) UpdateCommunityTokenSupply(chainID int, contractAddress string, supply *bigint.BigInt) error {
	return m.communitiesManager.UpdateCommunityTokenSupply(chainID, contractAddress, supply)
}

func (m *Messenger) RemoveCommunityToken(chainID int, contractAddress string) error {
	return m.communitiesManager.RemoveCommunityToken(chainID, contractAddress)
}

func (m *Messenger) CheckPermissionsToJoinCommunity(request *requests.CheckPermissionToJoinCommunity) (*communities.CheckPermissionToJoinResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	var addresses []gethcommon.Address

	if len(request.Addresses) == 0 {
		accounts, err := m.settings.GetActiveAccounts()
		if err != nil {
			return nil, err
		}

		for _, a := range accounts {
			addresses = append(addresses, gethcommon.HexToAddress(a.Address.Hex()))
		}
	} else {
		for _, v := range request.Addresses {
			addresses = append(addresses, gethcommon.HexToAddress(v))
		}
	}

	return m.communitiesManager.CheckPermissionToJoin(request.CommunityID, addresses)
}

func (m *Messenger) CheckCommunityChannelPermissions(request *requests.CheckCommunityChannelPermissions) (*communities.CheckChannelPermissionsResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var addresses []gethcommon.Address

	if len(request.Addresses) == 0 {
		accounts, err := m.settings.GetActiveAccounts()
		if err != nil {
			return nil, err
		}

		for _, a := range accounts {
			addresses = append(addresses, gethcommon.HexToAddress(a.Address.Hex()))
		}
	} else {
		for _, v := range request.Addresses {
			addresses = append(addresses, gethcommon.HexToAddress(v))
		}
	}

	return m.communitiesManager.CheckChannelPermissions(request.CommunityID, request.ChatID, addresses)
}

func (m *Messenger) CheckAllCommunityChannelsPermissions(request *requests.CheckAllCommunityChannelsPermissions) (*communities.CheckAllChannelsPermissionsResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var addresses []gethcommon.Address

	if len(request.Addresses) == 0 {
		accounts, err := m.settings.GetActiveAccounts()
		if err != nil {
			return nil, err
		}

		for _, a := range accounts {
			addresses = append(addresses, gethcommon.HexToAddress(a.Address.Hex()))
		}
	} else {
		for _, v := range request.Addresses {
			addresses = append(addresses, gethcommon.HexToAddress(v))
		}
	}

	return m.communitiesManager.CheckAllChannelsPermissions(request.CommunityID, addresses)
}

func (m *Messenger) GetCommunityCheckChannelPermissionResponses(communityID types.HexBytes) (*communities.CheckAllChannelsPermissionsResponse, error) {
	return m.communitiesManager.GetCheckChannelPermissionResponses(communityID)
}

func chunkSlice[T comparable](slice []T, chunkSize int) [][]T {
	var chunks [][]T
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		// necessary check to avoid slicing beyond
		// slice capacity
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}

func chunkAttachmentsByByteSize(slice []*protobuf.DiscordMessageAttachment, maxFileSizeBytes uint64) [][]*protobuf.DiscordMessageAttachment {
	var chunks [][]*protobuf.DiscordMessageAttachment

	currentChunkSize := uint64(0)
	currentChunk := make([]*protobuf.DiscordMessageAttachment, 0)

	for i, attachment := range slice {
		payloadBytes := attachment.GetFileSizeBytes()
		if currentChunkSize+payloadBytes > maxFileSizeBytes && len(currentChunk) > 0 {
			chunks = append(chunks, currentChunk)
			currentChunk = make([]*protobuf.DiscordMessageAttachment, 0)
			currentChunkSize = uint64(0)
		}
		currentChunk = append(currentChunk, attachment)
		currentChunkSize = currentChunkSize + payloadBytes
		if i == len(slice)-1 {
			chunks = append(chunks, currentChunk)
		}
	}
	return chunks
}

// startCommunityRekeyLoop creates a 5-minute ticker and starts a routine that attempts to rekey every community every tick
func (m *Messenger) startCommunityRekeyLoop() {
	logger := m.logger.Named("CommunityRekeyLoop")
	var d time.Duration
	if m.communitiesManager.RekeyInterval != 0 {
		if m.communitiesManager.RekeyInterval < 10 {
			d = time.Nanosecond
		} else {
			d = m.communitiesManager.RekeyInterval / 10
		}
	} else {
		d = 5 * time.Minute
	}

	ticker := time.NewTicker(d)
	go func() {
		for {
			select {
			case <-ticker.C:
				m.rekeyCommunities(logger)
			case <-m.quit:
				ticker.Stop()
				logger.Debug("CommunityRekeyLoop stopped")
				return
			}
		}
	}()
}

// rekeyCommunities loops over controlled communities and rekeys if rekey interval elapsed
func (m *Messenger) rekeyCommunities(logger *zap.Logger) {
	// TODO in future have a community level rki rather than a global rki
	var rekeyInterval time.Duration
	if m.communitiesManager.RekeyInterval == 0 {
		rekeyInterval = 48 * time.Hour
	} else {
		rekeyInterval = m.communitiesManager.RekeyInterval
	}

	shouldRekey := func(hashRatchetGroupID []byte) bool {
		key, err := m.sender.GetCurrentKeyForGroup(hashRatchetGroupID)
		if err != nil {
			logger.Error("failed to get current hash ratchet key", zap.Error(err))
			return false
		}

		keyDistributedAt := time.UnixMilli(int64(key.Timestamp))
		return time.Now().After(keyDistributedAt.Add(rekeyInterval))
	}

	controlledCommunities, err := m.ControlledCommunities()
	if err != nil {
		logger.Error("error getting communities", zap.Error(err))
		return
	}

	for _, c := range controlledCommunities {
		keyActions := &communities.EncryptionKeyActions{
			CommunityKeyAction: communities.EncryptionKeyAction{},
			ChannelKeysActions: map[string]communities.EncryptionKeyAction{},
		}

		if c.Encrypted() && shouldRekey(c.ID()) {
			keyActions.CommunityKeyAction = communities.EncryptionKeyAction{
				ActionType: communities.EncryptionKeyRekey,
				Members:    c.Members(),
			}
		}

		for channelID, channel := range c.Chats() {
			if c.ChannelEncrypted(channelID) && shouldRekey([]byte(c.IDString()+channelID)) {
				keyActions.ChannelKeysActions[channelID] = communities.EncryptionKeyAction{
					ActionType: communities.EncryptionKeyRekey,
					Members:    channel.Members,
				}
			}
		}

		err = m.communitiesKeyDistributor.Distribute(c, keyActions)
		if err != nil {
			logger.Error("failed to rekey community", zap.Error(err), zap.String("community ID", c.IDString()))
			continue
		}
	}
}

func (m *Messenger) GetCommunityMembersForWalletAddresses(communityID types.HexBytes, chainID uint64) (map[string]*Contact, error) {
	community, err := m.communitiesManager.GetByID(communityID)
	if err != nil {
		return nil, err
	}

	membersForAddresses := map[string]*Contact{}

	for _, memberPubKey := range community.GetMemberPubkeys() {
		memberPubKeyStr := common.PubkeyToHex(memberPubKey)
		revealedAccounts, err := m.communitiesManager.GetRevealedAddresses(communityID, memberPubKeyStr)
		if err != nil {
			return nil, err
		}
		for _, revealedAccount := range revealedAccounts {
			if !slices.Contains(revealedAccount.ChainIds, chainID) {
				continue
			}

			contact, ok := m.allContacts.Load(memberPubKeyStr)
			if ok {
				membersForAddresses[revealedAccount.Address] = contact
			} else {
				m.logger.Error("community member is not a contact", zap.String("contact ID", memberPubKeyStr))
			}
		}
	}

	return membersForAddresses, nil
}

func (m *Messenger) processCommunityChanges(messageState *ReceivedMessageState) {
	// Process any community changes
	for _, changes := range messageState.Response.CommunityChanges {
		if changes.ShouldMemberJoin {
			response, err := m.joinCommunity(context.TODO(), changes.Community.ID(), false)
			if err != nil {
				m.logger.Error("cannot join community", zap.Error(err))
				continue
			}

			if err := messageState.Response.Merge(response); err != nil {
				m.logger.Error("cannot merge join community response", zap.Error(err))
				continue
			}

		} else if changes.MemberKicked {
			response, err := m.kickedOutOfCommunity(changes.Community.ID())
			if err != nil {
				m.logger.Error("cannot leave community", zap.Error(err))
				continue
			}

			if err := messageState.Response.Merge(response); err != nil {
				m.logger.Error("cannot merge join community response", zap.Error(err))
				continue
			}

			// Activity Center notification
			now := m.GetCurrentTimeInMillis()
			notification := &ActivityCenterNotification{
				ID:          types.FromHex(uuid.New().String()),
				Type:        ActivityCenterNotificationTypeCommunityKicked,
				Timestamp:   now,
				CommunityID: changes.Community.IDString(),
				Read:        false,
				UpdatedAt:   now,
			}

			err = m.addActivityCenterNotification(response, notification, nil)
			if err != nil {
				m.logger.Error("failed to save notification", zap.Error(err))
				continue
			}

			if err := messageState.Response.Merge(response); err != nil {
				m.logger.Error("cannot merge notification response", zap.Error(err))
				continue
			}
		}
	}
	// Clean up as not used by clients currently
	messageState.Response.CommunityChanges = nil
}

func (m *Messenger) SetCommunitySignerPubKey(ctx context.Context, communityID []byte, chainID uint64, contractAddress string, txArgs transactions.SendTxArgs, password string, newSignerPubKey string) (string, error) {
	if m.communityTokensService == nil {
		return "", errors.New("tokens service not initialized")
	}

	return m.communityTokensService.SetSignerPubKey(ctx, chainID, contractAddress, txArgs, password, newSignerPubKey)
}

func (m *Messenger) PromoteSelfToControlNode(communityID types.HexBytes) (*MessengerResponse, error) {
	clock, _ := m.getLastClockWithRelatedChat()
	changes, err := m.communitiesManager.PromoteSelfToControlNode(communityID, clock)
	if err != nil {
		return nil, err
	}

	if len(changes.MembersRemoved) > 0 {
		err = m.communitiesManager.GenerateRequestsToJoinForAutoApprovalOnNewOwnership(changes.Community.ID(), changes.MembersRemoved)
		if err != nil {
			return nil, err
		}
	}

	err = m.syncCommunity(context.Background(), changes.Community, m.dispatchMessage)
	if err != nil {
		return nil, err
	}

	var response MessengerResponse
	response.AddCommunity(changes.Community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	return &response, nil
}
