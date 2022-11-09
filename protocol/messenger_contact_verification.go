package protocol

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/verification"
)

const minContactVerificationMessageLen = 1
const maxContactVerificationMessageLen = 280

func (m *Messenger) SendContactVerificationRequest(ctx context.Context, contactID string, challenge string) (*MessengerResponse, error) {
	if len(challenge) < minContactVerificationMessageLen || len(challenge) > maxContactVerificationMessageLen {
		return nil, errors.New("invalid verification request challenge length")
	}

	contact, ok := m.allContacts.Load(contactID)
	if !ok || !contact.Added || !contact.HasAddedUs {
		return nil, errors.New("must be a mutual contact")
	}

	verifRequest := &verification.Request{
		From:          common.PubkeyToHex(&m.identity.PublicKey),
		To:            contact.ID,
		Challenge:     challenge,
		RequestStatus: verification.RequestStatusPENDING,
		RepliedAt:     0,
	}

	chat, ok := m.allChats.Load(contactID)
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return nil, err
		}
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats.Store(chat.ID, chat)
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	request := &protobuf.RequestContactVerification{
		Clock:     clock,
		Challenge: challenge,
	}

	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_REQUEST_CONTACT_VERIFICATION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	contact.VerificationStatus = VerificationStatusVERIFYING
	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return nil, err
	}

	// We sync the contact with the other devices
	err = m.syncContact(context.Background(), contact)
	if err != nil {
		return nil, err
	}

	m.allContacts.Store(contact.ID, contact)

	verifRequest.RequestedAt = clock
	verifRequest.ID = rawMessage.ID

	err = m.verificationDatabase.SaveVerificationRequest(verifRequest)
	if err != nil {
		return nil, err
	}

	err = m.SyncVerificationRequest(context.Background(), verifRequest)
	if err != nil {
		return nil, err
	}

	chatMessage, err := m.createLocalContactVerificationMessage(request.Challenge, chat, rawMessage.ID, common.ContactVerificationStatePending)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{chatMessage})
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{
		VerificationRequests: []*verification.Request{verifRequest},
	}

	response.AddMessage(chatMessage)

	m.prepareMessages(response.messages)

	return response, nil
}

func (m *Messenger) GetVerificationRequestSentTo(ctx context.Context, contactID string) (*verification.Request, error) {
	_, ok := m.allContacts.Load(contactID)
	if !ok {
		return nil, errors.New("contact not found")
	}

	return m.verificationDatabase.GetVerificationRequestSentTo(contactID)
}

func (m *Messenger) GetReceivedVerificationRequests(ctx context.Context) ([]*verification.Request, error) {
	myPubKey := hexutil.Encode(crypto.FromECDSAPub(&m.identity.PublicKey))
	return m.verificationDatabase.GetReceivedVerificationRequests(myPubKey)
}

func (m *Messenger) CancelVerificationRequest(ctx context.Context, contactID string) error {
	contact, ok := m.allContacts.Load(contactID)
	if !ok || !contact.Added || !contact.HasAddedUs {
		return errors.New("must be a mutual contact")
	}

	verifRequest, err := m.verificationDatabase.GetVerificationRequestSentTo(contactID)
	if err != nil {
		return err
	}

	if verifRequest == nil {
		return errors.New("no contact verification found")
	}

	if verifRequest.RequestStatus != verification.RequestStatusPENDING {
		return errors.New("can't cancel a request already verified")
	}

	verifRequest.RequestStatus = verification.RequestStatusCANCELED
	err = m.verificationDatabase.SaveVerificationRequest(verifRequest)
	if err != nil {
		return err
	}
	contact.VerificationStatus = VerificationStatusUNVERIFIED
	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()

	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	// We sync the contact with the other devices
	err = m.syncContact(context.Background(), contact)
	if err != nil {
		return err
	}

	err = m.SyncVerificationRequest(context.Background(), verifRequest)
	if err != nil {
		return err
	}

	m.allContacts.Store(contact.ID, contact)

	return nil
}

func (m *Messenger) AcceptContactVerificationRequest(ctx context.Context, id string, response string) (*MessengerResponse, error) {

	verifRequest, err := m.verificationDatabase.GetVerificationRequest(id)
	if err != nil {
		return nil, err
	}

	if verifRequest == nil {
		m.logger.Error("could not find verification request with id", zap.String("id", id))
		return nil, verification.ErrVerificationRequestNotFound
	}

	contactID := verifRequest.From

	contact, ok := m.allContacts.Load(contactID)
	if !ok || !contact.Added || !contact.HasAddedUs {
		return nil, errors.New("must be a mutual contact")
	}

	chat, ok := m.allChats.Load(contactID)
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return nil, err
		}
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats.Store(chat.ID, chat)
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	err = m.verificationDatabase.AcceptContactVerificationRequest(id, response)
	if err != nil {
		return nil, err
	}

	verifRequest, err = m.verificationDatabase.GetVerificationRequest(id)
	if err != nil {
		return nil, err
	}

	err = m.SyncVerificationRequest(context.Background(), verifRequest)
	if err != nil {
		return nil, err
	}

	request := &protobuf.AcceptContactVerification{
		Clock:    clock,
		Id:       verifRequest.ID,
		Response: response,
	}

	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	rawMessage, err := m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_ACCEPT_CONTACT_VERIFICATION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	// Pull one from the db if there
	notification, err := m.persistence.GetActivityCenterNotificationByID(types.FromHex(id))
	if err != nil {
		return nil, err
	}
	resp := &MessengerResponse{
		VerificationRequests: []*verification.Request{verifRequest},
	}

	chatMessage, err := m.createLocalContactVerificationMessage(response, chat, rawMessage.ID, common.ContactVerificationStateAccepted)
	if err != nil {
		return nil, err
	}

	err = m.persistence.SaveMessages([]*common.Message{chatMessage})
	if err != nil {
		return nil, err
	}

	resp.AddMessage(chatMessage)

	if notification != nil {
		// TODO: Should we update only the message or only the notification or both?
		err := m.persistence.UpdateActivityCenterNotificationContactVerificationStatus(notification.ID, verification.RequestStatusACCEPTED)
		if err != nil {
			return nil, err
		}

		notification.ContactVerificationStatus = verification.RequestStatusACCEPTED
		message := notification.Message
		message.ContactVerificationState = common.ContactVerificationStateAccepted
		err = m.persistence.UpdateActivityCenterNotificationMessage(notification.ID, message)
		if err != nil {
			return nil, err
		}
		resp.AddActivityCenterNotification(notification)

	}

	return resp, nil
}

func (m *Messenger) VerifiedTrusted(ctx context.Context, contactID string) error {
	contact, ok := m.allContacts.Load(contactID)
	if !ok || !contact.Added || !contact.HasAddedUs {
		return errors.New("must be a mutual contact")
	}

	err := m.verificationDatabase.SetTrustStatus(contactID, verification.TrustStatusTRUSTED, m.getTimesource().GetCurrentTime())
	if err != nil {
		return err
	}

	err = m.SyncTrustedUser(context.Background(), contactID, verification.TrustStatusTRUSTED)
	if err != nil {
		return err
	}

	contact.VerificationStatus = VerificationStatusVERIFIED
	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()
	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	chat, ok := m.allChats.Load(contactID)
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return err
		}
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	request := &protobuf.ContactVerificationTrusted{
		Clock: clock,
	}

	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_CONTACT_VERIFICATION_TRUSTED,
		ResendAutomatically: true,
	})

	if err != nil {
		return err
	}

	verifRequest, err := m.verificationDatabase.GetVerificationRequestSentTo(contactID)
	if err != nil {
		return err
	}

	if verifRequest == nil {
		return errors.New("no contact verification found")
	}

	verifRequest.RequestStatus = verification.RequestStatusTRUSTED
	verifRequest.RepliedAt = clock
	err = m.verificationDatabase.SaveVerificationRequest(verifRequest)
	if err != nil {
		return err
	}

	err = m.SyncVerificationRequest(context.Background(), verifRequest)
	if err != nil {
		return err
	}

	// We sync the contact with the other devices
	err = m.syncContact(context.Background(), contact)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) VerifiedUntrustworthy(ctx context.Context, contactID string) error {
	contact, ok := m.allContacts.Load(contactID)
	if !ok || !contact.Added || !contact.HasAddedUs {
		return errors.New("must be a mutual contact")
	}

	err := m.verificationDatabase.SetTrustStatus(contactID, verification.TrustStatusUNTRUSTWORTHY, m.getTimesource().GetCurrentTime())
	if err != nil {
		return err
	}

	err = m.SyncTrustedUser(context.Background(), contactID, verification.TrustStatusUNTRUSTWORTHY)
	if err != nil {
		return err
	}

	contact.VerificationStatus = VerificationStatusVERIFIED
	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()
	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}

	// We sync the contact with the other devices
	err = m.syncContact(context.Background(), contact)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) DeclineContactVerificationRequest(ctx context.Context, id string) (*MessengerResponse, error) {
	verifRequest, err := m.verificationDatabase.GetVerificationRequest(id)
	if err != nil {
		return nil, err
	}

	if verifRequest == nil {
		m.logger.Error("could not find verification request with id", zap.String("id", id))
		return nil, verification.ErrVerificationRequestNotFound
	}

	contact, ok := m.allContacts.Load(verifRequest.From)
	if !ok || !contact.Added || !contact.HasAddedUs {
		return nil, errors.New("must be a mutual contact")
	}

	if verifRequest == nil {
		return nil, errors.New("no contact verification found")
	}

	chat, ok := m.allChats.Load(verifRequest.From)
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return nil, err
		}
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats.Store(chat.ID, chat)
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	verifRequest.RequestStatus = verification.RequestStatusDECLINED
	verifRequest.RepliedAt = clock
	err = m.verificationDatabase.SaveVerificationRequest(verifRequest)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}

	response.AddVerificationRequest(verifRequest)

	err = m.SyncVerificationRequest(context.Background(), verifRequest)
	if err != nil {
		return nil, err
	}

	request := &protobuf.DeclineContactVerification{
		Id:    id,
		Clock: clock,
	}

	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_DECLINE_CONTACT_VERIFICATION,
		ResendAutomatically: true,
	})

	if err != nil {
		return nil, err
	}

	err = m.verificationDatabase.DeclineContactVerificationRequest(id)
	if err != nil {
		return nil, err
	}

	notification, err := m.persistence.GetActivityCenterNotificationByID(types.FromHex(id))
	if err != nil {
		return nil, err
	}

	if notification != nil {
		// TODO: Should we update only the message or only the notification or both?
		err := m.persistence.UpdateActivityCenterNotificationContactVerificationStatus(notification.ID, verification.RequestStatusDECLINED)
		if err != nil {
			return nil, err
		}

		notification.ContactVerificationStatus = verification.RequestStatusDECLINED
		message := notification.Message
		message.ContactVerificationState = common.ContactVerificationStateDeclined
		err = m.persistence.UpdateActivityCenterNotificationMessage(notification.ID, message)
		if err != nil {
			return nil, err
		}
		response.AddActivityCenterNotification(notification)
		response.AddMessage(message)
	}

	return response, nil
}

func (m *Messenger) MarkAsTrusted(ctx context.Context, contactID string) error {
	err := m.verificationDatabase.SetTrustStatus(contactID, verification.TrustStatusTRUSTED, m.getTimesource().GetCurrentTime())
	if err != nil {
		return err
	}

	return m.SyncTrustedUser(ctx, contactID, verification.TrustStatusTRUSTED)
}

func (m *Messenger) MarkAsUntrustworthy(ctx context.Context, contactID string) error {
	err := m.verificationDatabase.SetTrustStatus(contactID, verification.TrustStatusUNTRUSTWORTHY, m.getTimesource().GetCurrentTime())
	if err != nil {
		return err
	}

	return m.SyncTrustedUser(ctx, contactID, verification.TrustStatusUNTRUSTWORTHY)
}

func (m *Messenger) RemoveTrustStatus(ctx context.Context, contactID string) error {
	err := m.verificationDatabase.SetTrustStatus(contactID, verification.TrustStatusUNKNOWN, m.getTimesource().GetCurrentTime())
	if err != nil {
		return err
	}

	return m.SyncTrustedUser(ctx, contactID, verification.TrustStatusUNKNOWN)
}

func (m *Messenger) GetTrustStatus(contactID string) (verification.TrustStatus, error) {
	return m.verificationDatabase.GetTrustStatus(contactID)
}

func ValidateContactVerificationRequest(request protobuf.RequestContactVerification) error {
	challengeLen := len(strings.TrimSpace(request.Challenge))
	if challengeLen < minContactVerificationMessageLen || challengeLen > maxContactVerificationMessageLen {
		return errors.New("invalid verification request challenge length")
	}

	return nil
}

func (m *Messenger) HandleRequestContactVerification(state *ReceivedMessageState, request protobuf.RequestContactVerification) error {
	if err := ValidateContactVerificationRequest(request); err != nil {
		m.logger.Debug("Invalid verification request", zap.Error(err))
		return err
	}

	id := state.CurrentMessageState.MessageID

	if common.IsPubKeyEqual(state.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
		return nil // Is ours, do nothing
	}

	myPubKey := hexutil.Encode(crypto.FromECDSAPub(&m.identity.PublicKey))
	contactID := hexutil.Encode(crypto.FromECDSAPub(state.CurrentMessageState.PublicKey))

	contact := state.CurrentMessageState.Contact
	if !contact.Added || !contact.HasAddedUs {
		m.logger.Debug("Received a verification request for a non added mutual contact", zap.String("contactID", contactID))
		return errors.New("must be a mutual contact")
	}

	persistedVR, err := m.verificationDatabase.GetVerificationRequest(id)
	if err != nil {
		m.logger.Debug("Error obtaining verification request", zap.Error(err))
		return err
	}

	if persistedVR != nil && persistedVR.RequestedAt > request.Clock {
		return nil // older message, ignore it
	}

	if persistedVR == nil {
		// This is a new verification request, and we have not received its acceptance/decline before
		persistedVR = &verification.Request{}
		persistedVR.ID = id
		persistedVR.From = contactID
		persistedVR.To = myPubKey
		persistedVR.RequestStatus = verification.RequestStatusPENDING
	}

	if persistedVR.From != contactID {
		return errors.New("mismatch contactID and ID")
	}

	persistedVR.Challenge = request.Challenge
	persistedVR.RequestedAt = request.Clock

	err = m.verificationDatabase.SaveVerificationRequest(persistedVR)
	if err != nil {
		m.logger.Debug("Error storing verification request", zap.Error(err))
		return err
	}
	m.logger.Info("SAVED", zap.String("id", persistedVR.ID))

	err = m.SyncVerificationRequest(context.Background(), persistedVR)
	if err != nil {
		return err
	}

	chat, ok := m.allChats.Load(contactID)
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return err
		}
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats.Store(chat.ID, chat)

	chatMessage, err := m.createContactVerificationMessage(request.Challenge, chat, state, common.ContactVerificationStatePending)
	if err != nil {
		return err
	}

	state.Response.AddMessage(chatMessage)

	state.AllVerificationRequests = append(state.AllVerificationRequests, persistedVR)

	// TODO: update activity center notification, this only creates a new one
	return m.createContactVerificationNotification(contact, state, persistedVR, chatMessage)
}

func ValidateAcceptContactVerification(request protobuf.AcceptContactVerification) error {
	responseLen := len(strings.TrimSpace(request.Response))
	if responseLen < minContactVerificationMessageLen || responseLen > maxContactVerificationMessageLen {
		return errors.New("invalid verification request response length")
	}

	return nil
}

func (m *Messenger) HandleAcceptContactVerification(state *ReceivedMessageState, request protobuf.AcceptContactVerification) error {
	if err := ValidateAcceptContactVerification(request); err != nil {
		m.logger.Debug("Invalid AcceptContactVerification", zap.Error(err))
		return err
	}

	if common.IsPubKeyEqual(state.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
		return nil // Is ours, do nothing
	}

	myPubKey := hexutil.Encode(crypto.FromECDSAPub(&m.identity.PublicKey))
	contactID := hexutil.Encode(crypto.FromECDSAPub(state.CurrentMessageState.PublicKey))

	contact := state.CurrentMessageState.Contact
	if !contact.Added || !contact.HasAddedUs {
		m.logger.Debug("Received a verification response for a non mutual contact", zap.String("contactID", contactID))
		return errors.New("must be a mutual contact")
	}

	persistedVR, err := m.verificationDatabase.GetVerificationRequest(request.Id)
	if err != nil {
		m.logger.Debug("Error obtaining verification request", zap.Error(err))
		return err
	}
	m.logger.Info("PAST 1")

	if persistedVR != nil && persistedVR.RepliedAt > request.Clock {
		return nil // older message, ignore it
	}

	if persistedVR.RequestStatus == verification.RequestStatusCANCELED {
		return nil // Do nothing, We have already cancelled the verification request
	}
	m.logger.Info("PAST 2")

	if persistedVR == nil {
		m.logger.Info("PAST 3")
		// This is a response for which we have not received its request before
		persistedVR = &verification.Request{}
		persistedVR.ID = request.Id
		persistedVR.From = contactID
		persistedVR.To = myPubKey
	}

	persistedVR.RequestStatus = verification.RequestStatusACCEPTED
	persistedVR.Response = request.Response
	persistedVR.RepliedAt = request.Clock

	err = m.verificationDatabase.SaveVerificationRequest(persistedVR)
	if err != nil {
		m.logger.Debug("Error storing verification request", zap.Error(err))
		return err
	}
	m.logger.Info("PAST 4")

	err = m.SyncVerificationRequest(context.Background(), persistedVR)
	if err != nil {
		return err
	}

	m.logger.Info("PAST 5")

	chat, ok := m.allChats.Load(contactID)
	if !ok {
		publicKey, err := contact.PublicKey()
		if err != nil {
			return err
		}
		chat = OneToOneFromPublicKey(publicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats.Store(chat.ID, chat)

	chatMessage, err := m.createContactVerificationMessage(request.Response, chat, state, common.ContactVerificationStateAccepted)
	if err != nil {
		return err
	}

	state.Response.AddMessage(chatMessage)

	err = m.createContactVerificationNotification(contact, state, persistedVR, chatMessage)
	if err != nil {
		return err
	}

	state.AllVerificationRequests = append(state.AllVerificationRequests, persistedVR)

	// TODO: create or update activity center notification

	return nil
}

func (m *Messenger) HandleDeclineContactVerification(state *ReceivedMessageState, request protobuf.DeclineContactVerification) error {
	if common.IsPubKeyEqual(state.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
		return nil // Is ours, do nothing
	}

	myPubKey := hexutil.Encode(crypto.FromECDSAPub(&m.identity.PublicKey))
	contactID := hexutil.Encode(crypto.FromECDSAPub(state.CurrentMessageState.PublicKey))

	contact := state.CurrentMessageState.Contact
	if !contact.Added || !contact.HasAddedUs {
		m.logger.Debug("Received a verification decline for a non mutual contact", zap.String("contactID", contactID))
		return errors.New("must be a mutual contact")
	}

	persistedVR, err := m.verificationDatabase.GetVerificationRequest(request.Id)
	if err != nil {
		m.logger.Debug("Error obtaining verification request", zap.Error(err))
		return err
	}

	if persistedVR != nil && persistedVR.RepliedAt > request.Clock {
		return nil // older message, ignore it
	}

	if persistedVR.RequestStatus == verification.RequestStatusCANCELED {
		return nil // Do nothing, We have already cancelled the verification request
	}

	if persistedVR == nil {
		// This is a response for which we have not received its request before
		persistedVR = &verification.Request{}
		persistedVR.From = contactID
		persistedVR.To = myPubKey
	}

	persistedVR.RequestStatus = verification.RequestStatusDECLINED
	persistedVR.RepliedAt = request.Clock

	err = m.verificationDatabase.SaveVerificationRequest(persistedVR)
	if err != nil {
		m.logger.Debug("Error storing verification request", zap.Error(err))
		return err
	}

	err = m.SyncVerificationRequest(context.Background(), persistedVR)
	if err != nil {
		return err
	}

	state.AllVerificationRequests = append(state.AllVerificationRequests, persistedVR)

	msg, err := m.persistence.MessageByID(request.Id)
	if err != nil {
		return err
	}

	if msg != nil {
		msg.ContactVerificationState = common.ContactVerificationStateDeclined
		state.Response.AddMessage(msg)
	}

	return m.createContactVerificationNotification(contact, state, persistedVR, msg)
}

func (m *Messenger) HandleContactVerificationTrusted(state *ReceivedMessageState, request protobuf.ContactVerificationTrusted) error {
	if common.IsPubKeyEqual(state.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
		return nil // Is ours, do nothing
	}

	if len(request.Id) == 0 {
		return errors.New("invalid ContactVerificationTrusted")
	}

	myPubKey := hexutil.Encode(crypto.FromECDSAPub(&m.identity.PublicKey))
	contactID := hexutil.Encode(crypto.FromECDSAPub(state.CurrentMessageState.PublicKey))

	contact := state.CurrentMessageState.Contact
	if !contact.Added || !contact.HasAddedUs {
		m.logger.Debug("Received a verification trusted for a non mutual contact", zap.String("contactID", contactID))
		return errors.New("must be a mutual contact")
	}

	err := m.verificationDatabase.SetTrustStatus(contactID, verification.TrustStatusTRUSTED, m.getTimesource().GetCurrentTime())
	if err != nil {
		return err
	}

	err = m.SyncTrustedUser(context.Background(), contactID, verification.TrustStatusTRUSTED)
	if err != nil {
		return err
	}

	persistedVR, err := m.verificationDatabase.GetVerificationRequest(request.Id)
	if err != nil {
		m.logger.Debug("Error obtaining verification request", zap.Error(err))
		return err
	}

	if persistedVR != nil && persistedVR.RepliedAt > request.Clock {
		return nil // older message, ignore it
	}

	if persistedVR.RequestStatus == verification.RequestStatusCANCELED {
		return nil // Do nothing, We have already cancelled the verification request
	}

	if persistedVR == nil {
		// This is a response for which we have not received its request before
		persistedVR = &verification.Request{}
		persistedVR.From = contactID
		persistedVR.To = myPubKey
	}

	persistedVR.RequestStatus = verification.RequestStatusTRUSTED

	err = m.verificationDatabase.SaveVerificationRequest(persistedVR)
	if err != nil {
		m.logger.Debug("Error storing verification request", zap.Error(err))
		return err
	}

	err = m.SyncVerificationRequest(context.Background(), persistedVR)
	if err != nil {
		return err
	}

	state.AllVerificationRequests = append(state.AllVerificationRequests, persistedVR)

	contact.VerificationStatus = VerificationStatusVERIFIED
	contact.LastUpdatedLocally = m.getTimesource().GetCurrentTime()
	err = m.persistence.SaveContact(contact, nil)
	if err != nil {
		return err
	}
	state.ModifiedContacts.Store(contact.ID, true)
	state.AllContacts.Store(contact.ID, contact)

	// We sync the contact with the other devices
	err = m.syncContact(context.Background(), contact)
	if err != nil {
		return err
	}

	// TODO: create or update activity center notification

	return nil
}

func (m *Messenger) GetLatestVerificationRequestFrom(contactID string) (*verification.Request, error) {
	return m.verificationDatabase.GetLatestVerificationRequestFrom(contactID)
}

func (m *Messenger) createContactVerificationNotification(contact *Contact, messageState *ReceivedMessageState, vr *verification.Request, chatMessage *common.Message) error {
	notification := &ActivityCenterNotification{
		ID:                        types.FromHex(vr.ID),
		Name:                      contact.CanonicalName(),
		Type:                      ActivityCenterNotificationTypeContactVerification,
		Author:                    messageState.CurrentMessageState.Contact.ID,
		Message:                   chatMessage,
		Timestamp:                 messageState.CurrentMessageState.WhisperTimestamp,
		ChatID:                    contact.ID,
		ContactVerificationStatus: vr.RequestStatus,
	}

	return m.addActivityCenterNotification(messageState, notification)
}

func (m *Messenger) createContactVerificationMessage(challenge string, chat *Chat, state *ReceivedMessageState, verificationStatus common.ContactVerificationState) (*common.Message, error) {

	chatMessage := &common.Message{}
	chatMessage.ID = state.CurrentMessageState.MessageID
	chatMessage.From = state.CurrentMessageState.Contact.ID
	chatMessage.Alias = state.CurrentMessageState.Contact.Alias
	chatMessage.SigPubKey = state.CurrentMessageState.PublicKey
	chatMessage.Identicon = state.CurrentMessageState.Contact.Identicon
	chatMessage.WhisperTimestamp = state.CurrentMessageState.WhisperTimestamp

	chatMessage.ChatId = chat.ID
	chatMessage.Text = challenge
	chatMessage.ContentType = protobuf.ChatMessage_IDENTITY_VERIFICATION
	chatMessage.ContactVerificationState = verificationStatus

	err := chatMessage.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}
	return chatMessage, nil
}

func (m *Messenger) createLocalContactVerificationMessage(challenge string, chat *Chat, id string, status common.ContactVerificationState) (*common.Message, error) {

	chatMessage := &common.Message{}
	chatMessage.ID = id
	err := extendMessageFromChat(chatMessage, chat, &m.identity.PublicKey, m.getTimesource())
	if err != nil {
		return nil, err
	}

	chatMessage.ChatId = chat.ID
	chatMessage.Text = challenge
	chatMessage.ContentType = protobuf.ChatMessage_IDENTITY_VERIFICATION
	chatMessage.ContactVerificationState = status
	err = extendMessageFromChat(chatMessage, chat, &m.identity.PublicKey, m.getTimesource())
	if err != nil {
		return nil, err
	}

	err = chatMessage.PrepareContent(common.PubkeyToHex(&m.identity.PublicKey))
	if err != nil {
		return nil, err
	}
	return chatMessage, nil
}
