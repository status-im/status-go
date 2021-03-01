package protocol

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/ens"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/transport"
	v1protocol "github.com/status-im/status-go/protocol/v1"

	"go.uber.org/zap"
)

const (
	transactionRequestDeclinedMessage           = "Transaction request declined"
	requestAddressForTransactionAcceptedMessage = "Request address for transaction accepted"
	requestAddressForTransactionDeclinedMessage = "Request address for transaction declined"
)

type MessageHandler struct {
	identity           *ecdsa.PrivateKey
	persistence        *sqlitePersistence
	transport          transport.Transport
	ensVerifier        *ens.Verifier
	communitiesManager *communities.Manager
	logger             *zap.Logger
}

func newMessageHandler(identity *ecdsa.PrivateKey, logger *zap.Logger, persistence *sqlitePersistence, communitiesManager *communities.Manager, transport transport.Transport, ensVerifier *ens.Verifier) *MessageHandler {
	return &MessageHandler{
		identity:           identity,
		persistence:        persistence,
		communitiesManager: communitiesManager,
		ensVerifier:        ensVerifier,
		transport:          transport,
		logger:             logger}
}

// HandleMembershipUpdate updates a Chat instance according to the membership updates.
// It retrieves chat, if exists, and merges membership updates from the message.
// Finally, the Chat is updated with the new group events.
func (m *MessageHandler) HandleMembershipUpdate(messageState *ReceivedMessageState, chat *Chat, rawMembershipUpdate protobuf.MembershipUpdateMessage, translations map[protobuf.MembershipUpdateEvent_EventType]string) error {
	var group *v1protocol.Group
	var err error

	logger := m.logger.With(zap.String("site", "HandleMembershipUpdate"))

	message, err := v1protocol.MembershipUpdateMessageFromProtobuf(&rawMembershipUpdate)
	if err != nil {
		return err

	}

	if err := ValidateMembershipUpdateMessage(message, messageState.Timesource.GetCurrentTime()); err != nil {
		logger.Warn("failed to validate message", zap.Error(err))
		return err
	}

	//if chat.InvitationAdmin exists means we are waiting for invitation request approvement, and in that case
	//we need to create a new chat instance like we don't have a chat and just use a regular invitation flow
	if chat == nil || len(chat.InvitationAdmin) > 0 {
		if len(message.Events) == 0 {
			return errors.New("can't create new group chat without events")
		}

		//approve invitations
		if chat != nil && len(chat.InvitationAdmin) > 0 {

			groupChatInvitation := &GroupChatInvitation{
				GroupChatInvitation: protobuf.GroupChatInvitation{
					ChatId: message.ChatID,
				},
				From: types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
			}

			groupChatInvitation, err = m.persistence.InvitationByID(groupChatInvitation.ID())
			if err != nil && err != common.ErrRecordNotFound {
				return err
			}
			if groupChatInvitation != nil {
				groupChatInvitation.State = protobuf.GroupChatInvitation_APPROVED

				err := m.persistence.SaveInvitation(groupChatInvitation)
				if err != nil {
					return err
				}
				messageState.GroupChatInvitations[groupChatInvitation.ID()] = groupChatInvitation
			}
		}

		group, err = v1protocol.NewGroupWithEvents(message.ChatID, message.Events)
		if err != nil {
			return err
		}

		// A new chat must contain us
		if !group.IsMember(contactIDFromPublicKey(&m.identity.PublicKey)) {
			return errors.New("can't create a new group chat without us being a member")
		}
		newChat := CreateGroupChat(messageState.Timesource)
		chat = &newChat
	} else {
		existingGroup, err := newProtocolGroupFromChat(chat)
		if err != nil {
			return errors.Wrap(err, "failed to create a Group from Chat")
		}
		updateGroup, err := v1protocol.NewGroupWithEvents(message.ChatID, message.Events)
		if err != nil {
			return errors.Wrap(err, "invalid membership update")
		}
		merged := v1protocol.MergeMembershipUpdateEvents(existingGroup.Events(), updateGroup.Events())
		group, err = v1protocol.NewGroupWithEvents(chat.ID, merged)
		if err != nil {
			return errors.Wrap(err, "failed to create a group with new membership updates")
		}
	}

	chat.updateChatFromGroupMembershipChanges(contactIDFromPublicKey(&m.identity.PublicKey), group)

	systemMessages := buildSystemMessages(message.Events, translations)

	for _, message := range systemMessages {
		messageID := message.ID
		exists, err := m.messageExists(messageID, messageState.ExistingMessagesMap)
		if err != nil {
			m.logger.Warn("failed to check message exists", zap.Error(err))
		}
		if exists {
			continue
		}
		messageState.Response.Messages = append(messageState.Response.Messages, message)
	}

	// Store in chats map as it might be a new one
	messageState.AllChats[chat.ID] = chat
	messageState.Response.AddChat(chat)

	if message.Message != nil {
		messageState.CurrentMessageState.Message = *message.Message
		return m.HandleChatMessage(messageState)
	} else if message.EmojiReaction != nil {
		return m.HandleEmojiReaction(messageState, *message.EmojiReaction)
	}

	return nil
}

func (m *MessageHandler) handleCommandMessage(state *ReceivedMessageState, message *common.Message) error {
	message.ID = state.CurrentMessageState.MessageID
	message.From = state.CurrentMessageState.Contact.ID
	message.Alias = state.CurrentMessageState.Contact.Alias
	message.SigPubKey = state.CurrentMessageState.PublicKey
	message.Identicon = state.CurrentMessageState.Contact.Identicon
	message.WhisperTimestamp = state.CurrentMessageState.WhisperTimestamp

	if err := message.PrepareContent(); err != nil {
		return fmt.Errorf("failed to prepare content: %v", err)
	}
	chat, err := m.matchChatEntity(message, state.AllChats, state.Timesource)
	if err != nil {
		return err
	}

	// If deleted-at is greater, ignore message
	if chat.DeletedAtClockValue >= message.Clock {
		return nil
	}

	// Set the LocalChatID for the message
	message.LocalChatID = chat.ID

	if c, ok := state.AllChats[chat.ID]; ok {
		chat = c
	}

	// Set the LocalChatID for the message
	message.LocalChatID = chat.ID

	// Increase unviewed count
	if !common.IsPubKeyEqual(message.SigPubKey, &m.identity.PublicKey) {
		chat.UnviewedMessagesCount++
		message.OutgoingStatus = ""
	} else {
		// Our own message, mark as sent
		message.OutgoingStatus = common.OutgoingStatusSent
	}

	err = chat.UpdateFromMessage(message, state.Timesource)
	if err != nil {
		return err
	}

	// Set chat active
	chat.Active = true
	// Set in the modified maps chat
	state.Response.AddChat(chat)
	state.AllChats[chat.ID] = chat

	// Add to response
	if message != nil {
		state.Response.Messages = append(state.Response.Messages, message)
	}
	return nil
}

func (m *MessageHandler) HandleSyncInstallationContact(state *ReceivedMessageState, message protobuf.SyncInstallationContact) error {
	chat, ok := state.AllChats[state.CurrentMessageState.Contact.ID]
	if !ok {
		chat = OneToOneFromPublicKey(state.CurrentMessageState.PublicKey, state.Timesource)
		// We don't want to show the chat to the user
		chat.Active = false
	}

	contact, ok := state.AllContacts[message.Id]
	if !ok {
		var err error
		contact, err = buildContactFromPkString(message.Id)
		if err != nil {
			return err
		}
	}

	if contact.LastUpdated < message.Clock {
		if !contact.IsAdded() {
			contact.SystemTags = append(contact.SystemTags, contactAdded)
		}
		if contact.Name != message.EnsName {
			contact.Name = message.EnsName
			contact.ENSVerified = false
		}
		contact.LastUpdated = message.Clock
		contact.LocalNickname = message.LocalNickname

		state.ModifiedContacts[contact.ID] = true
		state.AllContacts[contact.ID] = contact
	}

	state.AllChats[chat.ID] = chat

	return nil
}

func (m *MessageHandler) HandleSyncInstallationPublicChat(state *ReceivedMessageState, message protobuf.SyncInstallationPublicChat) bool {
	chatID := message.Id
	_, ok := state.AllChats[chatID]
	if ok {
		return false
	}

	chat := CreatePublicChat(chatID, state.Timesource)

	state.AllChats[chat.ID] = chat
	state.Response.AddChat(chat)

	return true
}

func (m *MessageHandler) HandleContactUpdate(state *ReceivedMessageState, message protobuf.ContactUpdate) error {
	logger := m.logger.With(zap.String("site", "HandleContactUpdate"))
	contact := state.CurrentMessageState.Contact
	chat, ok := state.AllChats[contact.ID]
	if !ok {
		chat = OneToOneFromPublicKey(state.CurrentMessageState.PublicKey, state.Timesource)
		// We don't want to show the chat to the user
		chat.Active = false
	}

	logger.Info("Handling contact update")

	if contact.LastUpdated < message.Clock {
		logger.Info("Updating contact")
		if !contact.HasBeenAdded() && contact.ID != contactIDFromPublicKey(&m.identity.PublicKey) {
			contact.SystemTags = append(contact.SystemTags, contactRequestReceived)
		}
		if contact.Name != message.EnsName {
			contact.Name = message.EnsName
			contact.ENSVerified = false
		}
		contact.LastUpdated = message.Clock
		state.ModifiedContacts[contact.ID] = true
		state.AllContacts[contact.ID] = contact
	}

	if chat.LastClockValue < message.Clock {
		chat.LastClockValue = message.Clock
	}

	state.Response.AddChat(chat)
	state.AllChats[chat.ID] = chat

	return nil
}

func (m *MessageHandler) HandlePairInstallation(state *ReceivedMessageState, message protobuf.PairInstallation) error {
	logger := m.logger.With(zap.String("site", "HandlePairInstallation"))
	if err := ValidateReceivedPairInstallation(&message, state.CurrentMessageState.WhisperTimestamp); err != nil {
		logger.Warn("failed to validate message", zap.Error(err))
		return err
	}

	installation, ok := state.AllInstallations[message.InstallationId]
	if !ok {
		return errors.New("installation not found")
	}

	metadata := &multidevice.InstallationMetadata{
		Name:       message.Name,
		DeviceType: message.DeviceType,
	}

	installation.InstallationMetadata = metadata
	state.AllInstallations[message.InstallationId] = installation
	state.ModifiedInstallations[message.InstallationId] = true

	return nil
}

// HandleCommunityDescription handles an community description
func (m *MessageHandler) HandleCommunityDescription(state *ReceivedMessageState, signer *ecdsa.PublicKey, description protobuf.CommunityDescription, rawPayload []byte) error {
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

		oldChat, ok := state.AllChats[chat.ID]
		if !ok {
			// Beware, don't use the reference in the range (i.e chat) as it's a shallow copy
			state.AllChats[chat.ID] = chats[i]

			state.Response.AddChat(chat)
			chatIDs = append(chatIDs, chat.ID)
			// Update name, currently is the only field is mutable
		} else if oldChat.Name != chat.Name {
			state.AllChats[chat.ID].Name = chat.Name
			state.Response.AddChat(chat)
		}
	}

	// Load transport filters
	filters, err := m.transport.InitPublicFilters(chatIDs)
	if err != nil {
		return err
	}

	for _, filter := range filters {
		state.AllFilters[filter.ChatID] = filter
	}

	return nil
}

// HandleCommunityInvitation handles an community invitation
func (m *MessageHandler) HandleCommunityInvitation(state *ReceivedMessageState, signer *ecdsa.PublicKey, invitation protobuf.CommunityInvitation, rawPayload []byte) error {
	if invitation.PublicKey == nil {
		return errors.New("invalid pubkey")
	}
	pk, err := crypto.DecompressPubkey(invitation.PublicKey)
	if err != nil {
		return err
	}

	if !common.IsPubKeyEqual(pk, &m.identity.PublicKey) {
		return errors.New("invitation not for us")
	}

	communityResponse, err := m.communitiesManager.HandleCommunityInvitation(signer, &invitation, rawPayload)
	if err != nil {
		return err
	}

	community := communityResponse.Community

	state.Response.AddCommunity(community)
	state.Response.CommunityChanges = append(state.Response.CommunityChanges, communityResponse.Changes)

	return nil
}

// HandleCommunityRequestToJoin handles an community request to join
func (m *MessageHandler) HandleCommunityRequestToJoin(state *ReceivedMessageState, signer *ecdsa.PublicKey, requestToJoinProto protobuf.CommunityRequestToJoin) error {
	if requestToJoinProto.CommunityId == nil {
		return errors.New("invalid community id")
	}

	requestToJoin, err := m.communitiesManager.HandleCommunityRequestToJoin(signer, &requestToJoinProto)
	if err != nil {
		return err
	}

	state.Response.RequestsToJoinCommunity = append(state.Response.RequestsToJoinCommunity, requestToJoin)

	community, err := m.communitiesManager.GetByID(requestToJoinProto.CommunityId)
	if err != nil {
		return err
	}

	contactID := contactIDFromPublicKey(signer)

	contact := state.AllContacts[contactID]

	state.Response.AddNotification(NewCommunityRequestToJoinNotification(requestToJoin.ID.String(), community, contact))

	return nil
}

// handleWrappedCommunityDescriptionMessage handles a wrapped community description
func (m *MessageHandler) handleWrappedCommunityDescriptionMessage(payload []byte) (*communities.CommunityResponse, error) {
	return m.communitiesManager.HandleWrappedCommunityDescriptionMessage(payload)
}

func (m *MessageHandler) HandleChatMessage(state *ReceivedMessageState) error {
	logger := m.logger.With(zap.String("site", "handleChatMessage"))
	if err := ValidateReceivedChatMessage(&state.CurrentMessageState.Message, state.CurrentMessageState.WhisperTimestamp); err != nil {
		logger.Warn("failed to validate message", zap.Error(err))
		return err
	}
	receivedMessage := &common.Message{
		ID:               state.CurrentMessageState.MessageID,
		ChatMessage:      state.CurrentMessageState.Message,
		From:             state.CurrentMessageState.Contact.ID,
		Alias:            state.CurrentMessageState.Contact.Alias,
		SigPubKey:        state.CurrentMessageState.PublicKey,
		Identicon:        state.CurrentMessageState.Contact.Identicon,
		WhisperTimestamp: state.CurrentMessageState.WhisperTimestamp,
	}

	err := receivedMessage.PrepareContent()
	if err != nil {
		return fmt.Errorf("failed to prepare message content: %v", err)
	}
	chat, err := m.matchChatEntity(receivedMessage, state.AllChats, state.Timesource)
	if err != nil {
		return err // matchChatEntity returns a descriptive error message
	}

	// It looks like status-react created profile chats as public chats
	// so for now we need to check for the presence of "@" in their chatID
	if chat.Public() && receivedMessage.ContentType == protobuf.ChatMessage_IMAGE && !chat.ProfileUpdates() {
		return errors.New("images are not allowed in public chats")
	}

	// If profile updates check if author is the same as chat profile public key
	if chat.ProfileUpdates() && receivedMessage.From != chat.Profile {
		return nil
	}

	// If deleted-at is greater, ignore message
	if chat.DeletedAtClockValue >= receivedMessage.Clock {
		return nil
	}

	// Set the LocalChatID for the message
	receivedMessage.LocalChatID = chat.ID

	if c, ok := state.AllChats[chat.ID]; ok {
		chat = c
	}

	// Set the LocalChatID for the message
	receivedMessage.LocalChatID = chat.ID

	// Increase unviewed count
	if !common.IsPubKeyEqual(receivedMessage.SigPubKey, &m.identity.PublicKey) {
		chat.UnviewedMessagesCount++
	} else {
		// Our own message, mark as sent
		receivedMessage.OutgoingStatus = common.OutgoingStatusSent
	}

	err = chat.UpdateFromMessage(receivedMessage, state.Timesource)
	if err != nil {
		return err
	}

	// Set chat active
	chat.Active = true
	// Set in the modified maps chat
	state.Response.AddChat(chat)
	state.AllChats[chat.ID] = chat

	contact := state.CurrentMessageState.Contact
	if receivedMessage.EnsName != "" {
		oldRecord, err := m.ensVerifier.Add(contact.ID, receivedMessage.EnsName, receivedMessage.Clock)
		if err != nil {
			m.logger.Warn("failed to verify ENS name", zap.Error(err))
		} else if oldRecord == nil {
			// If oldRecord is nil, a new verification process will take place
			// so we reset the record
			contact.ENSVerified = false
			state.ModifiedContacts[contact.ID] = true
			state.AllContacts[contact.ID] = contact
		}
	}

	if receivedMessage.ContentType == protobuf.ChatMessage_COMMUNITY {
		m.logger.Debug("Handling community content type")

		communityResponse, err := m.handleWrappedCommunityDescriptionMessage(receivedMessage.GetCommunity())
		if err != nil {
			return err
		}
		community := communityResponse.Community
		receivedMessage.CommunityID = community.IDString()

		state.Response.AddCommunity(community)
		state.Response.CommunityChanges = append(state.Response.CommunityChanges, communityResponse.Changes)
	}
	// Add to response
	state.Response.Messages = append(state.Response.Messages, receivedMessage)

	return nil
}

func (m *MessageHandler) HandleRequestAddressForTransaction(messageState *ReceivedMessageState, command protobuf.RequestAddressForTransaction) error {
	err := ValidateReceivedRequestAddressForTransaction(&command, messageState.CurrentMessageState.WhisperTimestamp)
	if err != nil {
		return err
	}
	message := &common.Message{
		ChatMessage: protobuf.ChatMessage{
			Clock:       command.Clock,
			Timestamp:   messageState.CurrentMessageState.WhisperTimestamp,
			Text:        "Request address for transaction",
			ChatId:      contactIDFromPublicKey(&m.identity.PublicKey),
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			ContentType: protobuf.ChatMessage_TRANSACTION_COMMAND,
		},
		CommandParameters: &common.CommandParameters{
			ID:           messageState.CurrentMessageState.MessageID,
			Value:        command.Value,
			Contract:     command.Contract,
			CommandState: common.CommandStateRequestAddressForTransaction,
		},
	}
	return m.handleCommandMessage(messageState, message)
}

func (m *MessageHandler) HandleRequestTransaction(messageState *ReceivedMessageState, command protobuf.RequestTransaction) error {
	err := ValidateReceivedRequestTransaction(&command, messageState.CurrentMessageState.WhisperTimestamp)
	if err != nil {
		return err
	}
	message := &common.Message{
		ChatMessage: protobuf.ChatMessage{
			Clock:       command.Clock,
			Timestamp:   messageState.CurrentMessageState.WhisperTimestamp,
			Text:        "Request transaction",
			ChatId:      contactIDFromPublicKey(&m.identity.PublicKey),
			MessageType: protobuf.MessageType_ONE_TO_ONE,
			ContentType: protobuf.ChatMessage_TRANSACTION_COMMAND,
		},
		CommandParameters: &common.CommandParameters{
			ID:           messageState.CurrentMessageState.MessageID,
			Value:        command.Value,
			Contract:     command.Contract,
			CommandState: common.CommandStateRequestTransaction,
			Address:      command.Address,
		},
	}
	return m.handleCommandMessage(messageState, message)
}

func (m *MessageHandler) HandleAcceptRequestAddressForTransaction(messageState *ReceivedMessageState, command protobuf.AcceptRequestAddressForTransaction) error {
	err := ValidateReceivedAcceptRequestAddressForTransaction(&command, messageState.CurrentMessageState.WhisperTimestamp)
	if err != nil {
		return err
	}
	initialMessage, err := m.persistence.MessageByID(command.Id)
	if err != nil {
		return err
	}
	if initialMessage == nil {
		return errors.New("message not found")
	}

	if initialMessage.LocalChatID != messageState.CurrentMessageState.Contact.ID {
		return errors.New("From must match")
	}

	if initialMessage.OutgoingStatus == "" {
		return errors.New("Initial message must originate from us")
	}

	if initialMessage.CommandParameters.CommandState != common.CommandStateRequestAddressForTransaction {
		return errors.New("Wrong state for command")
	}

	initialMessage.Clock = command.Clock
	initialMessage.Timestamp = messageState.CurrentMessageState.WhisperTimestamp
	initialMessage.Text = requestAddressForTransactionAcceptedMessage
	initialMessage.CommandParameters.Address = command.Address
	initialMessage.Seen = false
	initialMessage.CommandParameters.CommandState = common.CommandStateRequestAddressForTransactionAccepted

	// Hide previous message
	previousMessage, err := m.persistence.MessageByCommandID(messageState.CurrentMessageState.Contact.ID, command.Id)
	if err != nil && err != common.ErrRecordNotFound {
		return err
	}

	if previousMessage != nil {
		err = m.persistence.HideMessage(previousMessage.ID)
		if err != nil {
			return err
		}

		initialMessage.Replace = previousMessage.ID
	}

	return m.handleCommandMessage(messageState, initialMessage)
}

func (m *MessageHandler) HandleSendTransaction(messageState *ReceivedMessageState, command protobuf.SendTransaction) error {
	err := ValidateReceivedSendTransaction(&command, messageState.CurrentMessageState.WhisperTimestamp)
	if err != nil {
		return err
	}
	transactionToValidate := &TransactionToValidate{
		MessageID:       messageState.CurrentMessageState.MessageID,
		CommandID:       command.Id,
		TransactionHash: command.TransactionHash,
		FirstSeen:       messageState.CurrentMessageState.WhisperTimestamp,
		Signature:       command.Signature,
		Validate:        true,
		From:            messageState.CurrentMessageState.PublicKey,
		RetryCount:      0,
	}
	m.logger.Info("Saving transction to validate", zap.Any("transaction", transactionToValidate))

	return m.persistence.SaveTransactionToValidate(transactionToValidate)
}

func (m *MessageHandler) HandleDeclineRequestAddressForTransaction(messageState *ReceivedMessageState, command protobuf.DeclineRequestAddressForTransaction) error {
	err := ValidateReceivedDeclineRequestAddressForTransaction(&command, messageState.CurrentMessageState.WhisperTimestamp)
	if err != nil {
		return err
	}
	oldMessage, err := m.persistence.MessageByID(command.Id)
	if err != nil {
		return err
	}
	if oldMessage == nil {
		return errors.New("message not found")
	}

	if oldMessage.LocalChatID != messageState.CurrentMessageState.Contact.ID {
		return errors.New("From must match")
	}

	if oldMessage.OutgoingStatus == "" {
		return errors.New("Initial message must originate from us")
	}

	if oldMessage.CommandParameters.CommandState != common.CommandStateRequestAddressForTransaction {
		return errors.New("Wrong state for command")
	}

	oldMessage.Clock = command.Clock
	oldMessage.Timestamp = messageState.CurrentMessageState.WhisperTimestamp
	oldMessage.Text = requestAddressForTransactionDeclinedMessage
	oldMessage.Seen = false
	oldMessage.CommandParameters.CommandState = common.CommandStateRequestAddressForTransactionDeclined

	// Hide previous message
	err = m.persistence.HideMessage(command.Id)
	if err != nil {
		return err
	}
	oldMessage.Replace = command.Id

	return m.handleCommandMessage(messageState, oldMessage)
}

func (m *MessageHandler) HandleDeclineRequestTransaction(messageState *ReceivedMessageState, command protobuf.DeclineRequestTransaction) error {
	err := ValidateReceivedDeclineRequestTransaction(&command, messageState.CurrentMessageState.WhisperTimestamp)
	if err != nil {
		return err
	}
	oldMessage, err := m.persistence.MessageByID(command.Id)
	if err != nil {
		return err
	}
	if oldMessage == nil {
		return errors.New("message not found")
	}

	if oldMessage.LocalChatID != messageState.CurrentMessageState.Contact.ID {
		return errors.New("From must match")
	}

	if oldMessage.OutgoingStatus == "" {
		return errors.New("Initial message must originate from us")
	}

	if oldMessage.CommandParameters.CommandState != common.CommandStateRequestTransaction {
		return errors.New("Wrong state for command")
	}

	oldMessage.Clock = command.Clock
	oldMessage.Timestamp = messageState.CurrentMessageState.WhisperTimestamp
	oldMessage.Text = transactionRequestDeclinedMessage
	oldMessage.Seen = false
	oldMessage.CommandParameters.CommandState = common.CommandStateRequestTransactionDeclined

	// Hide previous message
	err = m.persistence.HideMessage(command.Id)
	if err != nil {
		return err
	}
	oldMessage.Replace = command.Id

	return m.handleCommandMessage(messageState, oldMessage)
}

func (m *MessageHandler) matchChatEntity(chatEntity common.ChatEntity, chats map[string]*Chat, timesource common.TimeSource) (*Chat, error) {
	if chatEntity.GetSigPubKey() == nil {
		m.logger.Error("public key can't be empty")
		return nil, errors.New("received a chatEntity with empty public key")
	}

	switch {
	case chatEntity.GetMessageType() == protobuf.MessageType_PUBLIC_GROUP:
		// For public messages, all outgoing and incoming messages have the same chatID
		// equal to a public chat name.
		chatID := chatEntity.GetChatId()
		chat := chats[chatID]
		if chat == nil {
			return nil, errors.New("received a public chatEntity from non-existing chat")
		}
		return chat, nil
	case chatEntity.GetMessageType() == protobuf.MessageType_ONE_TO_ONE && common.IsPubKeyEqual(chatEntity.GetSigPubKey(), &m.identity.PublicKey):
		// It's a private message coming from us so we rely on Message.ChatID
		// If chat does not exist, it should be created to support multidevice synchronization.
		chatID := chatEntity.GetChatId()
		chat := chats[chatID]
		if chat == nil {
			if len(chatID) != PubKeyStringLength {
				return nil, errors.New("invalid pubkey length")
			}
			bytePubKey, err := hex.DecodeString(chatID[2:])
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode hex chatID")
			}

			pubKey, err := crypto.UnmarshalPubkey(bytePubKey)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode pubkey")
			}

			chat = CreateOneToOneChat(chatID[:8], pubKey, timesource)
		}
		return chat, nil
	case chatEntity.GetMessageType() == protobuf.MessageType_ONE_TO_ONE:
		// It's an incoming private chatEntity. ChatID is calculated from the signature.
		// If a chat does not exist, a new one is created and saved.
		chatID := contactIDFromPublicKey(chatEntity.GetSigPubKey())
		chat := chats[chatID]
		if chat == nil {
			// TODO: this should be a three-word name used in the mobile client
			chat = CreateOneToOneChat(chatID[:8], chatEntity.GetSigPubKey(), timesource)
		}
		return chat, nil
	case chatEntity.GetMessageType() == protobuf.MessageType_COMMUNITY_CHAT:
		chatID := chatEntity.GetChatId()
		chat := chats[chatID]
		if chat == nil {
			return nil, errors.New("received community chat chatEntity for non-existing chat")
		}

		if chat.CommunityID == "" || chat.ChatType != ChatTypeCommunityChat {
			return nil, errors.New("not an community chat")
		}

		var emojiReaction bool
		// We allow emoji reactions from anyone
		switch chatEntity.(type) {
		case *EmojiReaction:
			emojiReaction = true
		}

		canPost, err := m.communitiesManager.CanPost(chatEntity.GetSigPubKey(), chat.CommunityID, chat.CommunityChatID(), chatEntity.GetGrant())
		if err != nil {
			return nil, err
		}
		if !emojiReaction && !canPost {
			return nil, errors.New("user can't post")
		}

		return chat, nil
	case chatEntity.GetMessageType() == protobuf.MessageType_PRIVATE_GROUP:
		// In the case of a group chatEntity, ChatID is the same for all messages belonging to a group.
		// It needs to be verified if the signature public key belongs to the chat.
		chatID := chatEntity.GetChatId()
		chat := chats[chatID]
		if chat == nil {
			return nil, errors.New("received group chat chatEntity for non-existing chat")
		}

		theirKeyHex := contactIDFromPublicKey(chatEntity.GetSigPubKey())
		myKeyHex := contactIDFromPublicKey(&m.identity.PublicKey)
		var theyJoined bool
		var iJoined bool
		for _, member := range chat.Members {
			if member.ID == theirKeyHex && member.Joined {
				theyJoined = true
			}
		}
		for _, member := range chat.Members {
			if member.ID == myKeyHex && member.Joined {
				iJoined = true
			}
		}

		if theyJoined && iJoined {
			return chat, nil
		}

		return nil, errors.New("did not find a matching group chat")
	default:
		return nil, errors.New("can not match a chat because there is no valid case")
	}
}

func (m *MessageHandler) messageExists(messageID string, existingMessagesMap map[string]bool) (bool, error) {
	if _, ok := existingMessagesMap[messageID]; ok {
		return true, nil
	}

	existingMessagesMap[messageID] = true

	// Check against the database, this is probably a bit slow for
	// each message, but for now might do, we'll make it faster later
	existingMessage, err := m.persistence.MessageByID(messageID)
	if err != nil && err != common.ErrRecordNotFound {
		return false, err
	}
	if existingMessage != nil {
		return true, nil
	}
	return false, nil
}

func (m *MessageHandler) HandleEmojiReaction(state *ReceivedMessageState, pbEmojiR protobuf.EmojiReaction) error {
	logger := m.logger.With(zap.String("site", "HandleEmojiReaction"))
	if err := ValidateReceivedEmojiReaction(&pbEmojiR, state.Timesource.GetCurrentTime()); err != nil {
		logger.Error("invalid emoji reaction", zap.Error(err))
		return err
	}

	from := state.CurrentMessageState.Contact.ID

	emojiReaction := &EmojiReaction{
		EmojiReaction: pbEmojiR,
		From:          from,
		SigPubKey:     state.CurrentMessageState.PublicKey,
	}

	existingEmoji, err := m.persistence.EmojiReactionByID(emojiReaction.ID())
	if err != common.ErrRecordNotFound && err != nil {
		return err
	}

	if existingEmoji != nil && existingEmoji.Clock >= pbEmojiR.Clock {
		// this is not a valid emoji, ignoring
		return nil
	}

	chat, err := m.matchChatEntity(emojiReaction, state.AllChats, state.Timesource)
	if err != nil {
		return err // matchChatEntity returns a descriptive error message
	}

	// Set local chat id
	emojiReaction.LocalChatID = chat.ID

	logger.Debug("Handling emoji reaction")

	if chat.LastClockValue < pbEmojiR.Clock {
		chat.LastClockValue = pbEmojiR.Clock
	}

	state.Response.AddChat(chat)
	state.AllChats[chat.ID] = chat

	// save emoji reaction
	err = m.persistence.SaveEmojiReaction(emojiReaction)
	if err != nil {
		return err
	}

	state.EmojiReactions[emojiReaction.ID()] = emojiReaction

	return nil
}

func (m *MessageHandler) HandleGroupChatInvitation(state *ReceivedMessageState, pbGHInvitations protobuf.GroupChatInvitation) error {
	logger := m.logger.With(zap.String("site", "HandleGroupChatInvitation"))
	if err := ValidateReceivedGroupChatInvitation(&pbGHInvitations); err != nil {
		logger.Error("invalid group chat invitation", zap.Error(err))
		return err
	}

	groupChatInvitation := &GroupChatInvitation{
		GroupChatInvitation: pbGHInvitations,
		SigPubKey:           state.CurrentMessageState.PublicKey,
	}

	//From is the PK of author of invitation request
	if groupChatInvitation.State == protobuf.GroupChatInvitation_REJECTED {
		//rejected so From is the current user who received this rejection
		groupChatInvitation.From = types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey))
	} else {
		//invitation request, so From is the author of message
		groupChatInvitation.From = state.CurrentMessageState.Contact.ID
	}

	existingInvitation, err := m.persistence.InvitationByID(groupChatInvitation.ID())
	if err != common.ErrRecordNotFound && err != nil {
		return err
	}

	if existingInvitation != nil && existingInvitation.Clock >= pbGHInvitations.Clock {
		// this is not a valid invitation, ignoring
		return nil
	}

	// save invitation
	err = m.persistence.SaveInvitation(groupChatInvitation)
	if err != nil {
		return err
	}

	state.GroupChatInvitations[groupChatInvitation.ID()] = groupChatInvitation

	return nil
}

// HandleChatIdentity handles an incoming protobuf.ChatIdentity
// extracts contact information stored in the protobuf and adds it to the user's contact for update.
func (m *MessageHandler) HandleChatIdentity(state *ReceivedMessageState, ci protobuf.ChatIdentity) error {
	logger := m.logger.With(zap.String("site", "HandleChatIdentity"))
	contact := state.CurrentMessageState.Contact

	logger.Info("Handling contact update")
	newImages, err := m.persistence.SaveContactChatIdentity(contact.ID, &ci)
	if err != nil {
		return err
	}
	if newImages {
		for imageType, image := range ci.Images {
			if contact.Images == nil {
				contact.Images = make(map[string]images.IdentityImage)
			}
			contact.Images[imageType] = images.IdentityImage{Name: imageType, Payload: image.Payload}

		}
		state.ModifiedContacts[contact.ID] = true
		state.AllContacts[contact.ID] = contact
	}

	return nil
}
