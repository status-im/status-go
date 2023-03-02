package protocol

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	_errors "errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/meirf/gopart"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/discord"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
	"github.com/status-im/status-go/signal"
)

// 7 days interval
var messageArchiveInterval = 7 * 24 * time.Hour

const discordTimestampLayout = "2006-01-02T15:04:05+00:00"

func (m *Messenger) publishOrg(org *communities.Community) error {
	m.logger.Debug("publishing org", zap.String("org-id", org.IDString()), zap.Any("org", org))
	payload, err := org.MarshaledDescription()
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload: payload,
		Sender:  org.PrivateKey(),
		// we don't want to wrap in an encryption layer message
		SkipEncryption: true,
		MessageType:    protobuf.ApplicationMetadataMessage_COMMUNITY_DESCRIPTION,
	}
	_, err = m.sender.SendPublic(context.Background(), org.IDString(), rawMessage)
	return err
}

func (m *Messenger) publishOrgInvitation(org *communities.Community, invitation *protobuf.CommunityInvitation) error {
	m.logger.Debug("publishing org invitation", zap.String("org-id", org.IDString()), zap.Any("org", org))
	pk, err := crypto.DecompressPubkey(invitation.PublicKey)
	if err != nil {
		return err
	}

	payload, err := proto.Marshal(invitation)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		Payload: payload,
		Sender:  org.PrivateKey(),
		// we don't want to wrap in an encryption layer message
		SkipEncryption: true,
		MessageType:    protobuf.ApplicationMetadataMessage_COMMUNITY_INVITATION,
	}
	_, err = m.sender.SendPrivate(context.Background(), pk, &rawMessage)
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

					if c.IsAdmin() {
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

	go func() {
		for {
			select {
			case sub, more := <-c:
				if !more {
					return
				}
				if sub.Community != nil {
					err := m.publishOrg(sub.Community)
					if err != nil {
						m.logger.Warn("failed to publish org", zap.Error(err))
					}
				}

				for _, invitation := range sub.Invitations {
					err := m.publishOrgInvitation(sub.Community, invitation)
					if err != nil {
						m.logger.Warn("failed to publish org invitation", zap.Error(err))
					}
				}

				m.logger.Debug("published org")
			case <-ticker.C:
				// If we are not online, we don't even try
				if !m.online() {
					continue
				}

				// If not enough time has passed since last advertisement, we skip this
				if time.Now().Unix()-lastPublished < communityAdvertiseIntervalSecond {
					continue
				}

				orgs, err := m.communitiesManager.Created()
				if err != nil {
					m.logger.Warn("failed to retrieve orgs", zap.Error(err))
				}

				for idx := range orgs {
					org := orgs[idx]
					_, beingImported := m.importingCommunities[org.IDString()]
					if !beingImported {
						err := m.publishOrg(org)
						if err != nil {
							m.logger.Warn("failed to publish org", zap.Error(err))
						}
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

func (m *Messenger) Communities() ([]*communities.Community, error) {
	return m.communitiesManager.All()
}

func (m *Messenger) JoinedCommunities() ([]*communities.Community, error) {
	return m.communitiesManager.Joined()
}

func (m *Messenger) SpectatedCommunities() ([]*communities.Community, error) {
	return m.communitiesManager.Spectated()
}

func (m *Messenger) CuratedCommunities() (*communities.KnownCommunitiesResponse, error) {
	// Revert code to https://github.com/status-im/status-go/blob/e6a3f63ec7f2fa691878ed35f921413dc8acfc66/protocol/messenger_communities.go#L211-L226 once the curated communities contract is deployed to mainnet

	chainID := uint64(420) // Optimism Goerli
	sDB, err := accounts.NewDB(m.database)
	if err != nil {
		return nil, err
	}
	nodeConfig, err := sDB.GetNodeConfig()
	if err != nil {
		return nil, err
	}
	var backend *ethclient.Client
	for _, n := range nodeConfig.Networks {
		if n.ChainID == chainID {
			b, err := ethclient.Dial(n.RPCURL)
			if err != nil {
				return nil, err
			}
			backend = b
		}
	}
	directory, err := m.contractMaker.NewDirectoryWithBackend(chainID, backend)
	if err != nil {
		return nil, err
	}
	// --- end delete

	callOpts := &bind.CallOpts{Context: context.Background(), Pending: false}

	communities, err := directory.GetCommunities(callOpts)
	if err != nil {
		return nil, err
	}
	var communityIDs []types.HexBytes
	for _, c := range communities {
		communityIDs = append(communityIDs, c)
	}

	response, err := m.communitiesManager.GetStoredDescriptionForCommunities(communityIDs)
	if err != nil {
		return nil, err
	}

	go m.requestCommunitiesFromMailserver(response.UnknownCommunities)

	return response, nil
}

func (m *Messenger) initCommunityChats(community *communities.Community) ([]*Chat, error) {
	logger := m.logger.Named("initCommunityChats")

	chatIDs := community.DefaultFilters()

	chats := CreateCommunityChats(community, m.getTimesource())

	for _, chat := range chats {
		chatIDs = append(chatIDs, chat.ID)
	}

	// Load transport filters
	filters, err := m.transport.InitPublicFilters(chatIDs)
	if err != nil {
		logger.Debug("m.transport.InitPublicFilters error", zap.Error(err))
		return nil, err
	}

	if community.IsAdmin() {
		// Init the community filter so we can receive messages on the community
		communityFilters, err := m.transport.InitCommunityFilters([]*ecdsa.PrivateKey{community.PrivateKey()})
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

func (m *Messenger) JoinCommunity(ctx context.Context, communityID types.HexBytes) (*MessengerResponse, error) {
	mr, err := m.joinCommunity(ctx, communityID)
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

func (m *Messenger) joinCommunity(ctx context.Context, communityID types.HexBytes) (*MessengerResponse, error) {
	logger := m.logger.Named("joinCommunity")

	response := &MessengerResponse{}

	community, err := m.communitiesManager.JoinCommunity(communityID)
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

	return response, nil
}

func (m *Messenger) SetMuted(communityID types.HexBytes, muted bool) error {
	return m.communitiesManager.SetMuted(communityID, muted)
}

func (m *Messenger) SetMutePropertyOnChatsByCategory(communityID string, categoryID string, muted bool) error {
	community, err := m.communitiesManager.GetByIDString(communityID)
	if err != nil {
		return err
	}

	for _, chatID := range community.ChatsByCategoryID(categoryID) {
		if muted {
			err = m.MuteChat(communityID + chatID)
		} else {
			err = m.UnmuteChat(communityID + chatID)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Messenger) RequestToJoinCommunity(request *requests.RequestToJoinCommunity) (*MessengerResponse, error) {
	logger := m.logger.Named("RequestToJoinCommunity")
	if err := request.Validate(); err != nil {
		logger.Debug("request failed to validate", zap.Error(err), zap.Any("request", request))
		return nil, err
	}

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return nil, err
	}

	community, requestToJoin, err := m.communitiesManager.RequestToJoin(&m.identity.PublicKey, request)
	if err != nil {
		return nil, err
	}
	err = m.syncCommunity(context.Background(), community, m.dispatchMessage)
	if err != nil {
		return nil, err
	}

	requestToJoinProto := &protobuf.CommunityRequestToJoin{
		Clock:       requestToJoin.Clock,
		EnsName:     requestToJoin.ENSName,
		DisplayName: displayName,
		CommunityId: community.ID(),
	}

	payload, err := proto.Marshal(requestToJoinProto)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		Payload:        payload,
		CommunityID:    community.ID(),
		SkipEncryption: true,
		MessageType:    protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN,
	}
	_, err = m.sender.SendCommunityMessage(context.Background(), rawMessage)
	if err != nil {
		return nil, err
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
	}

	err = m.addActivityCenterNotification(response, notification)
	if err != nil {
		m.logger.Error("failed to save notification", zap.Error(err))
		return nil, err
	}

	return response, nil
}

func (m *Messenger) CreateCommunityCategory(request *requests.CreateCommunityCategory) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var response MessengerResponse
	community, changes, err := m.communitiesManager.CreateCategory(request, true)
	if err != nil {
		return nil, err
	}
	response.AddCommunity(community)
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

func (m *Messenger) CancelRequestToJoinCommunity(request *requests.CancelRequestToJoinCommunity) (*MessengerResponse, error) {
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
		Payload:        payload,
		CommunityID:    community.ID(),
		SkipEncryption: true,
		MessageType:    protobuf.ApplicationMetadataMessage_COMMUNITY_CANCEL_REQUEST_TO_JOIN,
	}
	_, err = m.sender.SendCommunityMessage(context.Background(), rawMessage)

	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) AcceptRequestToJoinCommunity(request *requests.AcceptRequestToJoinCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, err := m.communitiesManager.AcceptRequestToJoin(request)
	if err != nil {
		return nil, err
	}

	requestToJoin, err := m.communitiesManager.GetRequestToJoin(request.ID)
	if err != nil {
		return nil, err
	}

	pk, err := common.HexToPubkey(requestToJoin.PublicKey)
	if err != nil {
		return nil, err
	}

	grant, err := community.BuildGrant(pk, "")
	if err != nil {
		return nil, err
	}

	requestToJoinResponseProto := &protobuf.CommunityRequestToJoinResponse{
		Clock:       community.Clock(),
		Accepted:    true,
		CommunityId: community.ID(),
		Community:   community.Description(),
		Grant:       grant,
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

	err = m.SendKeyExchangeMessage(community.ID(), []*ecdsa.PublicKey{pk}, common.KeyExMsgReuse)
	if err != nil {
		return nil, err
	}

	rawMessage := &common.RawMessage{
		Payload:        payload,
		Sender:         community.PrivateKey(),
		SkipEncryption: true,
		MessageType:    protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN_RESPONSE,
	}

	_, err = m.sender.SendPrivate(context.Background(), pk, rawMessage)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)

	// Activity Center notification
	notification, err := m.persistence.GetActivityCenterNotificationByID(request.ID)
	if err != nil {
		return nil, err
	}

	if notification != nil {
		notification.MembershipStatus = ActivityCenterMembershipStatusAccepted
		notification.Read = true
		notification.Accepted = true

		err = m.addActivityCenterNotification(response, notification)
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

	err := m.communitiesManager.DeclineRequestToJoin(request)
	if err != nil {
		return nil, err
	}

	// Activity Center notification
	notification, err := m.persistence.GetActivityCenterNotificationByID(request.ID)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}

	if notification != nil {
		notification.MembershipStatus = ActivityCenterMembershipStatusDeclined
		notification.Read = true
		notification.Dismissed = true

		err = m.addActivityCenterNotification(response, notification)
		if err != nil {
			m.logger.Error("failed to save notification", zap.Error(err))
			return nil, err
		}
	}

	return response, nil
}

func (m *Messenger) LeaveCommunity(communityID types.HexBytes) (*MessengerResponse, error) {
	err := m.persistence.DismissAllActivityCenterNotificationsFromCommunity(communityID.String())
	if err != nil {
		return nil, err
	}

	mr, err := m.leaveCommunity(communityID)
	if err != nil {
		return nil, err
	}

	err = m.communitiesManager.DeleteCommunitySettings(communityID)
	if err != nil {
		return nil, err
	}

	m.communitiesManager.StopHistoryArchiveTasksInterval(communityID)

	if com, ok := mr.communities[communityID.String()]; ok {
		err = m.syncCommunity(context.Background(), com, m.dispatchMessage)
		if err != nil {
			return nil, err
		}
	}

	isAdmin, err := m.communitiesManager.IsAdminCommunityByID(communityID)
	if err != nil {
		return nil, err
	}

	if !isAdmin {
		requestToLeaveProto := &protobuf.CommunityRequestToLeave{
			Clock:       uint64(time.Now().Unix()),
			CommunityId: communityID,
		}

		payload, err := proto.Marshal(requestToLeaveProto)
		if err != nil {
			return nil, err
		}

		rawMessage := common.RawMessage{
			Payload:        payload,
			CommunityID:    communityID,
			SkipEncryption: true,
			MessageType:    protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_LEAVE,
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

func (m *Messenger) CreateCommunityChat(communityID types.HexBytes, c *protobuf.CommunityChat) (*MessengerResponse, error) {
	var response MessengerResponse

	c.Identity.FirstMessageTimestamp = FirstMessageTimestampNoMessage
	community, changes, err := m.communitiesManager.CreateChat(communityID, c, true, "")
	if err != nil {
		return nil, err
	}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}

	var chats []*Chat
	var chatIDs []string
	for chatID, chat := range changes.ChatsAdded {
		c := CreateCommunityChat(community.IDString(), chatID, chat, m.getTimesource())
		chats = append(chats, c)
		chatIDs = append(chatIDs, c.ID)
		response.AddChat(c)
	}

	// Load filters
	filters, err := m.transport.InitPublicFilters(chatIDs)
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
	var chatIDs []string
	for chatID, change := range changes.ChatsModified {
		c := CreateCommunityChat(community.IDString(), chatID, change.ChatModified, m.getTimesource())
		chats = append(chats, c)
		chatIDs = append(chatIDs, c.ID)
		response.AddChat(c)
	}

	// Load filters
	filters, err := m.transport.InitPublicFilters(chatIDs)
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

	// Init the community filter so we can receive messages on the community
	_, err = m.transport.InitCommunityFilters([]*ecdsa.PrivateKey{community.PrivateKey()})
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
				Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
			},
		})
		if err != nil {
			return nil, err
		}

		// updating community so it contains the general chat
		community = chatResponse.Communities()[0]
		response.AddChat(chatResponse.Chats()[0])
	}

	if request.Encrypted {
		// Init hash ratchet for community
		_, err = m.encryptor.GenerateHashRatchetKey(community.ID())

		if err != nil {
			return nil, err
		}
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

func (m *Messenger) CreateCommunityTokenPermission(request *requests.CreateCommunityTokenPermission) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, changes, err := m.communitiesManager.CreateCommunityTokenPermission(request)
	if err != nil {
		return nil, err
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

	response := &MessengerResponse{}
	response.AddCommunity(community)
	response.CommunityChanges = []*communities.CommunityChanges{changes}
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

func (m *Messenger) ExportCommunity(id types.HexBytes) (*ecdsa.PrivateKey, error) {
	return m.communitiesManager.ExportCommunity(id)
}

func (m *Messenger) ImportCommunity(ctx context.Context, key *ecdsa.PrivateKey) (*MessengerResponse, error) {
	community, err := m.communitiesManager.ImportCommunity(key)
	if err != nil {
		return nil, err
	}

	// Load filters
	_, err = m.transport.InitPublicFilters(community.DefaultFilters())
	if err != nil {
		return nil, err
	}

	// TODO Init hash ratchet for community
	_, err = m.encryptor.GenerateHashRatchetKey(community.ID())

	if err != nil {
		return nil, err
	}

	_, err = m.RequestCommunityInfoFromMailserver(community.IDString(), false)
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

	response, err := m.JoinCommunity(ctx, community.ID())
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

func (m *Messenger) InviteUsersToCommunity(request *requests.InviteUsersToCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	var messages []*common.Message

	var publicKeys []*ecdsa.PublicKey
	community, err := m.communitiesManager.GetByID(request.CommunityID)

	if err != nil {
		return nil, err
	}

	for _, pkBytes := range request.Users {
		publicKey, err := common.HexToPubkey(pkBytes.String())
		if err != nil {
			return nil, err
		}
		publicKeys = append(publicKeys, publicKey)

		message := &common.Message{}
		message.ChatId = pkBytes.String()
		message.CommunityID = request.CommunityID.String()
		message.Text = fmt.Sprintf("You have been invited to community %s", community.Name())
		messages = append(messages, message)
		r, err := m.CreateOneToOneChat(&requests.CreateOneToOneChat{ID: pkBytes})
		if err != nil {
			return nil, err
		}

		if err := response.Merge(r); err != nil {
			return nil, err
		}
	}

	err = m.SendKeyExchangeMessage(community.ID(), publicKeys, common.KeyExMsgReuse)
	if err != nil {
		return nil, err
	}

	community, err = m.communitiesManager.InviteUsersToCommunity(request.CommunityID, publicKeys)
	if err != nil {
		return nil, err
	}
	sendMessagesResponse, err := m.SendChatMessages(context.Background(), messages)
	if err != nil {
		return nil, err
	}

	if err := response.Merge(sendMessagesResponse); err != nil {
		return nil, err
	}

	response.AddCommunity(community)
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
		message := &common.Message{}
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

func (m *Messenger) PendingRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.PendingRequestsToJoinForCommunity(id)
}

func (m *Messenger) DeclinedRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.DeclinedRequestsToJoinForCommunity(id)
}

func (m *Messenger) CanceledRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.CanceledRequestsToJoinForCommunity(id)
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

// TODO
func (m *Messenger) SendKeyExchangeMessage(communityID []byte, pubkeys []*ecdsa.PublicKey, msgType common.CommKeyExMsgType) error {
	rawMessage := common.RawMessage{
		SkipEncryption:        false,
		CommunityID:           communityID,
		CommunityKeyExMsgType: msgType,
		Recipients:            pubkeys,
		MessageType:           protobuf.ApplicationMetadataMessage_CHAT_MESSAGE,
	}
	_, err := m.sender.SendCommunityMessage(context.Background(), rawMessage)

	if err != nil {
		return err
	}
	return nil
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

func (m *Messenger) BanUserFromCommunity(request *requests.BanUserFromCommunity) (*MessengerResponse, error) {
	community, err := m.communitiesManager.BanUserFromCommunity(request)
	if err != nil {
		return nil, err
	}

	// TODO generate new encryption key
	err = m.SendKeyExchangeMessage(community.ID(), community.GetMemberPubkeys(), common.KeyExMsgRekey)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response, err = m.DeclineAllPendingGroupInvitesFromUser(response, request.User.String())
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

// RequestCommunityInfoFromMailserver installs filter for community and requests its details
// from mailserver. It waits until it has the community before returning it.
// If useDatabase is true, it searches for community in database and does not request mailserver.
func (m *Messenger) RequestCommunityInfoFromMailserver(communityID string, useDatabase bool) (*communities.Community, error) {
	if useDatabase {
		community, err := m.findCommunityInfoFromDB(communityID)
		if err != nil {
			return nil, err
		}
		if community != nil {
			return community, nil
		}
	}

	return m.requestCommunityInfoFromMailserver(communityID, true)
}

// RequestCommunityInfoFromMailserverAsync installs filter for community and requests its details
// from mailserver. When response received it will be passed through signals handler
func (m *Messenger) RequestCommunityInfoFromMailserverAsync(communityID string) error {
	community, err := m.findCommunityInfoFromDB(communityID)
	if err != nil {
		return err
	}
	if community != nil {
		m.config.messengerSignalsHandler.CommunityInfoFound(community)
		return nil
	}
	_, err = m.requestCommunityInfoFromMailserver(communityID, false)
	return err
}

// RequestCommunityInfoFromMailserver installs filter for community and requests its details
// from mailserver. When response received it will be passed through signals handler
func (m *Messenger) requestCommunityInfoFromMailserver(communityID string, waitForResponse bool) (*communities.Community, error) {
	m.requestedCommunitiesLock.Lock()
	defer m.requestedCommunitiesLock.Unlock()

	if _, ok := m.requestedCommunities[communityID]; ok {
		return nil, nil
	}

	//If filter wasn't installed we create it and remember for deinstalling after
	//response received
	filter := m.transport.FilterByChatID(communityID)
	if filter == nil {
		filters, err := m.transport.InitPublicFilters([]string{communityID})
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

	to := uint32(m.transport.GetCurrentTime() / 1000)
	from := to - oneMonthInSeconds

	_, err := m.performMailserverRequest(func() (*MessengerResponse, error) {

		batch := MailserverBatch{From: from, To: to, Topics: []types.TopicType{filter.Topic}}
		m.logger.Info("Requesting historic")
		err := m.processMailserverBatch(batch)
		return nil, err
	})
	if err != nil {
		return nil, err
	}

	if !waitForResponse {
		return nil, nil
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var community *communities.Community

	fetching := true

	for fetching {
		select {
		case <-time.After(200 * time.Millisecond):
			//send signal to client that message status updated
			community, err = m.communitiesManager.GetByIDString(communityID)
			if err != nil {
				return nil, err
			}

			if community != nil && community.Name() != "" && community.DescriptionText() != "" {
				fetching = false
			}

		case <-ctx.Done():
			fetching = false
		}
	}

	if community == nil {
		return nil, nil
	}

	//if there is no info helpful for client, we don't post it
	if community.Name() == "" && community.DescriptionText() == "" {
		return nil, nil
	}

	m.forgetCommunityRequest(communityID)

	return community, nil
}

// RequestCommunityInfoFromMailserver installs filter for community and requests its details
// from mailserver. When response received it will be passed through signals handler
func (m *Messenger) requestCommunitiesFromMailserver(communityIDs []string) {
	m.requestedCommunitiesLock.Lock()
	defer m.requestedCommunitiesLock.Unlock()

	var topics []types.TopicType
	for _, communityID := range communityIDs {
		if _, ok := m.requestedCommunities[communityID]; ok {
			continue
		}

		//If filter wasn't installed we create it and remember for deinstalling after
		//response received
		filter := m.transport.FilterByChatID(communityID)
		if filter == nil {
			filters, err := m.transport.InitPublicFilters([]string{communityID})
			if err != nil {
				m.logger.Error("Can't install filter for community", zap.Error(err))
				continue
			}
			if len(filters) != 1 {
				m.logger.Error("Unexpected amount of filters created")
				continue
			}
			filter = filters[0]
			m.requestedCommunities[communityID] = filter
		} else {
			//we don't remember filter id associated with community because it was already installed
			m.requestedCommunities[communityID] = nil
		}
		topics = append(topics, filter.Topic)
	}

	to := uint32(m.transport.GetCurrentTime() / 1000)
	from := to - oneMonthInSeconds

	_, err := m.performMailserverRequest(func() (*MessengerResponse, error) {
		batch := MailserverBatch{From: from, To: to, Topics: topics}
		m.logger.Info("Requesting historic")
		err := m.processMailserverBatch(batch)
		return nil, err
	})

	if err != nil {
		m.logger.Error("Err performing mailserver request", zap.Error(err))
		return
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	fetching := true
	for fetching {
		select {
		case <-time.After(200 * time.Millisecond):
			allLoaded := true
			for _, c := range communityIDs {
				community, err := m.communitiesManager.GetByIDString(c)
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

	for _, c := range communityIDs {
		m.forgetCommunityRequest(c)
	}

}

// forgetCommunityRequest removes community from requested ones and removes filter
func (m *Messenger) forgetCommunityRequest(communityID string) {
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
func (m *Messenger) handleCommunityDescription(state *ReceivedMessageState, signer *ecdsa.PublicKey, description protobuf.CommunityDescription, rawPayload []byte) error {
	communityResponse, err := m.communitiesManager.HandleCommunityDescriptionMessage(signer, &description, rawPayload)
	if err != nil {
		return err
	}

	community := communityResponse.Community

	state.Response.AddCommunity(community)
	state.Response.CommunityChanges = append(state.Response.CommunityChanges, communityResponse.Changes)

	// If we haven't joined the org, nothing to do
	if !community.Joined() {
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
	var chatIDs []string
	for i, chat := range chats {

		oldChat, ok := state.AllChats.Load(chat.ID)
		if !ok {
			// Beware, don't use the reference in the range (i.e chat) as it's a shallow copy
			state.AllChats.Store(chat.ID, chats[i])

			state.Response.AddChat(chat)
			chatIDs = append(chatIDs, chat.ID)
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
	filters, err := m.transport.InitPublicFilters(chatIDs)
	if err != nil {
		return err
	}
	_, err = m.scheduleSyncFilters(filters)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) handleSyncCommunity(messageState *ReceivedMessageState, syncCommunity protobuf.SyncCommunity) error {
	logger := m.logger.Named("handleSyncCommunity")

	// Should handle community
	shouldHandle, err := m.communitiesManager.ShouldHandleSyncCommunity(&syncCommunity)
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
		_, err := m.encryptor.HandleHashRatchetKeys(syncCommunity.Id, syncCommunity.EncryptionKeys)
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

	err = m.handleCommunityDescription(messageState, orgPubKey, cd, syncCommunity.Description)
	if err != nil {
		logger.Debug("m.handleCommunityDescription error", zap.Error(err))
		return err
	}

	if syncCommunity.Settings != nil {
		err = m.handleSyncCommunitySettings(messageState, *syncCommunity.Settings)
		if err != nil {
			logger.Debug("m.handleSyncCommunitySettings error", zap.Error(err))
			return err
		}
	}

	// associate private key with community if set
	if syncCommunity.PrivateKey != nil {
		orgPrivKey, err := crypto.ToECDSA(syncCommunity.PrivateKey)
		if err != nil {
			logger.Debug("crypto.ToECDSA", zap.Error(err))
			return err
		}
		err = m.communitiesManager.SetPrivateKey(syncCommunity.Id, orgPrivKey)
		if err != nil {
			logger.Debug("m.communitiesManager.SetPrivateKey", zap.Error(err))
			return err
		}
	}

	// if we are not waiting for approval, join or leave the community
	if !pending {
		var mr *MessengerResponse
		if syncCommunity.Joined {
			mr, err = m.joinCommunity(context.Background(), syncCommunity.Id)
			if err != nil {
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
		err = messageState.Response.Merge(mr)
		if err != nil {
			logger.Debug("messageState.Response.Merge error", zap.Error(err))
			return err
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

func (m *Messenger) handleSyncCommunitySettings(messageState *ReceivedMessageState, syncCommunitySettings protobuf.SyncCommunitySettings) error {
	shouldHandle, err := m.communitiesManager.ShouldHandleSyncCommunitySettings(&syncCommunitySettings)
	if err != nil {
		m.logger.Debug("m.communitiesManager.ShouldHandleSyncCommunitySettings error", zap.Error(err))
		return err
	}
	m.logger.Debug("ShouldHandleSyncCommunity result", zap.Bool("shouldHandle", shouldHandle))
	if !shouldHandle {
		return nil
	}

	communitySettings, err := m.communitiesManager.HandleSyncCommunitySettings(&syncCommunitySettings)
	if err != nil {
		return err
	}

	messageState.Response.AddCommunitySettings(communitySettings)
	return nil
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
				topics = append(topics, filter.Topic)
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

func (m *Messenger) resumeHistoryArchivesImport(communityID types.HexBytes) error {
	archiveIDsToImport, err := m.communitiesManager.GetMessageArchiveIDsToImport(communityID)
	if err != nil {
		return err
	}

	if len(archiveIDsToImport) == 0 {
		return nil
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

func (m *Messenger) importHistoryArchives(communityID types.HexBytes, cancel chan struct{}) error {
	importTicker := time.NewTicker(100 * time.Millisecond)
	defer importTicker.Stop()

importMessageArchivesLoop:
	for {
		select {
		case <-cancel:
			m.communitiesManager.LogStdout("interrupted importing history archive messages")
			return nil
		case <-importTicker.C:

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
			response, err := m.handleArchiveMessages(archiveMessages, communityID)
			if err != nil {
				m.communitiesManager.LogStdout("failed to handle archive messages", zap.Error(err))
				continue
			}

			err = m.communitiesManager.SetMessageArchiveIDImported(communityID, downloadedArchiveID, true)
			if err != nil {
				m.communitiesManager.LogStdout("failed to mark history message archive as imported", zap.Error(err))
				continue
			}

			if !response.IsEmpty() {
				notifications := response.Notifications()
				response.ClearNotifications()
				signal.SendNewMessages(response)
				localnotifications.PushMessages(notifications)
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
		MessageType:          protobuf.ApplicationMetadataMessage_COMMUNITY_ARCHIVE_MAGNETLINK,
		SkipGroupMessageWrap: true,
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

	communities, err := m.communitiesManager.Created()
	if err != nil {
		return err
	}

	if len(communities) > 0 {
		go m.InitHistoryArchiveTasks(communities)
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

		if createCommunityRequest.Encrypted {
			// Init hash ratchet for community
			_, err = m.encryptor.GenerateHashRatchetKey(discordCommunity.ID())

			if err != nil {
				m.cleanUpImport(discordCommunity.IDString())
				importProgress.AddTaskError(discord.CommunityCreationTask, discord.Error(err.Error()))
				importProgress.StopTask(discord.CommunityCreationTask)
				progressUpdates <- importProgress
				return
			}
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
						Access: protobuf.CommunityPermissions_NO_MEMBERSHIP,
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
				communityWithChats, changes, err := m.communitiesManager.CreateChat(discordCommunity.ID(), communityChat, false, channel.Channel.ID)
				if err != nil {
					m.cleanUpImport(communityID)
					errmsg := err.Error()
					if _errors.Is(err, communities.ErrInvalidCommunityDescriptionDuplicatedName) {
						errmsg = fmt.Sprintf("Couldn't create channel '%s': %s", communityChat.Identity.DisplayName, err.Error())
					}
					importProgress.AddTaskError(discord.ChannelsCreationTask, discord.Error(errmsg))
					importProgress.StopTask(discord.ChannelsCreationTask)
					progressUpdates <- importProgress
					return
				}
				discordCommunity = communityWithChats

				// This looks like we keep overriding the chat id value
				// as we iterate over `ChatsAdded`, however at this point we
				// know there was only a single such change (and it's a map)
				for chatID, chat := range changes.ChatsAdded {
					c := CreateCommunityChat(communityID, chatID, chat, m.getTimesource())
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
					_, exists := messagesToSave[communityID+discordMessage.Reference.MessageId]
					if exists {
						chatMessage.ResponseTo = communityID + discordMessage.Reference.MessageId
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
					ChatMessage:      chatMessage,
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

					_, exists := messagesToSave[communityID+discordMessage.Reference.MessageId]
					if exists {
						pinMessage := protobuf.PinMessage{
							Clock:       messageToSave.WhisperTimestamp,
							MessageId:   communityID + discordMessage.Reference.MessageId,
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

						messageID := v1protocol.MessageID(&communityPubKey, wrappedPayload)

						pinMessageToSave := common.PinMessage{
							ID:               types.EncodeHex(messageID),
							PinMessage:       pinMessage,
							LocalChatID:      processedChannelIds[channel.Channel.ID],
							From:             messageToSave.From,
							SigPubKey:        messageToSave.SigPubKey,
							WhisperTimestamp: messageToSave.WhisperTimestamp,
						}

						pinMessagesToSave = append(pinMessagesToSave, &pinMessageToSave)
					}
				} else {
					messagesToSave[communityID+discordMessage.Id] = messageToSave
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
				discordMessages = append(discordMessages, msg.GetDiscordMessage())
			}

			// We save these messages in chunks so we don't block the database
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

			var messages []*common.Message
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
					progressValue := calculateProgress(i+1, totalImportChunkCount, (float32(assetCounter.Value())/float32(totalAssetsCount))*0.25)
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
						progressValue := calculateProgress(i+1, totalImportChunkCount, 0.25+(float32(assetCounter.Value())/float32(totalAssetsCount))*0.25)
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
		_, err = m.transport.InitCommunityFilters([]*ecdsa.PrivateKey{discordCommunity.PrivateKey()})
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

func (m *Messenger) DiscordImportMarkedAsCancelled(communityID string) bool {
	cancelled, exists := m.importingCommunities[communityID]
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
			Sig:       crypto.FromECDSAPub(&c.PrivateKey().PublicKey),
			Timestamp: uint32(msg.WhisperTimestamp / 1000),
			Topic:     filter.Topic,
			Payload:   wrappedPayload,
			Padding:   []byte{1},
			Hash:      hash[:],
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

		hash := crypto.Keccak256Hash([]byte(c.IDString() + msg.GetDiscordMessage().Id))
		wakuMessage := &types.Message{
			Sig:          crypto.FromECDSAPub(&c.PrivateKey().PublicKey),
			Timestamp:    uint32(msg.WhisperTimestamp / 1000),
			Topic:        filter.Topic,
			Payload:      wrappedPayload,
			Padding:      []byte{1},
			Hash:         hash[:],
			ThirdPartyID: c.IDString() + msg.GetDiscordMessage().Id,
		}
		wakuMessages = append(wakuMessages, wakuMessage)
	}

	return wakuMessages, nil
}

func (m *Messenger) GetCommunityTokens(communityID string) ([]*communities.CommunityToken, error) {
	return m.communitiesManager.GetCommunityTokens(communityID)
}

func (m *Messenger) AddCommunityToken(token *communities.CommunityToken) error {
	return m.communitiesManager.AddCommunityToken(token)
}

func (m *Messenger) UpdateCommunityTokenState(contractAddress string, deployState communities.DeployState) error {
	return m.communitiesManager.UpdateCommunityTokenState(contractAddress, deployState)
}
