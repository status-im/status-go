package protocol

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/verification"
)

const minContactVerificationMessageLen = 1
const maxContactVerificationMessageLen = 280

func (m *Messenger) SendContactVerificationRequest(ctx context.Context, contactID string, challenge string) error {
	if len(challenge) < minContactVerificationMessageLen || len(challenge) > maxContactVerificationMessageLen {
		return errors.New("invalid verification request challenge length")
	}

	contact, ok := m.allContacts.Load(contactID)
	if !ok || !contact.Added || !contact.HasAddedUs {
		return errors.New("must be a mutual contact")
	}

	verifRequest, err := m.verificationDatabase.GetVerificationRequestFrom(contactID)
	if err != nil {
		return err
	}

	if verifRequest != nil && verifRequest.RequestStatus == verification.RequestStatusACCEPTED {
		return errors.New("verification request already accepted")
	}

	if verifRequest != nil && verifRequest.RequestStatus == verification.RequestStatusPENDING {
		return errors.New("verification request already sent")
	}

	if verifRequest == nil {
		verifRequest = &verification.Request{
			From:          common.PubkeyToHex(&m.identity.PublicKey),
			To:            contact.ID,
			Challenge:     challenge,
			RequestStatus: verification.RequestStatusPENDING,
			RepliedAt:     0,
		}
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
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	request := &protobuf.RequestContactVerification{
		Clock:     clock,
		Challenge: challenge,
	}

	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_REQUEST_CONTACT_VERIFICATION,
		ResendAutomatically: true,
	})

	if err != nil {
		return err
	}

	contact.VerificationStatus = VerificationStatusVERIFYING
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

	m.allContacts.Store(contact.ID, contact)

	verifRequest.RequestedAt = clock

	err = m.verificationDatabase.SaveVerificationRequest(verifRequest)
	if err != nil {
		return err
	}

	return m.SyncVerificationRequest(context.Background(), verifRequest)
}

func (m *Messenger) GetVerificationRequestSentTo(ctx context.Context, contactID string) (*verification.Request, error) {
	_, ok := m.allContacts.Load(contactID)
	if !ok {
		return nil, errors.New("contact not found")
	}

	return m.verificationDatabase.GetVerificationRequestSentTo(contactID)
}

func (m *Messenger) GetVerificationRequestFrom(ctx context.Context, contactID string) (*verification.Request, error) {
	_, ok := m.allContacts.Load(contactID)
	if !ok {
		return nil, errors.New("contact not found")
	}

	return m.verificationDatabase.GetVerificationRequestFrom(contactID)
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

func (m *Messenger) AcceptContactVerificationRequest(ctx context.Context, contactID string, response string) error {
	contact, ok := m.allContacts.Load(contactID)
	if !ok || !contact.Added || !contact.HasAddedUs {
		return errors.New("must be a mutual contact")
	}

	verifRequest, err := m.verificationDatabase.GetVerificationRequestFrom(contactID)
	if err != nil {
		return err
	}

	if verifRequest == nil {
		return errors.New("no contact verification found")
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
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	verifRequest.RequestStatus = verification.RequestStatusACCEPTED
	verifRequest.RepliedAt = clock
	err = m.verificationDatabase.SaveVerificationRequest(verifRequest)
	if err != nil {
		return err
	}

	err = m.SyncVerificationRequest(context.Background(), verifRequest)
	if err != nil {
		return err
	}

	request := &protobuf.AcceptContactVerification{
		Clock:    clock,
		Response: response,
	}

	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_ACCEPT_CONTACT_VERIFICATION,
		ResendAutomatically: true,
	})

	if err != nil {
		return err
	}

	return m.verificationDatabase.AcceptContactVerificationRequest(contactID, response)
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

func (m *Messenger) DeclineContactVerificationRequest(ctx context.Context, contactID string) error {
	contact, ok := m.allContacts.Load(contactID)
	if !ok || !contact.Added || !contact.HasAddedUs {
		return errors.New("must be a mutual contact")
	}

	verifRequest, err := m.verificationDatabase.GetVerificationRequestFrom(contactID)
	if err != nil {
		return err
	}

	if verifRequest == nil {
		return errors.New("no contact verification found")
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
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	verifRequest.RequestStatus = verification.RequestStatusDECLINED
	verifRequest.RepliedAt = clock
	err = m.verificationDatabase.SaveVerificationRequest(verifRequest)
	if err != nil {
		return err
	}

	err = m.SyncVerificationRequest(context.Background(), verifRequest)
	if err != nil {
		return err
	}

	request := &protobuf.DeclineContactVerification{
		Clock: clock,
	}

	encodedMessage, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(ctx, common.RawMessage{
		LocalChatID:         chat.ID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_DECLINE_CONTACT_VERIFICATION,
		ResendAutomatically: true,
	})

	if err != nil {
		return err
	}

	return m.verificationDatabase.DeclineContactVerificationRequest(contactID)
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

	persistedVR, err := m.verificationDatabase.GetVerificationRequestFrom(contactID)
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
		persistedVR.From = contactID
		persistedVR.To = myPubKey
		persistedVR.RequestStatus = verification.RequestStatusPENDING
	}

	persistedVR.Challenge = request.Challenge
	persistedVR.RequestedAt = request.Clock

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

	// TODO: create or update activity center notification

	return nil
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

	persistedVR, err := m.verificationDatabase.GetVerificationRequestSentTo(contactID)
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

	persistedVR.RequestStatus = verification.RequestStatusACCEPTED
	persistedVR.Response = request.Response
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

	persistedVR, err := m.verificationDatabase.GetVerificationRequestSentTo(contactID)
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

	// TODO: create or update activity center notification

	return nil
}

func (m *Messenger) HandleContactVerificationTrusted(state *ReceivedMessageState, request protobuf.ContactVerificationTrusted) error {
	if common.IsPubKeyEqual(state.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
		return nil // Is ours, do nothing
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

	persistedVR, err := m.verificationDatabase.GetVerificationRequestFrom(contactID)
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
