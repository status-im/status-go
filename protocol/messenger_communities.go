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

const communityInvitationText = "Upgrade to see a community invitation"

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
	_, err = m.processor.SendPublic(context.Background(), org.IDString(), rawMessage)
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
	_, err = m.processor.SendPrivate(context.Background(), pk, &rawMessage)
	return err
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

func (m *Messenger) JoinCommunity(communityID types.HexBytes) (*MessengerResponse, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.joinCommunity(communityID)
}

func (m *Messenger) joinCommunity(communityID types.HexBytes) (*MessengerResponse, error) {
	response := &MessengerResponse{}

	community, err := m.communitiesManager.JoinCommunity(communityID)
	if err != nil {
		return nil, err
	}

	chatIDs := []string{community.IDString()}

	chats := CreateCommunityChats(community, m.getTimesource())
	response.AddChats(chats)

	for _, chat := range response.Chats() {
		chatIDs = append(chatIDs, chat.ID)
	}

	// Load transport filters
	filters, err := m.transport.InitPublicFilters(chatIDs)
	if err != nil {
		return nil, err
	}

	response.Filters = filters
	response.AddCommunity(community)

	return response, m.saveChats(chats)
}

func (m *Messenger) RequestToJoinCommunity(request *requests.RequestToJoinCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	community, requestToJoin, err := m.communitiesManager.RequestToJoin(&m.identity.PublicKey, request)
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
	_, err = m.processor.SendCommunityMessage(context.Background(), community.PublicKey(), rawMessage)
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
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.leaveCommunity(communityID)
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

		filter, err := m.transport.RemoveFilterByChatID(communityChatID)
		if err != nil {
			return nil, err
		}

		if filter != nil {
			response.RemovedFilters = append(response.RemovedFilters, filter)
		}
	}

	filter, err := m.transport.RemoveFilterByChatID(communityID.String())
	if err != nil {
		return nil, err
	}

	if filter != nil {
		response.RemovedFilters = append(response.RemovedFilters, filter)
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
	response.Filters = filters

	return &response, m.saveChats(chats)
}

func (m *Messenger) CreateCommunity(request *requests.CreateCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	description, err := request.ToCommunityDescription()
	if err != nil {
		return nil, err
	}

	description.Members = make(map[string]*protobuf.CommunityMember)
	description.Members[common.PubkeyToHex(&m.identity.PublicKey)] = &protobuf.CommunityMember{Roles: []protobuf.CommunityMember_Roles{protobuf.CommunityMember_ROLE_ALL}}

	community, err := m.communitiesManager.CreateCommunity(description)
	if err != nil {
		return nil, err
	}

	// Init the community filter so we can receive messages on the community
	filters, err := m.transport.InitCommunityFilters([]*ecdsa.PrivateKey{community.PrivateKey()})
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{
		Filters: filters,
	}

	response.AddCommunity(community)

	return response, nil
}

func (m *Messenger) ExportCommunity(id types.HexBytes) (*ecdsa.PrivateKey, error) {
	return m.communitiesManager.ExportCommunity(id)
}

func (m *Messenger) ImportCommunity(key *ecdsa.PrivateKey) (*MessengerResponse, error) {
	org, err := m.communitiesManager.ImportCommunity(key)
	if err != nil {
		return nil, err
	}

	// Load filters
	filters, err := m.transport.InitPublicFilters([]string{org.IDString()})
	if err != nil {
		return nil, err
	}

	//request info already stored on mailserver, but its success is not crucial
	// for import
	_ = m.RequestCommunityInfoFromMailserver(org.IDString())

	return &MessengerResponse{
		Filters: filters,
	}, nil
}

func (m *Messenger) InviteUsersToCommunity(request *requests.InviteUsersToCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	var messages []*common.Message

	var publicKeys []*ecdsa.PublicKey
	for _, pkBytes := range request.Users {
		publicKey, err := common.HexToPubkey(pkBytes.String())
		if err != nil {
			return nil, err
		}
		publicKeys = append(publicKeys, publicKey)

		message := &common.Message{}
		message.ChatId = pkBytes.String()
		message.CommunityID = request.CommunityID.String()
		message.Text = communityInvitationText
		messages = append(messages, message)
		r, err := m.CreateOneToOneChat(&requests.CreateOneToOneChat{ID: pkBytes})
		if err != nil {
			return nil, err
		}

		if err := response.Merge(r); err != nil {
			return nil, err
		}
	}

	community, err := m.communitiesManager.InviteUsersToCommunity(request.CommunityID, publicKeys)
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

func (m *Messenger) ShareCommunity(request *requests.ShareCommunity) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	response := &MessengerResponse{}

	var messages []*common.Message
	for _, pk := range request.Users {
		message := &common.Message{}
		message.ChatId = pk.String()
		message.CommunityID = request.CommunityID.String()
		message.Text = communityInvitationText
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
	response.AddCommunity(community)
	return response, nil
}

// RequestCommunityInfoFromMailserver installs filter for community and requests its details
// from mailserver. When response received it will be passed through signals handler
func (m *Messenger) RequestCommunityInfoFromMailserver(communityID string) error {

	if _, ok := m.requestedCommunities[communityID]; ok {
		return nil
	}

	//If filter wasn't installed we create it and remember for deinstalling after
	//response received
	filter := m.transport.FilterByChatID(communityID)
	if filter == nil {
		filters, err := m.transport.InitPublicFilters([]string{communityID})
		if err != nil {
			return fmt.Errorf("Can't install filter for community: %v", err)
		}
		if len(filters) != 1 {
			return fmt.Errorf("Unexpected amount of filters created")
		}
		filter = filters[0]
		m.requestedCommunities[communityID] = filter
	} else {
		//we don't remember filter id associated with community because it was already installed
		m.requestedCommunities[communityID] = nil
	}

	now := uint32(m.transport.GetCurrentTime() / 1000)
	monthAgo := now - (86400 * 30)

	_, err := m.RequestHistoricMessagesForFilter(context.Background(),
		monthAgo,
		now,
		nil,
		filter,
		false)

	//It is possible that we already processed last existing message for community
	//and won't get any updates, so send stored info in this case after timeout
	go func() {
		time.Sleep(15 * time.Second)
		m.mutex.Lock()
		defer m.mutex.Unlock()

		if _, ok := m.requestedCommunities[communityID]; ok {
			m.passStoredCommunityInfoToSignalHandler(communityID)
		}
	}()

	return err
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
	if community.Name() == "" && community.Description() == "" && community.MembersCount() == 0 {
		return
	}

	if err != nil {
		m.logger.Warn("cant get community and pass it to signal handler", zap.Error(err))
		return
	}

	m.config.messengerSignalsHandler.CommunityInfoFound(community)
	m.forgetCommunityRequest(communityID)
}
