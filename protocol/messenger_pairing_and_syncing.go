package protocol

import (
	"context"
	"errors"
	"time"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/images"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/encryption/multidevice"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
)

type InstallationIDProvider interface {
	GetInstallationID() string
	Validate() error
}

func (m *Messenger) EnableInstallationAndSync(request *requests.EnableInstallationAndSync) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	installation, err := m.EnableInstallation(request.InstallationID)
	if err != nil {
		return nil, err
	}

	response := &MessengerResponse{}
	response.AddInstallation(installation)

	pairResponse, err := m.SendPairInstallation(context.Background(), request.InstallationID, nil)
	if err != nil {
		return nil, err
	}

	if err = m.SyncDevices(context.Background(), "", "", nil); err != nil {
		return nil, err
	}

	if err = m.deleteNotification(pairResponse, request.InstallationID); err != nil {
		return nil, err
	}

	if err = pairResponse.Merge(response); err != nil {
		return nil, err
	}

	return pairResponse, nil
}

func (m *Messenger) EnableInstallationAndPair(request InstallationIDProvider) (*MessengerResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	myIdentity := crypto.CompressPubkey(&m.identity.PublicKey)
	timestamp := time.Now().UnixNano()
	installationID := request.GetInstallationID()

	installation := &multidevice.Installation{
		ID:        installationID,
		Enabled:   true,
		Version:   2,
		Timestamp: timestamp,
	}

	_, err := m.encryptor.AddInstallation(myIdentity, timestamp, installation, true)
	if err != nil {
		return nil, err
	}
	i, ok := m.allInstallations.Load(installationID)
	if !ok {
		i = installation
	} else {
		i.Enabled = true
	}
	m.allInstallations.Store(installationID, i)
	response, err := m.SendPairInstallation(context.Background(), request.GetInstallationID(), nil)
	if err != nil {
		return nil, err
	}

	notification := &ActivityCenterNotification{
		ID:             types.FromHex(installationID),
		Type:           ActivityCenterNotificationTypeNewInstallationCreated,
		InstallationID: m.installationID, // Put our own installation ID, as we're the initiator of the pairing
		Timestamp:      m.getTimesource().GetCurrentTime(),
		Read:           false,
		Deleted:        false,
		UpdatedAt:      m.GetCurrentTimeInMillis(),
	}

	err = m.addActivityCenterNotification(response, notification, nil)
	if err != nil {
		return nil, err
	}
	return response, err
}

// SendPairInstallation sends a pair installation message
func (m *Messenger) SendPairInstallation(ctx context.Context, targetInstallationID string, rawMessageHandler RawMessageHandler) (*MessengerResponse, error) {
	var err error
	var response MessengerResponse

	installation, ok := m.allInstallations.Load(m.installationID)
	if !ok {
		return nil, errors.New("no installation found")
	}

	if installation.InstallationMetadata == nil {
		return nil, errors.New("no installation metadata")
	}

	clock, chat := m.getLastClockWithRelatedChat()

	pairMessage := &protobuf.SyncPairInstallation{
		Clock:                clock,
		Name:                 installation.InstallationMetadata.Name,
		InstallationId:       installation.ID,
		DeviceType:           installation.InstallationMetadata.DeviceType,
		Version:              installation.Version,
		TargetInstallationId: targetInstallationID,
	}
	encodedMessage, err := proto.Marshal(pairMessage)
	if err != nil {
		return nil, err
	}

	if rawMessageHandler == nil {
		rawMessageHandler = m.dispatchPairInstallationMessage
	}
	_, err = rawMessageHandler(ctx, common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_SYNC_PAIR_INSTALLATION,
		ResendType:  common.ResendTypeDataSync,
	})
	if err != nil {
		return nil, err
	}

	response.AddChat(chat)

	chat.LastClockValue = clock
	err = m.saveChat(chat)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// SyncDevices sends all public chats and contacts to paired devices
// TODO remove use of photoPath in contacts
func (m *Messenger) SyncDevices(ctx context.Context, ensName, photoPath string, rawMessageHandler RawMessageHandler) (err error) {
	if rawMessageHandler == nil {
		rawMessageHandler = m.dispatchMessage
	}

	myID := contactIDFromPublicKey(&m.identity.PublicKey)

	displayName, err := m.settings.DisplayName()
	if err != nil {
		return err
	}

	if _, err = m.sendContactUpdate(ctx, myID, displayName, ensName, photoPath, m.account.GetCustomizationColor(), rawMessageHandler); err != nil {
		return err
	}

	m.allChats.Range(func(chatID string, chat *Chat) bool {
		if !chat.shouldBeSynced() {
			return true

		}
		err = m.syncChat(ctx, chat, rawMessageHandler)
		return err == nil
	})
	if err != nil {
		return err
	}

	m.allContacts.Range(func(contactID string, contact *Contact) bool {
		if contact.ID == myID {
			return true
		}
		if contact.LocalNickname != "" || contact.added() || contact.hasAddedUs() || contact.Blocked {
			if err = m.syncContact(ctx, contact, rawMessageHandler); err != nil {
				return false
			}
		}
		return true
	})

	cs, err := m.communitiesManager.JoinedAndPendingCommunitiesWithRequests()
	if err != nil {
		return err
	}
	for _, c := range cs {
		if err = m.syncCommunity(ctx, c, rawMessageHandler); err != nil {
			return err
		}
	}

	bookmarks, err := m.browserDatabase.GetBookmarks()
	if err != nil {
		return err
	}
	for _, b := range bookmarks {
		if err = m.SyncBookmark(ctx, b, rawMessageHandler); err != nil {
			return err
		}
	}

	trustedUsers, err := m.verificationDatabase.GetAllTrustStatus()
	if err != nil {
		return err
	}
	for id, ts := range trustedUsers {
		if err = m.SyncTrustedUser(ctx, id, ts, rawMessageHandler); err != nil {
			return err
		}
	}

	verificationRequests, err := m.verificationDatabase.GetVerificationRequests()
	if err != nil {
		return err
	}
	for i := range verificationRequests {
		if err = m.SyncVerificationRequest(ctx, &verificationRequests[i], rawMessageHandler); err != nil {
			return err
		}
	}

	err = m.syncSettings(rawMessageHandler)
	if err != nil {
		return err
	}

	err = m.syncProfilePicturesFromDatabase(rawMessageHandler)
	if err != nil {
		return err
	}

	if err = m.syncLatestContactRequests(ctx, rawMessageHandler); err != nil {
		return err
	}

	// we have to sync deleted keypairs as well
	keypairs, err := m.settings.GetAllKeypairs()
	if err != nil {
		return err
	}

	for _, kp := range keypairs {
		err = m.syncKeypair(kp, rawMessageHandler)
		if err != nil {
			return err
		}
	}

	// we have to sync deleted watch only accounts as well
	woAccounts, err := m.settings.GetAllWatchOnlyAccounts()
	if err != nil {
		return err
	}

	for _, woAcc := range woAccounts {
		err = m.syncWalletAccount(woAcc, rawMessageHandler)
		if err != nil {
			return err
		}
	}

	savedAddresses, err := m.savedAddressesManager.GetRawSavedAddresses()
	if err != nil {
		return err
	}

	for i := range savedAddresses {
		sa := savedAddresses[i]

		err = m.syncSavedAddress(ctx, sa, rawMessageHandler)
		if err != nil {
			return err
		}
	}

	if err = m.syncEnsUsernameDetails(ctx, rawMessageHandler); err != nil {
		return err
	}

	if err = m.syncDeleteForMeMessage(ctx, rawMessageHandler); err != nil {
		return err
	}

	err = m.syncAccountsPositions(rawMessageHandler)
	if err != nil {
		return err
	}

	err = m.syncProfileShowcasePreferences(context.Background(), rawMessageHandler)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) syncProfilePictures(rawMessageHandler RawMessageHandler, identityImages []*images.IdentityImage) error {
	if !m.hasPairedDevices() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pictures := make([]*protobuf.SyncProfilePicture, len(identityImages))
	clock, chat := m.getLastClockWithRelatedChat()
	for i, image := range identityImages {
		p := &protobuf.SyncProfilePicture{}
		p.Name = image.Name
		p.Payload = image.Payload
		p.Width = uint32(image.Width)
		p.Height = uint32(image.Height)
		p.FileSize = uint32(image.FileSize)
		p.ResizeTarget = uint32(image.ResizeTarget)
		if image.Clock == 0 {
			p.Clock = clock
		} else {
			p.Clock = image.Clock
		}
		pictures[i] = p
	}

	message := &protobuf.SyncProfilePictures{}
	message.KeyUid = m.account.KeyUID
	message.Pictures = pictures

	encodedMessage, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_SYNC_PROFILE_PICTURES,
		ResendType:  common.ResendTypeDataSync,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	chat.LastClockValue = clock
	return m.saveChat(chat)
}

func (m *Messenger) syncLatestContactRequests(ctx context.Context, rawMessageHandler RawMessageHandler) error {
	latestContactRequests, err := m.persistence.LatestContactRequests()

	if err != nil {
		return err
	}

	for _, r := range latestContactRequests {
		if r.ContactRequestState == common.ContactRequestStateAccepted || r.ContactRequestState == common.ContactRequestStateDismissed {
			accepted := r.ContactRequestState == common.ContactRequestStateAccepted
			err = m.syncContactRequestDecision(ctx, r.MessageID, r.ContactID, accepted, rawMessageHandler)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Messenger) syncContactRequestDecision(ctx context.Context, requestID, contactId string, accepted bool, rawMessageHandler RawMessageHandler) error {
	m.logger.Info("syncContactRequestDecision", zap.Any("from", requestID))
	if !m.hasPairedDevices() {
		return nil
	}

	clock, chat := m.getLastClockWithRelatedChat()

	var status protobuf.SyncContactRequestDecision_DecisionStatus
	if accepted {
		status = protobuf.SyncContactRequestDecision_ACCEPTED
	} else {
		status = protobuf.SyncContactRequestDecision_DECLINED
	}

	message := &protobuf.SyncContactRequestDecision{
		RequestId:      requestID,
		ContactId:      contactId,
		Clock:          clock,
		DecisionStatus: status,
	}

	encodedMessage, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	rawMessage := common.RawMessage{
		LocalChatID: chat.ID,
		Payload:     encodedMessage,
		MessageType: protobuf.ApplicationMetadataMessage_SYNC_CONTACT_REQUEST_DECISION,
		ResendType:  common.ResendTypeDataSync,
	}

	_, err = rawMessageHandler(ctx, rawMessage)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) getLastClockWithRelatedChat() (uint64, *Chat) {
	chatID := contactIDFromPublicKey(&m.identity.PublicKey)

	chat, ok := m.allChats.Load(chatID)
	if !ok {
		chat = OneToOneFromPublicKey(&m.identity.PublicKey, m.getTimesource())
		// We don't want to show the chat to the user
		chat.Active = false
	}

	m.allChats.Store(chat.ID, chat)
	clock, _ := chat.NextClockAndTimestamp(m.getTimesource())

	return clock, chat
}

func (m *Messenger) syncProfilePicturesFromDatabase(rawMessageHandler RawMessageHandler) error {
	keyUID := m.account.KeyUID
	identityImages, err := m.multiAccounts.GetIdentityImages(keyUID)
	if err != nil {
		return err
	}
	return m.syncProfilePictures(rawMessageHandler, identityImages)
}

func (m *Messenger) InitInstallations() error {
	installations, err := m.encryptor.GetOurInstallations(&m.identity.PublicKey)
	if err != nil {
		return err
	}

	for _, installation := range installations {
		m.allInstallations.Store(installation.ID, installation)
	}

	err = m.setInstallationHostname()
	if err != nil {
		return err
	}

	if m.telemetryClient != nil {
		installation, ok := m.allInstallations.Load(m.installationID)
		if ok {
			m.telemetryClient.SetDeviceType(installation.InstallationMetadata.DeviceType)
		}
	}

	return nil
}

func (m *Messenger) Installations() []*multidevice.Installation {
	installations := make([]*multidevice.Installation, m.allInstallations.Len())

	var i = 0
	m.allInstallations.Range(func(installationID string, installation *multidevice.Installation) (shouldContinue bool) {
		installations[i] = installation
		i++
		return true
	})
	return installations
}

func (m *Messenger) setInstallationMetadata(id string, data *multidevice.InstallationMetadata) error {
	installation, ok := m.allInstallations.Load(id)
	if !ok {
		return errors.New("no installation found")
	}

	installation.InstallationMetadata = data
	return m.encryptor.SetInstallationMetadata(m.IdentityPublicKey(), id, data)
}

func (m *Messenger) SetInstallationMetadata(id string, data *multidevice.InstallationMetadata) error {
	return m.setInstallationMetadata(id, data)
}

func (m *Messenger) SetInstallationName(id string, name string) error {
	installation, ok := m.allInstallations.Load(id)
	if !ok {
		return errors.New("no installation found")
	}

	installation.InstallationMetadata.Name = name
	return m.encryptor.SetInstallationName(m.IdentityPublicKey(), id, name)
}

// EnableInstallation enables an installation and returns the installation
func (m *Messenger) EnableInstallation(id string) (*multidevice.Installation, error) {
	installation, ok := m.allInstallations.Load(id)
	if !ok {
		return nil, errors.New("no installation found")
	}

	err := m.encryptor.EnableInstallation(&m.identity.PublicKey, id)
	if err != nil {
		return nil, err
	}
	installation.Enabled = true
	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allInstallations.Store(id, installation)
	return installation, nil
}

func (m *Messenger) DisableInstallation(id string) error {
	installation, ok := m.allInstallations.Load(id)
	if !ok {
		return errors.New("no installation found")
	}

	err := m.encryptor.DisableInstallation(&m.identity.PublicKey, id)
	if err != nil {
		return err
	}
	installation.Enabled = false
	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allInstallations.Store(id, installation)
	return nil
}
