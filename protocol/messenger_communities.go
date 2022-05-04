package protocol

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/transport"
)

// 7 days interval
var messageArchiveInterval = 7 * 24 * time.Hour

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
					err := m.publishOrg(org)
					if err != nil {
						m.logger.Warn("failed to publish org", zap.Error(err))
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

func (m *Messenger) JoinCommunity(ctx context.Context, communityID types.HexBytes) (*MessengerResponse, error) {
	mr, err := m.joinCommunity(ctx, communityID)
	if err != nil {
		return nil, err
	}

	communitySettings := communities.CommunitySettings{
		CommunityID:                  communityID.String(),
		HistoryArchiveSupportEnabled: true,
	}

	err = m.communitiesManager.SaveCommunitySettings(communitySettings)
	if err != nil {
		return nil, err
	}

	mr.AddCommunitySettings(&communitySettings)

	if com, ok := mr.communities[communityID.String()]; ok {
		err = m.syncCommunity(context.Background(), com)
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

	chatIDs := community.DefaultFilters()

	chats := CreateCommunityChats(community, m.getTimesource())
	response.AddChats(chats)

	for _, chat := range response.Chats() {
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

	response.AddCommunity(community)

	if err = m.saveChats(chats); err != nil {
		logger.Debug("m.saveChats error", zap.Error(err))
		return nil, err
	}

	err = m.reregisterForPushNotifications()
	if err != nil {
		return nil, err
	}

	err = m.sendCurrentUserStatusToCommunity(ctx, community)
	if err != nil {
		logger.Debug("m.sendCurrentUserStatusToCommunity error", zap.Error(err))
		return nil, err
	}

	return response, nil
}

func (m *Messenger) SetMuted(communityID types.HexBytes, muted bool) error {
	return m.communitiesManager.SetMuted(communityID, muted)
}

func (m *Messenger) RequestToJoinCommunity(request *requests.RequestToJoinCommunity) (*MessengerResponse, error) {
	logger := m.logger.Named("RequestToJoinCommunity")
	if err := request.Validate(); err != nil {
		logger.Debug("request failed to validate", zap.Error(err), zap.Any("request", request))
		return nil, err
	}

	community, requestToJoin, err := m.communitiesManager.RequestToJoin(&m.identity.PublicKey, request)
	if err != nil {
		return nil, err
	}
	err = m.syncCommunity(context.Background(), community)
	if err != nil {
		return nil, err
	}

	requestToJoinProto := &protobuf.CommunityRequestToJoin{
		Clock:       requestToJoin.Clock,
		EnsName:     requestToJoin.ENSName,
		CommunityId: community.ID(),
	}

	payload, err := proto.Marshal(requestToJoinProto)
	if err != nil {
		return nil, err
	}

	rawMessage := common.RawMessage{
		Payload:        payload,
		SkipEncryption: true,
		MessageType:    protobuf.ApplicationMetadataMessage_COMMUNITY_REQUEST_TO_JOIN,
	}
	_, err = m.sender.SendCommunityMessage(context.Background(), community.PublicKey(), rawMessage)
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

	return response, nil
}

func (m *Messenger) CreateCommunityCategory(request *requests.CreateCommunityCategory) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	var response MessengerResponse
	community, changes, err := m.communitiesManager.CreateCategory(request)
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

func (m *Messenger) AcceptRequestToJoinCommunity(request *requests.AcceptRequestToJoinCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, err := m.communitiesManager.AcceptRequestToJoin(request)

	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddCommunity(community)
	return response, nil
}

func (m *Messenger) DeclineRequestToJoinCommunity(request *requests.DeclineRequestToJoinCommunity) error {
	if err := request.Validate(); err != nil {
		return err
	}

	return m.communitiesManager.DeclineRequestToJoin(request)
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
		err = m.syncCommunity(context.Background(), com)
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
		err := m.deleteChat(communityChatID)
		if err != nil {
			return nil, err
		}
		response.AddRemovedChat(communityChatID)

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
	community, changes, err := m.communitiesManager.CreateChat(communityID, c)
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

func (m *Messenger) CreateCommunity(request *requests.CreateCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, err := m.communitiesManager.CreateCommunity(request)
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

	response := &MessengerResponse{}
	response.AddCommunity(community)
	response.AddCommunitySettings(&communitySettings)
	err = m.syncCommunity(context.Background(), community)
	if err != nil {
		return nil, err
	}

	if m.config.torrentConfig != nil && m.config.torrentConfig.Enabled && communitySettings.HistoryArchiveSupportEnabled {
		go m.communitiesManager.StartHistoryArchiveTasksInterval(community, messageArchiveInterval)
	}

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

	if m.config.torrentConfig.Enabled {
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

	//request info already stored on mailserver, but its success is not crucial
	// for import
	_, _ = m.RequestCommunityInfoFromMailserver(community.IDString())

	// We add ourselves
	_, err = m.communitiesManager.InviteUsersToCommunity(community.ID(), []*ecdsa.PublicKey{&m.identity.PublicKey})
	if err != nil {
		return nil, err
	}

	response, err := m.JoinCommunity(ctx, community.ID())
	if err != nil {
		return nil, err
	}

	if m.config.torrentConfig != nil && m.config.torrentConfig.Enabled {
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

func (m *Messenger) MyPendingRequestsToJoin() ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.PendingRequestsToJoinForUser(&m.identity.PublicKey)
}

func (m *Messenger) PendingRequestsToJoinForCommunity(id types.HexBytes) ([]*communities.RequestToJoin, error) {
	return m.communitiesManager.PendingRequestsToJoinForCommunity(id)
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

func (m *Messenger) BanUserFromCommunity(request *requests.BanUserFromCommunity) (*MessengerResponse, error) {
	community, err := m.communitiesManager.BanUserFromCommunity(request)
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

// RequestCommunityInfoFromMailserver installs filter for community and requests its details
// from mailserver. It waits until it  has the community before returning it
func (m *Messenger) RequestCommunityInfoFromMailserver(communityID string) (*communities.Community, error) {
	return m.requestCommunityInfoFromMailserver(communityID, true)
}

// RequestCommunityInfoFromMailserverAsync installs filter for community and requests its details
// from mailserver. When response received it will be passed through signals handler
func (m *Messenger) RequestCommunityInfoFromMailserverAsync(communityID string) error {
	_, err := m.requestCommunityInfoFromMailserver(communityID, false)
	return err
}

// RequestCommunityInfoFromMailserver installs filter for community and requests its details
// from mailserver. When response received it will be passed through signals handler
func (m *Messenger) requestCommunityInfoFromMailserver(communityID string, waitForResponse bool) (*communities.Community, error) {
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

	//if there is no info helpful for client, we don't post it
	if community.Name() == "" && community.DescriptionText() == "" && community.MembersCount() == 0 {
		return
	}

	if err != nil {
		m.logger.Warn("cant get community and pass it to signal handler", zap.Error(err))
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
			oldChat.Color != chat.Color {
			oldChat.Name = chat.Name
			oldChat.Description = chat.Description
			oldChat.Emoji = chat.Emoji
			oldChat.Color = chat.Color
			// TODO(samyoul) remove storing of an updated reference pointer?
			state.AllChats.Store(chat.ID, oldChat)
			state.Response.AddChat(chat)
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

func (m *Messenger) InitHistoryArchiveTasks(communities []*communities.Community) {

	for _, c := range communities {

		if c.Joined() {
			settings, err := m.communitiesManager.GetCommunitySettingsByID(c.ID())
			if err != nil {
				m.logger.Debug("failed to get community settings", zap.Error(err))
				continue
			}
			if !settings.HistoryArchiveSupportEnabled {
				continue
			}

			filters, err := m.communitiesManager.GetCommunityChatsFilters(c.ID())
			if err != nil {
				m.logger.Debug("failed to get community chats filters", zap.Error(err))
				continue
			}

			if len(filters) == 0 {
				m.logger.Debug("no filters or chats for this community starting interval", zap.String("id", c.IDString()))
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
				m.logger.Debug("failed to get Latest waku message timestamp", zap.Error(err))
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
				m.logger.Debug("failed to request missing messages", zap.Error(err))
				continue
			}

			// We figure out the end date of the last created archive and schedule
			// the interval for creating future archives
			// If the last end date is at least `interval` ago, we create an archive immediately first
			lastArchiveEndDateTimestamp, err := m.communitiesManager.GetHistoryArchivePartitionStartTimestamp(c.ID())
			if err != nil {
				m.logger.Debug("failed to get archive partition start timestamp", zap.Error(err))
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
					m.logger.Debug("failed to seed history archive", zap.Error(err))
				}
				timeToNextInterval := messageArchiveInterval - durationSinceLastArchive

				m.logger.Debug("Starting history archive tasks interval in", zap.Any("timeLeft", timeToNextInterval))
				time.AfterFunc(timeToNextInterval, func() {
					err := m.communitiesManager.CreateAndSeedHistoryArchive(c.ID(), topics, lastArchiveEndDate, to.Add(timeToNextInterval), messageArchiveInterval)
					if err != nil {
						m.logger.Debug("failed to get create and seed history archive", zap.Error(err))
					}
					go m.communitiesManager.StartHistoryArchiveTasksInterval(c, messageArchiveInterval)
				})
			} else {
				// Looks like the last archive was generated more than `interval`
				// ago, so lets create a new archive now and then schedule the archive
				// creation loop
				err := m.communitiesManager.CreateAndSeedHistoryArchive(c.ID(), topics, lastArchiveEndDate, to, messageArchiveInterval)
				if err != nil {
					m.logger.Debug("failed to get create and seed history archive", zap.Error(err))
				}

				go m.communitiesManager.StartHistoryArchiveTasksInterval(c, messageArchiveInterval)
			}
		}
	}
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
