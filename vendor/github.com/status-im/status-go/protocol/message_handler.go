package protocol

import (
	"crypto/ecdsa"
	"encoding/hex"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

type MessageHandler struct {
	identity    *ecdsa.PrivateKey
	persistence *sqlitePersistence
	logger      *zap.Logger
}

func newMessageHandler(identity *ecdsa.PrivateKey, logger *zap.Logger, persistence *sqlitePersistence) *MessageHandler {
	return &MessageHandler{
		identity:    identity,
		persistence: persistence,
		logger:      logger}
}

// HandleMembershipUpdate updates a Chat instance according to the membership updates.
// It retrieves chat, if exists, and merges membership updates from the message.
// Finally, the Chat is updated with the new group events.
func (m *MessageHandler) HandleMembershipUpdate(messageState *ReceivedMessageState, chat *Chat, rawMembershipUpdate protobuf.MembershipUpdateMessage, translations map[protobuf.MembershipUpdateEvent_EventType]string) error {
	var group *v1protocol.Group
	var err error

	message, err := v1protocol.MembershipUpdateMessageFromProtobuf(&rawMembershipUpdate)
	if err != nil {
		return err

	}

	if chat == nil {
		if len(message.Events) == 0 {
			return errors.New("can't create new group chat without events")
		}
		group, err = v1protocol.NewGroupWithEvents(message.ChatID, message.Events)
		if err != nil {
			return err
		}

		// A new chat must contain us
		if !group.IsMember(types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey))) {
			return errors.New("can't create a new group chat without us being a member")
		}
		newChat := createGroupChat()
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
		group, err = v1protocol.NewGroup(chat.ID, merged)
		if err != nil {
			return errors.Wrap(err, "failed to create a group with new membership updates")
		}
	}

	chat.updateChatFromProtocolGroup(group)
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
	// Set in the map
	messageState.ModifiedChats[chat.ID] = true

	if message.Message != nil {
		messageState.CurrentMessageState.Message = *message.Message
		return m.HandleChatMessage(messageState)
	}

	return nil
}

func (m *MessageHandler) handleCommandMessage(state *ReceivedMessageState, message *Message) error {
	message.ID = state.CurrentMessageState.MessageID
	message.From = state.CurrentMessageState.Contact.ID
	message.Alias = state.CurrentMessageState.Contact.Alias
	message.SigPubKey = state.CurrentMessageState.PublicKey
	message.Identicon = state.CurrentMessageState.Contact.Identicon
	message.WhisperTimestamp = state.CurrentMessageState.WhisperTimestamp

	message.PrepareContent()
	chat, err := m.matchMessage(message, state.AllChats)
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
	if !isPubKeyEqual(message.SigPubKey, &m.identity.PublicKey) {
		chat.UnviewedMessagesCount++
		message.OutgoingStatus = ""
	} else {
		// Our own message, mark as sent
		message.OutgoingStatus = OutgoingStatusSent
	}

	err = chat.UpdateFromMessage(message)
	if err != nil {
		return err
	}

	// Set chat active
	chat.Active = true
	// Set in the modified maps chat
	state.ModifiedChats[chat.ID] = true
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
		chat = OneToOneFromPublicKey(state.CurrentMessageState.PublicKey)
		// We don't want to show the chat to the user
		chat.Active = false
	}

	contact, ok := state.AllContacts[message.Id]
	if !ok {
		publicKeyBytes, err := hex.DecodeString(message.Id[2:])
		if err != nil {
			return err
		}
		publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
		if err != nil {
			return err
		}
		contact, err = buildContact(publicKey)
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
		contact.Photo = message.ProfileImage
		contact.LastUpdated = message.Clock
		state.ModifiedContacts[contact.ID] = true
		state.AllContacts[contact.ID] = contact
	}

	state.AllChats[chat.ID] = chat

	return nil
}

func (m *MessageHandler) HandleContactUpdate(state *ReceivedMessageState, message protobuf.ContactUpdate) error {
	logger := m.logger.With(zap.String("site", "HandleContactUpdate"))
	contact := state.CurrentMessageState.Contact
	chat, ok := state.AllChats[contact.ID]
	if !ok {
		chat = OneToOneFromPublicKey(state.CurrentMessageState.PublicKey)
		// We don't want to show the chat to the user
		chat.Active = false
	}

	logger.Info("Handling contact update")

	if contact.LastUpdated < message.Clock {
		logger.Info("Updating contact")
		if !contact.HasBeenAdded() {
			contact.SystemTags = append(contact.SystemTags, contactRequestReceived)
		}
		if contact.Name != message.EnsName {
			contact.Name = message.EnsName
			contact.ENSVerified = false
		}
		contact.Photo = message.ProfileImage
		contact.LastUpdated = message.Clock
		state.ModifiedContacts[contact.ID] = true
		state.AllContacts[contact.ID] = contact
	}

	if chat.LastClockValue < message.Clock {
		chat.LastClockValue = message.Clock
	}

	state.ModifiedChats[chat.ID] = true
	state.AllChats[chat.ID] = chat

	return nil
}

func (m *MessageHandler) HandlePairInstallation(state *ReceivedMessageState, message protobuf.PairInstallation) error {
	logger := m.logger.With(zap.String("site", "HandlePairInstallation"))
	if err := ValidateReceivedPairInstallation(&message); err != nil {
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

func (m *MessageHandler) HandleChatMessage(state *ReceivedMessageState) error {
	logger := m.logger.With(zap.String("site", "handleChatMessage"))
	if err := ValidateReceivedChatMessage(&state.CurrentMessageState.Message); err != nil {
		logger.Warn("failed to validate message", zap.Error(err))
		return err
	}
	receivedMessage := &Message{
		ID:               state.CurrentMessageState.MessageID,
		ChatMessage:      state.CurrentMessageState.Message,
		From:             state.CurrentMessageState.Contact.ID,
		Alias:            state.CurrentMessageState.Contact.Alias,
		SigPubKey:        state.CurrentMessageState.PublicKey,
		Identicon:        state.CurrentMessageState.Contact.Identicon,
		WhisperTimestamp: state.CurrentMessageState.WhisperTimestamp,
	}

	receivedMessage.PrepareContent()
	chat, err := m.matchMessage(receivedMessage, state.AllChats)
	if err != nil {
		return err
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
	if !isPubKeyEqual(receivedMessage.SigPubKey, &m.identity.PublicKey) {
		chat.UnviewedMessagesCount++
	} else {
		// Our own message, mark as sent
		receivedMessage.OutgoingStatus = OutgoingStatusSent
	}

	err = chat.UpdateFromMessage(receivedMessage)
	if err != nil {
		return err
	}

	// Set chat active
	chat.Active = true
	// Set in the modified maps chat
	state.ModifiedChats[chat.ID] = true
	state.AllChats[chat.ID] = chat

	// Add to response
	if receivedMessage != nil {
		state.Response.Messages = append(state.Response.Messages, receivedMessage)
	}

	return nil
}

func (m *MessageHandler) HandleRequestAddressForTransaction(messageState *ReceivedMessageState, command protobuf.RequestAddressForTransaction) error {
	err := ValidateReceivedRequestAddressForTransaction(&command)
	if err != nil {
		return err
	}
	message := &Message{
		ChatMessage: protobuf.ChatMessage{
			Clock:       command.Clock,
			Timestamp:   messageState.CurrentMessageState.WhisperTimestamp,
			Text:        "Request address for transaction",
			ChatId:      types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
			MessageType: protobuf.ChatMessage_ONE_TO_ONE,
			ContentType: protobuf.ChatMessage_TRANSACTION_COMMAND,
		},
		CommandParameters: &CommandParameters{
			ID:           messageState.CurrentMessageState.MessageID,
			Value:        command.Value,
			Contract:     command.Contract,
			CommandState: CommandStateRequestAddressForTransaction,
		},
	}
	return m.handleCommandMessage(messageState, message)
}

func (m *MessageHandler) HandleRequestTransaction(messageState *ReceivedMessageState, command protobuf.RequestTransaction) error {
	err := ValidateReceivedRequestTransaction(&command)
	if err != nil {
		return err
	}
	message := &Message{
		ChatMessage: protobuf.ChatMessage{
			Clock:       command.Clock,
			Timestamp:   messageState.CurrentMessageState.WhisperTimestamp,
			Text:        "Request transaction",
			ChatId:      types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
			MessageType: protobuf.ChatMessage_ONE_TO_ONE,
			ContentType: protobuf.ChatMessage_TRANSACTION_COMMAND,
		},
		CommandParameters: &CommandParameters{
			ID:           messageState.CurrentMessageState.MessageID,
			Value:        command.Value,
			Contract:     command.Contract,
			CommandState: CommandStateRequestTransaction,
			Address:      command.Address,
		},
	}
	return m.handleCommandMessage(messageState, message)
}

func (m *MessageHandler) HandleAcceptRequestAddressForTransaction(messageState *ReceivedMessageState, command protobuf.AcceptRequestAddressForTransaction) error {
	err := ValidateReceivedAcceptRequestAddressForTransaction(&command)
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

	if initialMessage.CommandParameters.CommandState != CommandStateRequestAddressForTransaction {
		return errors.New("Wrong state for command")
	}

	initialMessage.Clock = command.Clock
	initialMessage.Timestamp = messageState.CurrentMessageState.WhisperTimestamp
	initialMessage.Text = "Request address for transaction accepted"
	initialMessage.CommandParameters.Address = command.Address
	initialMessage.CommandParameters.CommandState = CommandStateRequestAddressForTransactionAccepted

	// Hide previous message
	previousMessage, err := m.persistence.MessageByCommandID(command.Id)
	if err != nil && err != errRecordNotFound {
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
	err := ValidateReceivedSendTransaction(&command)
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
	err := ValidateReceivedDeclineRequestAddressForTransaction(&command)
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

	if oldMessage.CommandParameters.CommandState != CommandStateRequestAddressForTransaction {
		return errors.New("Wrong state for command")
	}

	oldMessage.Clock = command.Clock
	oldMessage.Timestamp = messageState.CurrentMessageState.WhisperTimestamp
	oldMessage.Text = "Request address for transaction declined"
	oldMessage.CommandParameters.CommandState = CommandStateRequestAddressForTransactionDeclined

	// Hide previous message
	err = m.persistence.HideMessage(command.Id)
	if err != nil {
		return err
	}
	oldMessage.Replace = command.Id

	return m.handleCommandMessage(messageState, oldMessage)
}

func (m *MessageHandler) HandleDeclineRequestTransaction(messageState *ReceivedMessageState, command protobuf.DeclineRequestTransaction) error {
	err := ValidateReceivedDeclineRequestTransaction(&command)
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

	if oldMessage.CommandParameters.CommandState != CommandStateRequestTransaction {
		return errors.New("Wrong state for command")
	}

	oldMessage.Clock = command.Clock
	oldMessage.Timestamp = messageState.CurrentMessageState.WhisperTimestamp
	oldMessage.Text = "Transaction request declined"
	oldMessage.CommandParameters.CommandState = CommandStateRequestTransactionDeclined

	// Hide previous message
	err = m.persistence.HideMessage(command.Id)
	if err != nil {
		return err
	}
	oldMessage.Replace = command.Id

	return m.handleCommandMessage(messageState, oldMessage)
}

func (m *MessageHandler) matchMessage(message *Message, chats map[string]*Chat) (*Chat, error) {
	if message.SigPubKey == nil {
		m.logger.Error("public key can't be empty")
		return nil, errors.New("received a message with empty public key")
	}

	switch {
	case message.MessageType == protobuf.ChatMessage_PUBLIC_GROUP:
		// For public messages, all outgoing and incoming messages have the same chatID
		// equal to a public chat name.
		chatID := message.ChatId
		chat := chats[chatID]
		if chat == nil {
			return nil, errors.New("received a public message from non-existing chat")
		}
		return chat, nil
	case message.MessageType == protobuf.ChatMessage_ONE_TO_ONE && isPubKeyEqual(message.SigPubKey, &m.identity.PublicKey):
		// It's a private message coming from us so we rely on Message.ChatId
		// If chat does not exist, it should be created to support multidevice synchronization.
		chatID := message.ChatId
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

			newChat := CreateOneToOneChat(chatID[:8], pubKey)
			chat = &newChat
		}
		return chat, nil
	case message.MessageType == protobuf.ChatMessage_ONE_TO_ONE:
		// It's an incoming private message. ChatID is calculated from the signature.
		// If a chat does not exist, a new one is created and saved.
		chatID := types.EncodeHex(crypto.FromECDSAPub(message.SigPubKey))
		chat := chats[chatID]
		if chat == nil {
			// TODO: this should be a three-word name used in the mobile client
			newChat := CreateOneToOneChat(chatID[:8], message.SigPubKey)
			chat = &newChat
		}
		return chat, nil
	case message.MessageType == protobuf.ChatMessage_PRIVATE_GROUP:
		// In the case of a group message, ChatID is the same for all messages belonging to a group.
		// It needs to be verified if the signature public key belongs to the chat.
		chatID := message.ChatId
		chat := chats[chatID]
		if chat == nil {
			return nil, errors.New("received group chat message for non-existing chat")
		}

		theirKeyHex := types.EncodeHex(crypto.FromECDSAPub(message.SigPubKey))
		myKeyHex := types.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey))
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
	if err != nil && err != errRecordNotFound {
		return false, err
	}
	if existingMessage != nil {
		return true, nil
	}
	return false, nil
}
