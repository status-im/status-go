package protocol

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"

	gocommon "github.com/status-im/status-go/common"
	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/internal/crypto"
	types3 "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/internal/db/multiaccounts/accounts"
	multiaccountscommon "github.com/status-im/status-go/internal/db/multiaccounts/common"
	settings2 "github.com/status-im/status-go/internal/db/multiaccounts/settings"
	walletsettings "github.com/status-im/status-go/internal/db/multiaccounts/settings_wallet"
	types2 "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/pkg/pubsub"
	"github.com/status-im/status-go/protocol/contacts"
	"github.com/status-im/status-go/services/accounts/accountsevent"
	"github.com/status-im/status-go/services/browsers"
	"github.com/status-im/status-go/signal"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	archivecommons "github.com/status-im/status-go/protocol/communities/archive/commons"
	archivetypes "github.com/status-im/status-go/protocol/communities/archive/types"
	"github.com/status-im/status-go/protocol/protobuf"
	"github.com/status-im/status-go/protocol/requests"
	"github.com/status-im/status-go/protocol/syncing"
	v1protocol "github.com/status-im/status-go/protocol/v1"
	"github.com/status-im/status-go/protocol/verification"
)

var (
	ErrMessageNotAllowed                      = errors.New("message from a non-contact")
	ErrMessageForWrongChatType                = errors.New("message for the wrong chat type")
	ErrNotWatchOnlyAccount                    = errors.New("an account is not a watch only account")
	ErrWalletAccountNotSupportedForMobileApp  = errors.New("handling account is not supported for mobile app")
	ErrTryingToApplyOldWalletAccountsOrder    = errors.New("trying to apply old wallet accounts order")
	ErrTryingToStoreOldWalletAccount          = errors.New("trying to store an old wallet account")
	ErrTryingToStoreOldKeypair                = errors.New("trying to store an old keypair")
	ErrSomeFieldsMissingForWalletAccount      = errors.New("some fields are missing for wallet account")
	ErrUnknownKeypairForWalletAccount         = errors.New("keypair is not known for the wallet account")
	ErrInvalidCommunityID                     = errors.New("invalid community id")
	ErrTryingToApplyOldTokenPreferences       = errors.New("trying to apply old token preferences")
	ErrTryingToApplyOldCollectiblePreferences = errors.New("trying to apply old collectible preferences")
	ErrOutdatedCommunityRequestToJoin         = errors.New("outdated community request to join response")
)

// HandleMembershipUpdate updates a Chat instance according to the membership updates.
// It retrieves chat, if exists, and merges membership updates from the message.
// Finally, the Chat is updated with the new group events.
func (m *Messenger) HandleMembershipUpdateMessage(ctx context.Context, messageState *ReceivedMessageState, rawMembershipUpdate *protobuf.MembershipUpdateMessage, statusMessage *common.StatusMessage) error {
	chat, _ := messageState.AllChats.Load(rawMembershipUpdate.ChatId)

	return m.HandleMembershipUpdate(ctx, messageState, chat, rawMembershipUpdate, m.systemMessagesTranslations)
}

func (m *Messenger) HandleMembershipUpdate(ctx context.Context, messageState *ReceivedMessageState, chat *Chat, rawMembershipUpdate *protobuf.MembershipUpdateMessage, translations *systemMessageTranslationsMap) error {

	var group *v1protocol.Group
	var err error

	if rawMembershipUpdate == nil {
		return nil
	}

	logger := m.logger.With(zap.String("site", "HandleMembershipUpdate"))

	message, err := v1protocol.MembershipUpdateMessageFromProtobuf(rawMembershipUpdate)
	if err != nil {
		return err

	}

	if err := ValidateMembershipUpdateMessage(message, messageState.Timesource.GetCurrentTime()); err != nil {
		logger.Warn("failed to validate message", zap.Error(err))
		return err
	}

	senderID := messageState.CurrentMessageState.Contact.ID
	allowed, err := m.isMessageAllowedFrom(senderID, chat)
	if err != nil {
		return err
	}

	if !allowed {
		return ErrMessageNotAllowed
	}

	//if chat.InvitationAdmin exists means we are waiting for invitation request approvement, and in that case
	//we need to create a new chat instance like we don't have a chat and just use a regular invitation flow
	waitingForApproval := chat != nil && len(chat.InvitationAdmin) > 0
	ourKey := contacts.ContactIDFromPublicKey(&m.identity.PublicKey)
	isActive := messageState.CurrentMessageState.Contact.Added() || messageState.CurrentMessageState.Contact.ID == ourKey || waitingForApproval
	showPushNotification := isActive && messageState.CurrentMessageState.Contact.ID != ourKey

	// wasUserAdded indicates whether the user has been added to the group with this update
	wasUserAdded := false
	if chat == nil || waitingForApproval {
		if len(message.Events) == 0 {
			return errors.New("can't create new group chat without events")
		}

		//approve invitations
		if waitingForApproval {

			groupChatInvitation := &GroupChatInvitation{
				GroupChatInvitation: &protobuf.GroupChatInvitation{
					ChatId: message.ChatID,
				},
				From: types3.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey)),
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

		// A new chat must have contained us at some point
		wasEverMember, err := group.WasEverMember(ourKey)
		if err != nil {
			return err
		}

		if !wasEverMember {
			return errors.New("can't create a new group chat without us being a member")
		}

		wasUserAdded = group.IsMember(ourKey)
		newChat := CreateGroupChat(messageState.Timesource)
		// We set group chat inactive and create a notification instead
		// unless is coming from us or a contact or were waiting for approval.
		// Also, as message MEMBER_JOINED may come from member(not creator, not our contact)
		// reach earlier than CHAT_CREATED from creator, we need check if creator is our contact
		newChat.Active = isActive || m.checkIfCreatorIsOurContact(group)
		newChat.ReceivedInvitationAdmin = senderID
		chat = &newChat

		chat.updateChatFromGroupMembershipChanges(group)

		if err != nil {
			return errors.Wrap(err, "failed to get group creator")
		}

		publicKeys, err := group.MemberPublicKeys()
		if err != nil {
			return errors.Wrap(err, "failed to get group members")
		}
		filters, err := m.messaging.JoinGroupChat(publicKeys)
		if err != nil {
			return errors.Wrap(err, "failed to join group")
		}
		ok, err := m.scheduleSyncFilters(filters)
		if err != nil {
			return errors.Wrap(err, "failed to schedule sync filter")
		}
		m.logger.Debug("result of schedule sync filter", zap.Bool("ok", ok))
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
		chat.updateChatFromGroupMembershipChanges(group)

		// Reactivate deleted group chat on re-invite from contact
		chat.Active = chat.Active || (isActive && group.IsMember(ourKey))

		wasUserAdded = !existingGroup.IsMember(ourKey) && group.IsMember(ourKey)

		// Show push notifications when our key is added to members list and chat is Active
		showPushNotification = showPushNotification && wasUserAdded
	}
	maxClockVal := uint64(0)
	for _, event := range group.Events() {
		if event.ClockValue > maxClockVal {
			maxClockVal = event.ClockValue
		}
	}

	if chat.LastClockValue < maxClockVal {
		chat.LastClockValue = maxClockVal
	}

	// Only create a message notification when the user is added, not when removed
	if !chat.Active && wasUserAdded {
		chat.Highlight = true
		m.createMessageNotification(chat, messageState, chat.LastMessage)
	}

	profilePicturesVisibility, err := m.settings.GetProfilePicturesVisibility()
	if err != nil {
		return errors.Wrap(err, "failed to get profilePicturesVisibility setting")
	}

	if showPushNotification {
		// chat is highlighted for new group invites or group re-invites
		chat.Highlight = true
		// Respect Contact Requests notification setting for local notification
		if show, preview := m.contactRequestNotificationPreview(); show {
			messageState.Response.AddNotification(NewPrivateGroupInviteNotification(chat.ID, chat, messageState.CurrentMessageState.Contact, profilePicturesVisibility, preview))
		}
	}

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
		messageState.Response.AddMessage(message)
	}

	messageState.Response.AddChat(chat)
	// Store in chats map as it might be a new one
	messageState.AllChats.Store(chat.ID, chat)

	// explicit join has been removed, mimic auto-join for backward compatibility
	// no all cases are covered, e.g. if added to a group by non-contact
	autoJoin := chat.Active && wasUserAdded
	if autoJoin || waitingForApproval {
		_, err = m.ConfirmJoiningGroup(context.Background(), chat.ID)
		if err != nil {
			return err
		}
	}

	if message.Message != nil {
		return m.HandleChatMessage(ctx, messageState, message.Message, nil, false)
	} else if message.EmojiReaction != nil {
		return m.HandleEmojiReaction(ctx, messageState, message.EmojiReaction, nil)
	}

	return nil
}

func (m *Messenger) checkIfCreatorIsOurContact(group *v1protocol.Group) bool {
	creator, err := group.Creator()
	if err == nil {
		contact, _ := m.allContacts.Load(creator)
		return contact != nil && contact.Mutual()
	}
	m.logger.Warn("failed to get creator from group", zap.String("group name", group.Name()), zap.String("group chat id", group.ChatID()), zap.Error(err))
	return false
}

func (m *Messenger) createMessageNotification(chat *Chat, messageState *ReceivedMessageState, message *common.Message) {

	var notificationType ActivityCenterType
	if chat.OneToOne() {
		notificationType = ActivityCenterNotificationTypeNewOneToOne
	} else {
		notificationType = ActivityCenterNotificationTypeNewPrivateGroupChat
	}
	notification := &ActivityCenterNotification{
		ID:          types3.FromHex(chat.ID),
		Name:        chat.Name,
		Message:     message,
		Type:        notificationType,
		Author:      messageState.CurrentMessageState.Contact.ID,
		Timestamp:   messageState.CurrentMessageState.WhisperTimestamp,
		ChatID:      chat.ID,
		CommunityID: chat.CommunityID,
		UpdatedAt:   m.GetCurrentTimeInMillis(),
	}

	err := m.addActivityCenterNotification(messageState.Response, notification, nil)
	if err != nil {
		m.logger.Warn("failed to create activity center notification", zap.Error(err))
	}
}

func (m *Messenger) PendingNotificationContactRequest(contactID string) (*ActivityCenterNotification, error) {
	return m.persistence.ActiveContactRequestNotification(contactID)
}

func (m *Messenger) createDefaultContactRequest(contact *contacts.Contact, timestamp uint64) (*common.Message, error) {
	contactRequest, err := m.generateContactRequest(
		contact.ContactRequestRemoteClock,
		timestamp,
		contact,
		defaultContactRequestText(),
		false,
	)
	if err != nil {
		return nil, err
	}

	contactRequest.ID = defaultContactRequestID(contact.ID)

	// save this message
	err = m.persistence.SaveMessages([]*common.Message{contactRequest})
	if err != nil {
		return nil, err
	}

	return contactRequest, nil
}
func (m *Messenger) createContactRequestForContactUpdate(contact *contacts.Contact, messageState *ReceivedMessageState) (*common.Message, error) {
	contactRequest, err := m.createDefaultContactRequest(contact, messageState.CurrentMessageState.WhisperTimestamp)
	if err != nil {
		return nil, err
	}

	// save this message
	messageState.Response.AddMessage(contactRequest)

	return contactRequest, nil
}

func (m *Messenger) createIncomingContactRequestNotification(contact *contacts.Contact, messageState *ReceivedMessageState, contactRequest *common.Message, createNewNotification bool) error {
	if contactRequest.ContactRequestState == common.ContactRequestStateAccepted {
		// Pull one from the db if there
		notification, err := m.persistence.GetActivityCenterNotificationByID(types3.FromHex(contactRequest.ID))
		if err != nil {
			return err
		}

		if notification != nil {
			notification.Name = contact.PrimaryName()
			notification.Message = contactRequest
			notification.Read = true
			notification.Accepted = true
			notification.Dismissed = false
			notification.UpdatedAt = m.GetCurrentTimeInMillis()
			_, err = m.persistence.SaveActivityCenterNotification(notification, true)
			if err != nil {
				return err
			}
			messageState.Response.AddMessage(contactRequest)
			messageState.Response.AddActivityCenterNotification(notification)
		}
		return nil
	}

	if !createNewNotification {
		return nil
	}

	notification := &ActivityCenterNotification{
		ID:        types3.FromHex(contactRequest.ID),
		Name:      contact.PrimaryName(),
		Message:   contactRequest,
		Type:      ActivityCenterNotificationTypeContactRequest,
		Author:    contactRequest.From,
		Timestamp: contactRequest.WhisperTimestamp,
		ChatID:    contact.ID,
		Read:      contactRequest.ContactRequestState == common.ContactRequestStateAccepted || contactRequest.ContactRequestState == common.ContactRequestStateDismissed,
		Accepted:  contactRequest.ContactRequestState == common.ContactRequestStateAccepted,
		Dismissed: contactRequest.ContactRequestState == common.ContactRequestStateDismissed,
		UpdatedAt: m.GetCurrentTimeInMillis(),
	}

	return m.addActivityCenterNotification(messageState.Response, notification, nil)
}

func (m *Messenger) syncContactRequestForInstallationContact(contact *contacts.Contact, state *ReceivedMessageState, chat *Chat, outgoing bool) error {
	if contact.Mutual() {
		// We only need to generate a contact request if we are not mutual
		return nil
	}

	if chat == nil {
		return fmt.Errorf("no chat restored during the contact synchronisation, contact.ID = %s", gocommon.TruncateWithDot(contact.ID))
	}

	contactRequestID, err := m.persistence.LatestPendingContactRequestIDForContact(contact.ID)
	if err != nil {
		return err
	}

	if contactRequestID != "" {
		m.logger.Warn("syncContactRequestForInstallationContact: using existing contact request", zap.String("contactRequestID", contactRequestID))

		existingContactRequest, err := m.persistence.MessageByID(contactRequestID)
		if err != nil {
			if err != common.ErrRecordNotFound {
				return err
			}

			// The ID was returned by lookup but the message is missing. Fall back to
			// synthesizing a default request to keep contact-request actions available.
			m.logger.Warn("syncContactRequestForInstallationContact: contact request id found but message missing", zap.String("contactRequestID", contactRequestID))
		} else {
			if outgoing {
				notification := m.generateOutgoingContactRequestNotification(contact, existingContactRequest)
				notification.Timestamp = existingContactRequest.WhisperTimestamp
				err = m.addActivityCenterNotification(state.Response, notification, nil)
				if err != nil {
					return err
				}
			} else {
				err = m.createIncomingContactRequestNotification(contact, state, existingContactRequest, true)
				if err != nil {
					return err
				}
			}

			return nil
		}
	}

	clock, timestamp := chat.NextClockAndTimestamp(m.getTimesource())
	contactRequest, err := m.generateContactRequest(clock, timestamp, contact, defaultContactRequestText(), outgoing)
	if err != nil {
		return err
	}

	contactRequest.ID = defaultContactRequestID(contact.ID)

	state.Response.AddMessage(contactRequest)
	err = m.persistence.SaveMessages([]*common.Message{contactRequest})
	if err != nil {
		return err
	}

	if outgoing {
		notification := m.generateOutgoingContactRequestNotification(contact, contactRequest)
		err = m.addActivityCenterNotification(state.Response, notification, nil)
		if err != nil {
			return err
		}
	} else {
		err = m.createIncomingContactRequestNotification(contact, state, contactRequest, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Messenger) HandleSyncInstallationAccount(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncInstallationAccount, statusMessage *common.StatusMessage) error {
	// Noop
	return nil
}

func (m *Messenger) handleSyncChats(messageState *ReceivedMessageState, chats []*protobuf.SyncChat) error {
	for _, syncChat := range chats {
		oldChat, ok := m.allChats.Load(syncChat.Id)
		clock := int64(syncChat.Clock)
		if ok && oldChat.Timestamp > clock {
			// We already know this chat and its timestamp is newer than the syncChat
			continue
		}
		chat := &Chat{
			ID:                       syncChat.Id,
			Name:                     syncChat.Name,
			Timestamp:                clock,
			ReadMessagesAtClockValue: 0,
			Active:                   syncChat.Active,
			Muted:                    syncChat.Muted,
			Joined:                   clock,
			ChatType:                 ChatType(syncChat.ChatType),
			Highlight:                false,
		}
		if chat.PrivateGroupChat() {
			chat.MembershipUpdates = make([]v1protocol.MembershipUpdateEvent, len(syncChat.MembershipUpdateEvents))
			for i, membershipUpdate := range syncChat.MembershipUpdateEvents {
				chat.MembershipUpdates[i] = v1protocol.MembershipUpdateEvent{
					ClockValue: membershipUpdate.Clock,
					Type:       protobuf.MembershipUpdateEvent_EventType(membershipUpdate.Type),
					Members:    membershipUpdate.Members,
					Name:       membershipUpdate.Name,
					Signature:  membershipUpdate.Signature,
					ChatID:     membershipUpdate.ChatId,
					From:       membershipUpdate.From,
					RawPayload: membershipUpdate.RawPayload,
					Color:      membershipUpdate.Color,
					Image:      membershipUpdate.Image,
				}
			}
			group, err := newProtocolGroupFromChat(chat)
			if err != nil {
				return err
			}
			chat.updateChatFromGroupMembershipChanges(group)
		}

		err := m.saveChat(chat)
		if err != nil {
			return err
		}
		messageState.AllChats.Store(chat.ID, chat)
		messageState.Response.AddChat(chat)
	}

	return nil
}

func (m *Messenger) HandleSyncInstallationContactV2(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncInstallationContactV2, statusMessage *common.StatusMessage) error {
	canonicalID, err := contacts.ContactIDFromPublicKeyString(message.Id)
	if err != nil {
		m.logger.Warn("HandleSyncInstallationContactV2: failed to canonicalize contact ID, continuing with original",
			zap.String("contactID", message.Id),
			zap.Error(err))
		canonicalID = message.Id
	}
	message.Id = canonicalID

	// Ignore own contact installation
	if canonicalID == m.myHexIdentity() {
		m.logger.Warn("HandleSyncInstallationContactV2: skipping own contact")
		return nil
	}

	removed := message.Removed && !message.Blocked
	chat, ok := state.AllChats.Load(message.Id)
	if !ok && (message.Added || message.HasAddedUs || message.Muted) && !removed {
		pubKey, err := crypto.HexToPubkey(message.Id)
		if err != nil {
			return err
		}

		chat = OneToOneFromPublicKey(pubKey, state.Timesource)
		// We don't want to show the chat to the user
		chat.Active = false
	}

	contact, contactFound := state.AllContacts.Load(message.Id)
	if !contactFound {
		if message.Removed && !message.Blocked {
			// Nothing to do in case if contact doesn't exist
			return nil
		}

		var err error
		contact, err = contacts.BuildContactFromPkString(message.Id)
		if err != nil {
			return err
		}
	}

	// Check if we need to create a contact request
	if chat != nil {
		if message.ContactRequestRemoteClock != 0 || message.ContactRequestLocalClock != 0 {
			// Some local action about contact requests were performed,
			// process them
			contact.ProcessSyncContactRequestState(
				contacts.ContactRequestState(message.ContactRequestRemoteState),
				uint64(message.ContactRequestRemoteClock),
				contacts.ContactRequestState(message.ContactRequestLocalState),
				uint64(message.ContactRequestLocalClock))
			state.ModifiedContacts.Store(contact.ID, true)
			state.AllContacts.Store(contact.ID, contact)

			err := m.syncContactRequestForInstallationContact(contact, state, chat, contact.ContactRequestLocalState == contacts.ContactRequestStateSent)
			if err != nil {
				return err
			}
		}
	}

	// Sync last updated field
	// We don't set `LastUpdated`, since that would cause some issues
	// as `LastUpdated` tracks both display name & picture.
	// The case where it would break is as follow:
	// 1) User A pairs A1 with device A2.
	// 2) User B publishes display name and picture with LastUpdated = 3.
	// 3) Device A1 receives message from step 2.
	// 4) Device A1 syncs with A2 (which has not received message from step 3).
	// 5) Device A2 saves Display name and sets LastUpdated = 3,
	//    note that picture has not been set as it's not synced.
	// 6) Device A2 receives the message from 2. because LastUpdated is 3
	//    it will be discarded, A2 will not have B's picture.
	// The correct solution is to either sync profile image (expensive)
	// or split the clock for image/display name, so they can be synced
	// independently.
	if !contactFound || (contact.LastUpdated < message.LastUpdated) {
		if message.DisplayName != "" {
			contact.DisplayName = message.DisplayName
		}
		contact.CustomizationColor = multiaccountscommon.IDToColorFallbackToBlue(message.CustomizationColor)
		state.ModifiedContacts.Store(contact.ID, true)
		state.AllContacts.Store(contact.ID, contact)
	}

	if contact.LastUpdatedLocally < message.LastUpdatedLocally {
		// NOTE(cammellos): probably is cleaner to pass a flag
		// to method to tell them not to sync, or factor out in different
		// methods
		contact.IsSyncing = true
		defer func() {
			contact.IsSyncing = false
		}()

		if message.EnsName != "" && contact.EnsName != message.EnsName {
			contact.EnsName = message.EnsName
			publicKey, err := contact.PublicKey()
			if err != nil {
				return err
			}

			err = m.ENSVerified(crypto.PubkeyToHex(publicKey), message.EnsName)
			if err != nil {
				contact.ENSVerified = false
			}
			contact.ENSVerified = true
		}
		contact.CustomizationColor = multiaccountscommon.IDToColorFallbackToBlue(message.CustomizationColor)
		contact.LastUpdatedLocally = message.LastUpdatedLocally
		contact.LocalNickname = message.LocalNickname
		contact.TrustStatus = verification.TrustStatus(message.TrustStatus)
		contact.VerificationStatus = contacts.VerificationStatus(message.VerificationStatus)

		_, err := m.verificationDatabase.UpsertTrustStatus(contact.ID, contact.TrustStatus, message.LastUpdatedLocally)
		if err != nil {
			return err
		}

		if message.Blocked != contact.Blocked {
			if message.Blocked {
				state.AllContacts.Store(contact.ID, contact)
				response, err := m.BlockContact(context.TODO(), contact.ID, true)
				if err != nil {
					return err
				}
				err = state.Response.Merge(response)
				if err != nil {
					return err
				}
			} else {
				contact.Unblock(message.LastUpdatedLocally)
			}
		}
		if chat != nil && message.Muted != chat.Muted {
			if message.Muted {
				_, err := m.muteChat(chat, contact, time.Time{})
				if err != nil {
					return err
				}
			} else {
				err := m.unmuteChat(chat, contact)
				if err != nil {
					return err
				}
			}

			state.Response.AddChat(chat)
		}

		if err = m.updateContactImagesURL(contact); err != nil {
			return err
		}

		state.ModifiedContacts.Store(contact.ID, true)
		state.AllContacts.Store(contact.ID, contact)
	}

	if chat != nil {
		state.AllChats.Store(chat.ID, chat)
	}

	return nil
}

func (m *Messenger) HandleSyncProfilePictures(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncProfilePictures, statusMessage *common.StatusMessage) error {
	dbImages, err := m.multiAccounts.GetIdentityImages(message.KeyUid)
	if err != nil {
		return err
	}
	dbImageMap := make(map[string]*images.IdentityImage)
	for _, img := range dbImages {
		dbImageMap[img.Name] = img
	}
	idImages := make([]images.IdentityImage, len(message.Pictures))
	i := 0
	for _, message := range message.Pictures {
		dbImg := dbImageMap[message.Name]
		if dbImg != nil && message.Clock <= dbImg.Clock {
			continue
		}
		image := images.IdentityImage{
			Name:         message.Name,
			Payload:      message.Payload,
			Width:        int(message.Width),
			Height:       int(message.Height),
			FileSize:     int(message.FileSize),
			ResizeTarget: int(message.ResizeTarget),
			Clock:        message.Clock,
		}
		idImages[i] = image
		i++
	}

	if i == 0 {
		return nil
	}

	err = m.multiAccounts.StoreIdentityImages(message.KeyUid, idImages[:i], false)
	if err == nil {
		state.Response.IdentityImages = idImages[:i]
	}
	return err
}

func (m *Messenger) HandleSyncChat(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncChat, statusMessage *common.StatusMessage) error {
	chatID := message.Id
	existingChat, ok := state.AllChats.Load(chatID)
	if ok && (existingChat.Active || uint32(message.GetClock()/1000) < existingChat.SyncedTo) {
		return nil
	}

	chat := existingChat
	if !ok {
		chats := make([]*protobuf.SyncChat, 1)
		chats[0] = message
		return m.handleSyncChats(state, chats)
	}
	existingChat.Joined = int64(message.Clock)
	state.AllChats.Store(chat.ID, chat)

	state.Response.AddChat(chat)

	return nil
}

func (m *Messenger) HandleSyncChatRemoved(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncChatRemoved, statusMessage *common.StatusMessage) error {
	chat, ok := m.allChats.Load(message.Id)
	if !ok {
		return ErrChatNotFound
	}

	if chat.Joined > int64(message.Clock) {
		return nil
	}

	if chat.DeletedAtClockValue > message.Clock {
		return nil
	}

	if chat.PrivateGroupChat() {
		_, err := m.leaveGroupChat(context.Background(), state.Response, message.Id, true, false)
		if err != nil {
			return err
		}
	}

	response, err := m.deactivateChat(message.Id, message.Clock, false, true)
	if err != nil {
		return err
	}

	return state.Response.Merge(response)
}

func (m *Messenger) HandleSyncChatMessagesRead(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncChatMessagesRead, statusMessage *common.StatusMessage) error {
	chat, ok := m.allChats.Load(message.Id)
	if !ok {
		return ErrChatNotFound
	}

	if chat.ReadMessagesAtClockValue > message.Clock {
		return nil
	}

	err := m.markAllRead(message.Id, message.Clock, false)
	if err != nil {
		return err
	}

	state.Response.AddChat(chat)
	return nil
}

func (m *Messenger) handlePinMessage(pinner *contacts.Contact, whisperTimestamp uint64, response *MessengerResponse, message *protobuf.PinMessage, forceSeen bool) error {
	logger := m.logger.With(zap.String("site", "HandlePinMessage"))

	logger.Info("Handling pin message")

	publicKey, err := pinner.PublicKey()
	if err != nil {
		return err
	}

	pinMessage := &common.PinMessage{
		PinMessage: message,
		// MessageID:        message.MessageId,
		WhisperTimestamp: whisperTimestamp,
		From:             pinner.ID,
		SigPubKey:        publicKey,
		Alias:            pinner.Alias,
	}

	chat, err := m.matchChatEntity(pinMessage, protobuf.ApplicationMetadataMessage_PIN_MESSAGE)
	if err != nil {
		return err // matchChatEntity returns a descriptive error message
	}

	pinMessage.ID, err = generatePinMessageID(&m.identity.PublicKey, pinMessage, chat)
	if err != nil {
		return err
	}

	// If deleted-at is greater, ignore message
	if chat.DeletedAtClockValue >= pinMessage.Clock {
		return nil
	}

	if c, ok := m.allChats.Load(chat.ID); ok {
		chat = c
	}

	// Set the LocalChatID for the message
	pinMessage.LocalChatID = chat.ID

	inserted, err := m.persistence.SavePinMessage(pinMessage)
	if err != nil {
		return err
	}

	// Nothing to do, returning
	if !inserted {
		m.logger.Info("pin message already processed")
		return nil
	}

	if message.Pinned {
		id, err := generatePinMessageNotificationID(&m.identity.PublicKey, pinMessage, chat)
		if err != nil {
			return err
		}
		systemMessage := &common.Message{
			ChatMessage: &protobuf.ChatMessage{
				Clock:       message.Clock,
				Timestamp:   whisperTimestamp,
				ChatId:      chat.ID,
				MessageType: message.MessageType,
				ResponseTo:  message.MessageId,
				ContentType: protobuf.ChatMessage_SYSTEM_MESSAGE_PINNED_MESSAGE,
			},
			WhisperTimestamp: whisperTimestamp,
			ID:               id,
			LocalChatID:      chat.ID,
			From:             pinner.ID,
		}

		if forceSeen {
			systemMessage.Seen = true
		}

		response.AddMessage(systemMessage)
		chat.UnviewedMessagesCount++
	}

	if chat.LastClockValue < message.Clock {
		chat.LastClockValue = message.Clock
	}

	response.AddPinMessage(pinMessage)

	// Set in the modified maps chat
	response.AddChat(chat)
	m.allChats.Store(chat.ID, chat)
	return nil
}

func (m *Messenger) HandlePinMessage(ctx context.Context, state *ReceivedMessageState, message *protobuf.PinMessage, statusMessage *common.StatusMessage, fromArchive bool) error {
	return m.handlePinMessage(state.CurrentMessageState.Contact, state.CurrentMessageState.WhisperTimestamp, state.Response, message, fromArchive)
}

func (m *Messenger) handleAcceptContactRequest(
	response *MessengerResponse,
	contact *contacts.Contact,
	originalRequest *common.Message,
	clock uint64) (contacts.ContactRequestProcessingResponse, error) {

	m.logger.Debug("received contact request", zap.Uint64("clock-sent", clock), zap.Uint64("current-clock", contact.ContactRequestRemoteClock), zap.Uint64("current-state", uint64(contact.ContactRequestRemoteState)))
	if contact.ContactRequestRemoteClock > clock {
		m.logger.Debug("not handling accept since clock lower")
		return contacts.ContactRequestProcessingResponse{}, nil
	}

	// The contact request accepted wasn't found, a reason for this might
	// be that we sent a legacy contact request/contact-update, or another
	// device has sent it, and we haven't synchronized it
	if originalRequest == nil {
		return contact.ContactRequestAccepted(clock), nil
	}

	if originalRequest.LocalChatID != contact.ID {
		return contacts.ContactRequestProcessingResponse{}, errors.New("can't accept contact request not sent to user")
	}

	contact.ContactRequestAccepted(clock)

	originalRequest.ContactRequestState = common.ContactRequestStateAccepted

	err := m.persistence.SetContactRequestState(originalRequest.ID, originalRequest.ContactRequestState)
	if err != nil {
		return contacts.ContactRequestProcessingResponse{}, err
	}

	response.AddMessage(originalRequest)
	return contacts.ContactRequestProcessingResponse{}, nil
}

func (m *Messenger) handleAcceptContactRequestMessage(state *ReceivedMessageState, clock uint64, contactRequestID string, isOutgoing bool) error {
	request, err := m.persistence.MessageByID(contactRequestID)
	if err != nil && err != common.ErrRecordNotFound {
		return err
	}

	// We still want to handle acceptance of the CR even it was already accepted
	previouslyAccepted := request != nil && request.ContactRequestState == common.ContactRequestStateAccepted

	contact := state.CurrentMessageState.Contact
	wasMutual := contact.Mutual()

	// The request message will be added to the response here
	processingResponse, err := m.handleAcceptContactRequest(state.Response, contact, request, clock)
	if err != nil {
		return err
	}

	// If the state has changed from non-mutual contact, to mutual contact
	// we want to notify the user
	if contact.Mutual() {
		// We set the chat as active, this is currently the expected behavior
		// for mobile, it might change as we implement further the activity
		// center
		chat, _, err := m.getOneToOneAndNextClock(contact)
		if err != nil {
			return err
		}

		if chat.LastClockValue < clock {
			chat.LastClockValue = clock
		}

		// NOTE(cammellos): This will re-enable the chat if it was deleted, and only
		// after we became contact, currently seems safe, but that needs
		// discussing with UX.
		if chat.DeletedAtClockValue < clock {
			chat.Active = true
		}

		// Add mutual state update message only when transitioning to mutual.
		if !wasMutual && contact.Mutual() && !previouslyAccepted {
			clock, timestamp := chat.NextClockAndTimestamp(m.getTimesource())

			updateMessage, err := m.prepareMutualStateUpdateMessage(contact.ID, contacts.MutualStateUpdateTypeAdded, clock, timestamp, false)
			if err != nil {
				return err
			}

			err = m.prepareMessage(updateMessage, m.httpServer)
			if err != nil {
				return err
			}
			err = m.persistence.SaveMessages([]*common.Message{updateMessage})
			if err != nil {
				return err
			}
			state.Response.AddMessage(updateMessage)

			err = chat.UpdateFromMessage(updateMessage, m.getTimesource())
			if err != nil {
				return err
			}
			chat.UnviewedMessagesCount++

			// Dispatch profile message to add a contact to the encrypted profile part
			err = m.DispatchProfileShowcase()
			if err != nil {
				return err
			}
		}

		state.Response.AddChat(chat)
		state.AllChats.Store(chat.ID, chat)
	}

	if request != nil {
		if isOutgoing {
			notification := m.generateOutgoingContactRequestNotification(contact, request)
			err = m.addActivityCenterNotification(state.Response, notification, nil)
			if err != nil {
				return err
			}
		} else {
			err = m.createIncomingContactRequestNotification(contact, state, request, processingResponse.NewContactRequestReceived())
			if err != nil {
				return err
			}

			// With devices 1 and 2 paired, and userA logged in on both, while userB is on device 3:
			// When userA on device 1 sends a contact request to userB, userB accepts it on device 3.
			// The confirmation is sent to devices 1 and 2.
			// However, the contactRequestID in `AcceptContactRequestMessage` uses keccak256(...) instead of defaultContactRequestID(contact.ID).
			// Device 1 processes this, but device 2 doesn't due to an error `ErrRecordNotFound` from `m.persistence.MessageByID(contactRequestID)`.
			// The correct notification ID on device 2 should be defaultContactRequestID(contact.ID).
			// Thus, we must sync the accepted decision to device 2.
			err = m.syncActivityCenterAcceptedByIDs(context.TODO(), []types3.HexBytes{types3.FromHex(defaultContactRequestID(contact.ID))}, m.GetCurrentTimeInMillis())
			if err != nil {
				m.logger.Warn("could not sync activity center notification as accepted", zap.Error(err))
			}
		}
	}

	state.ModifiedContacts.Store(contact.ID, true)
	state.AllContacts.Store(contact.ID, contact)
	return nil
}

func (m *Messenger) HandleAcceptContactRequest(ctx context.Context, state *ReceivedMessageState, message *protobuf.AcceptContactRequest, statusMessage *common.StatusMessage) error {
	err := m.handleAcceptContactRequestMessage(state, message.Clock, message.Id, false)
	if err != nil {
		m.logger.Warn("could not accept contact request", zap.Error(err))
	}

	return nil
}

func (m *Messenger) handleRetractContactRequest(state *ReceivedMessageState, contact *contacts.Contact, message *protobuf.RetractContactRequest) error {
	if contact.ID == m.myHexIdentity() {
		m.logger.Debug("retraction coming from us, ignoring")
		return nil
	}

	m.logger.Debug("handling retracted contact request", zap.Uint64("clock", message.Clock))
	r := contact.ContactRequestRetracted(message.Clock, false)
	if !r.Processed() {
		m.logger.Debug("not handling retract since clock lower or already retracted")
		return nil
	}

	m.allContacts.Store(contact.ID, contact)

	if contact.Dismissed() {
		// We dismissed this contact request locally already, no need to proceed
		return nil
	}

	// System message for mutual state update
	chat, clock, err := m.getOneToOneAndNextClock(contact)
	if err != nil {
		return err
	}

	timestamp := m.getTimesource().GetCurrentTime()
	updateMessage, err := m.prepareMutualStateUpdateMessage(contact.ID, contacts.MutualStateUpdateTypeRemoved, clock, timestamp, false)
	if err != nil {
		return err
	}

	// Dispatch profile message to remove a contact from the encrypted profile part
	err = m.DispatchProfileShowcase()
	if err != nil {
		return err
	}

	err = m.prepareMessage(updateMessage, m.httpServer)
	if err != nil {
		return err
	}
	err = m.persistence.SaveMessages([]*common.Message{updateMessage})
	if err != nil {
		return err
	}
	state.Response.AddMessage(updateMessage)

	err = chat.UpdateFromMessage(updateMessage, m.getTimesource())
	if err != nil {
		return err
	}
	chat.UnviewedMessagesCount++
	state.Response.AddChat(chat)

	notification := &ActivityCenterNotification{
		ID:        types3.FromHex(uuid.New().String()),
		Type:      ActivityCenterNotificationTypeContactRemoved,
		Name:      contact.PrimaryName(),
		Author:    contact.ID,
		Timestamp: m.getTimesource().GetCurrentTime(),
		ChatID:    contact.ID,
		Read:      false,
		UpdatedAt: m.GetCurrentTimeInMillis(),
	}

	err = m.addActivityCenterNotification(state.Response, notification, nil)
	if err != nil {
		m.logger.Warn("failed to create activity center notification", zap.Error(err))
		return err
	}

	return nil
}

func (m *Messenger) HandleRetractContactRequest(ctx context.Context, state *ReceivedMessageState, message *protobuf.RetractContactRequest, statusMessage *common.StatusMessage) error {
	contact := state.CurrentMessageState.Contact
	err := m.handleRetractContactRequest(state, contact, message)
	if err != nil {
		return err
	}
	if contact.ID != m.myHexIdentity() {
		state.ModifiedContacts.Store(contact.ID, true)
	}

	return nil
}

func (m *Messenger) HandleContactUpdate(ctx context.Context, state *ReceivedMessageState, message *protobuf.ContactUpdate, statusMessage *common.StatusMessage) error {

	logger := m.logger.With(zap.String("site", "HandleContactUpdate"))
	if crypto.IsPubKeyEqual(state.CurrentMessageState.PublicKey, &m.identity.PublicKey) {
		logger.Warn("coming from us, ignoring")
		return nil
	}

	contact := state.CurrentMessageState.Contact
	chat, ok := state.AllChats.Load(contact.ID)

	allowed, err := m.isMessageAllowedFrom(state.CurrentMessageState.Contact.ID, chat)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrMessageNotAllowed
	}

	if err = gocommon.ValidateDisplayName(&message.DisplayName); err != nil {
		return err
	}

	if !ok {
		chat = OneToOneFromPublicKey(state.CurrentMessageState.PublicKey, state.Timesource)
		// We don't want to show the chat to the user
		chat.Active = false
	}

	logger.Debug("Handling contact update")

	if message.ContactRequestPropagatedState != nil {
		logger.Debug("handling contact request propagated state", zap.Any("state before update", contact.ContactRequestPropagatedState()))
		result := contact.ContactRequestPropagatedStateReceived(message.ContactRequestPropagatedState)
		if result.SendBackState() {
			logger.Debug("sending back state")
			// This is a bit dangerous, since it might trigger a ping-pong of contact updates
			// also it should backoff/debounce
			_, err = m.sendContactUpdate(context.Background(), contact.ID, contact.DisplayName, contact.EnsName, "", contact.CustomizationColor, m.dispatchMessage)
			if err != nil {
				return err
			}

		}
		if result.NewContactRequestReceived() {
			contactRequest, err := m.createContactRequestForContactUpdate(contact, state)
			if err != nil {
				return err
			}

			err = m.createIncomingContactRequestNotification(contact, state, contactRequest, true)
			if err != nil {
				return err
			}
		}

		logger.Debug("handled propagated state", zap.Any("state after update", contact.ContactRequestPropagatedState()))
		state.ModifiedContacts.Store(contact.ID, true)
		state.AllContacts.Store(contact.ID, contact)
	}

	if contact.LastUpdated < message.Clock {
		if contact.EnsName != message.EnsName {
			contact.EnsName = message.EnsName
			contact.ENSVerified = false
		}

		if len(message.DisplayName) != 0 {
			contact.DisplayName = message.DisplayName
		}

		contact.CustomizationColor = multiaccountscommon.IDToColorFallbackToBlue(message.CustomizationColor)

		r := contact.ContactRequestReceived(message.ContactRequestClock)
		if r.NewContactRequestReceived() {
			err = m.createIncomingContactRequestNotification(contact, state, nil, true)
			if err != nil {
				return err
			}
		}
		contact.LastUpdated = message.Clock
		state.ModifiedContacts.Store(contact.ID, true)
		state.AllContacts.Store(contact.ID, contact)
	}

	if chat.LastClockValue < message.Clock {
		chat.LastClockValue = message.Clock
	}

	if contact.Mutual() && chat.DeletedAtClockValue < message.Clock {
		chat.Active = true
	}

	state.Response.AddChat(chat)
	// TODO(samyoul) remove storing of an updated reference pointer?
	state.AllChats.Store(chat.ID, chat)

	return nil
}

func (m *Messenger) HandleSyncPairInstallation(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncPairInstallation, statusMessage *common.StatusMessage) error {
	logger := m.logger.With(zap.String("site", "HandlePairInstallation"))
	if err := ValidateReceivedPairInstallation(message, state.CurrentMessageState.WhisperTimestamp); err != nil {
		logger.Warn("failed to validate message", zap.Error(err))
		return err
	}

	installation, ok := state.AllInstallations.Load(message.InstallationId)
	if !ok {
		return errors.New("installation not found")
	}

	metadata := &types2.InstallationMetadata{
		Name:       message.Name,
		DeviceType: message.DeviceType,
	}

	installation.InstallationMetadata = metadata
	// TODO(samyoul) remove storing of an updated reference pointer?
	state.AllInstallations.Store(message.InstallationId, installation)
	state.ModifiedInstallations.Store(message.InstallationId, true)
	targeted := message.TargetInstallationId == m.installationID
	state.TargetedInstallations.Store(message.InstallationId, targeted)

	return nil
}

func (m *Messenger) HandleHistoryArchiveLinkMessage(state *ReceivedMessageState, communityPubKey *ecdsa.PublicKey, archiveLink string, clock uint64) error {
	m.logger.Debug("[LogosStorage][HandleHistoryArchiveLinkMessage] Handling history archive link message", zap.String("archiveLink", archiveLink))
	id := types3.HexBytes(crypto.CompressPubkey(communityPubKey))

	community, err := m.communitiesManager.GetByID(id)
	if err != nil && err != communities.ErrOrgNotFound {
		m.logger.Debug("Couldn't get community for community with id: ", zap.Any("id", id))
		return err
	}
	if community == nil {
		return nil
	}

	settings, err := m.communitiesManager.GetCommunitySettingsByID(id)
	if err != nil {
		m.logger.Debug("Couldn't get community settings for community with id: ", zap.Any("id", id))
		return err
	}
	if settings == nil {
		return nil
	}

	if m.archiveManager.IsStarted() && settings.HistoryArchiveSupportEnabled {
		m.logger.Debug("[LogosStorage][HandleHistoryArchiveLinkMessage] ArchiveManager is started and history archive support is enabled", zap.String("communityID", community.IDString()))

		lastArchiveLinkClock, err := m.communitiesManager.GetArchiveLinkMessageClock(id)
		if err != nil {
			return err
		}

		lastSeenArchiveLink, err := m.communitiesManager.GetLastSeenArchiveLink(id)
		if err != nil {
			return err
		}
		// We are only interested in a community archive archiveLink
		// if it originates from a community that the current account is
		// part of and doesn't own the private key at the same time
		m.logger.Debug("[LogosStorage][HandleHistoryArchiveLinkMessage] Checking community membership and lastArchiveLinkClock", zap.String("communityID", community.IDString()), zap.Uint64("lastArchiveLinkClock", lastArchiveLinkClock), zap.Uint64("messageClock", clock))

		if !community.IsControlNode() && community.Joined() && clock >= lastArchiveLinkClock {
			if lastSeenArchiveLink == archiveLink {
				m.logger.Debug("already processed this archiveLink")
				return nil
			}

			// All checks passed - proceed with download
			m.logger.Debug("[LogosStorage][HandleHistoryArchiveLinkMessage] Unseeding existing history archive link for community (if any)", zap.String("communityID", community.IDString()))

			m.archiveManager.UnseedHistoryArchive(id, lastSeenArchiveLink)
			currentTask := m.archiveManager.GetHistoryArchiveDownloadTask(id.String())

			go func(currentTask *archivetypes.HistoryArchiveDownloadTask, communityID types3.HexBytes) {
				defer gocommon.LogOnPanic()
				// Cancel ongoing download/import task
				if currentTask != nil && !currentTask.IsCancelled() {
					currentTask.Cancel()
					currentTask.Waiter.Wait()
				}

				// Create new task
				task := &archivetypes.HistoryArchiveDownloadTask{
					CancelChan: make(chan struct{}),
					Waiter:     *new(sync.WaitGroup),
					Cancelled:  false,
				}

				m.archiveManager.AddHistoryArchiveDownloadTask(communityID.String(), task)

				// this wait groups tracks the ongoing task for a particular community
				task.Waiter.Add(1)
				defer task.Waiter.Done()
				defer m.archiveManager.RemoveHistoryArchiveDownloadTask(communityID.String())

				// this wait groups tracks all ongoing tasks across communities
				m.shutdownWaitGroup.Add(1)
				defer m.shutdownWaitGroup.Done()

				m.logger.Debug("[LogosStorage][HandleHistoryArchiveLinkMessage] Calling downloadAndImportHistoryArchives", zap.String("archiveLink", archiveLink))

				m.downloadAndImportHistoryArchives(communityID, archiveLink, task.CancelChan)
			}(currentTask, id)

			m.logger.Debug("[LogosStorage][HandleHistoryArchiveLinkMessage] Updating archive link message clock", zap.String("communityID", community.IDString()), zap.Uint64("clock", clock))
			return m.communitiesManager.UpdateArchiveLinkMessageClock(id, clock)
		}
	}
	return nil
}

func (m *Messenger) downloadAndImportHistoryArchives(id types3.HexBytes, archiveLink string, cancel chan struct{}) {
	downloadTaskInfo, err := m.archiveManager.DownloadHistoryArchives(id, archiveLink, cancel)
	if err != nil {
		logMsg := "failed to download history archive data"
		if err == archivecommons.ErrArchiveTimedout {
			m.logger.Debug("archive has timed out, trying once more...")
			downloadTaskInfo, err = m.archiveManager.DownloadHistoryArchives(id, archiveLink, cancel)
			if err != nil {
				m.logger.Error(logMsg, zap.Error(err))
				return
			}
		} else {
			m.logger.Debug(logMsg, zap.Error(err))
			return
		}
	}

	if downloadTaskInfo.Cancelled {
		if downloadTaskInfo.TotalDownloadedArchivesCount > 0 {
			m.logger.Debug(fmt.Sprintf("downloaded %d of %d archives so far", downloadTaskInfo.TotalDownloadedArchivesCount, downloadTaskInfo.TotalArchivesCount))
		}
		m.archiveManager.UnseedHistoryArchive(id, archiveLink)
		return
	}

	m.logger.Debug("[LogosStorage][download_and_import_history_archives] Updating last seen archiveLink",
		zap.String("archiveLink", archiveLink))

	err = m.communitiesManager.UpdateLastSeenArchiveLink(id, archiveLink)
	if err != nil {
		m.logger.Error("couldn't update last seen archiveLink", zap.Error(err), zap.String("archiveLink", archiveLink))
	}

	m.archiveManager.PublishHistoryArchivesSeedingSignal(id)

	err = m.checkIfIMemberOfCommunity(id)
	if err != nil {
		return
	}

	m.logger.Debug("[LogosStorage][download_and_import_history_archives] Importing history archives now")

	err = m.importHistoryArchives(id, cancel, archiveLink)
	if err != nil {
		m.logger.Error("failed to import history archives", zap.Error(err))
		m.config.messengerSignalsHandler.DownloadingHistoryArchivesFinished(types3.EncodeHex(id))
		return
	}

	m.config.messengerSignalsHandler.DownloadingHistoryArchivesFinished(types3.EncodeHex(id))
}

func (m *Messenger) handleArchiveMessages(archiveMessages []*protobuf.WakuMessage) (*MessengerResponse, error) {

	messagesToHandle := make(map[types2.ChatFilter][]*types2.ReceivedMessage)

	for _, message := range archiveMessages {
		filter := m.messaging.ChatFilterByTopic(message.Topic)
		if filter != nil {
			shhMessage := &types2.ReceivedMessage{
				Sig:          message.Sig,
				Timestamp:    uint32(message.Timestamp),
				Topic:        types2.BytesToContentTopic(message.Topic),
				Payload:      message.Payload,
				Padding:      message.Padding,
				Hash:         message.Hash,
				ThirdPartyID: message.ThirdPartyId,
			}
			messagesToHandle[*filter] = append(messagesToHandle[*filter], shhMessage)
		}
	}

	importedMessages := make(map[types2.ChatFilter][]*types2.ReceivedMessage, 0)
	otherMessages := make(map[types2.ChatFilter][]*types2.ReceivedMessage, 0)

	for filter, messages := range messagesToHandle {
		for _, message := range messages {
			if message.ThirdPartyID != "" {
				importedMessages[filter] = append(importedMessages[filter], message)
			} else {
				otherMessages[filter] = append(otherMessages[filter], message)
			}
		}
	}

	err := m.handleImportedMessages(importedMessages)
	if err != nil {
		m.logger.Error("failed to handle imported messages", zap.Error(err))
		return nil, err
	}

	response, err := m.handleRetrievedMessages(otherMessages, false, true)
	if err != nil {
		m.logger.Error("failed to write history archive messages to database", zap.Error(err))
		return nil, err
	}

	return response, nil
}

func (m *Messenger) HandleCommunityCancelRequestToJoin(ctx context.Context, state *ReceivedMessageState, cancelRequestToJoinProto *protobuf.CommunityCancelRequestToJoin, statusMessage *common.StatusMessage) error {
	signer := state.CurrentMessageState.PublicKey
	if cancelRequestToJoinProto.CommunityId == nil {
		return ErrInvalidCommunityID
	}

	requestToJoin, err := m.communitiesManager.HandleCommunityCancelRequestToJoin(signer, cancelRequestToJoinProto)
	if err != nil {
		return err
	}

	state.Response.AddRequestToJoinCommunity(requestToJoin)

	// delete activity center notification
	notification, err := m.persistence.GetActivityCenterNotificationByID(requestToJoin.ID)
	if err != nil {
		return err
	}

	if notification != nil {
		updatedAt := m.GetCurrentTimeInMillis()
		notification.UpdatedAt = updatedAt
		// we shouldn't sync deleted notification here,
		// as the same user on different devices will receive the same message(CommunityCancelRequestToJoin) ?
		err = m.persistence.DeleteActivityCenterNotificationByID(types3.FromHex(requestToJoin.ID.String()), updatedAt)
		if err != nil {
			m.logger.Error("failed to delete notification from Activity Center", zap.Error(err))
			return err
		}

		// sending signal to client to remove the activity center notification from UI
		response := &MessengerResponse{}
		response.AddActivityCenterNotification(notification)

		signal.SendNewMessages(response)
	}

	return nil
}

// HandleCommunityRequestToJoin handles an community request to join
func (m *Messenger) HandleCommunityRequestToJoin(ctx context.Context, state *ReceivedMessageState, requestToJoinProto *protobuf.CommunityRequestToJoin, statusMessage *common.StatusMessage) error {
	signer := state.CurrentMessageState.PublicKey
	community, requestToJoin, err := m.communitiesManager.HandleCommunityRequestToJoin(signer, statusMessage.TransportLayer.Dst, requestToJoinProto)
	if err != nil {
		return err
	}

	switch requestToJoin.State {
	case communities.RequestToJoinStatePending:
		contact, _ := state.AllContacts.Load(contacts.ContactIDFromPublicKey(signer))
		contact.CustomizationColor = multiaccountscommon.IDToColorFallbackToBlue(requestToJoinProto.CustomizationColor)
		if len(requestToJoinProto.DisplayName) != 0 {
			contact.DisplayName = requestToJoinProto.DisplayName
			state.ModifiedContacts.Store(contact.ID, true)
			state.AllContacts.Store(contact.ID, contact)
			state.ModifiedContacts.Store(contact.ID, true)
		}

		state.Response.AddRequestToJoinCommunity(requestToJoin)

		// Respect Contact Requests notification setting for local notification
		if show, preview := m.contactRequestNotificationPreview(); show {
			profilePicturesVisibility, _ := m.settings.GetProfilePicturesVisibility()
			state.Response.AddNotification(NewCommunityRequestToJoinNotification(requestToJoin.ID.String(), community, contact, profilePicturesVisibility, preview))
		}

		// Activity Center notification, new for pending state
		notification := &ActivityCenterNotification{
			ID:               types3.FromHex(requestToJoin.ID.String()),
			Type:             ActivityCenterNotificationTypeCommunityMembershipRequest,
			Timestamp:        m.getTimesource().GetCurrentTime(),
			Author:           contact.ID,
			CommunityID:      community.IDString(),
			MembershipStatus: ActivityCenterMembershipStatusPending,
			Deleted:          false,
			UpdatedAt:        m.GetCurrentTimeInMillis(),
		}
		err = m.addActivityCenterNotification(state.Response, notification, nil)
		if err != nil {
			m.logger.Error("failed to save notification", zap.Error(err))
			return err
		}

	case communities.RequestToJoinStateDeclined:
		response, err := m.declineRequestToJoinCommunity(requestToJoin)
		if err == nil {
			err := state.Response.Merge(response)
			if err != nil {
				return err
			}
		} else {
			return err
		}

	case communities.RequestToJoinStateAccepted:
		response, err := m.acceptRequestToJoinCommunity(requestToJoin)
		if err == nil {
			err := state.Response.Merge(response) // new member has been added
			if err != nil {
				return err
			}
		} else if err == communities.ErrNoPermissionToJoin {
			// only control node will end up here as it's the only one that
			// performed token permission checks
			response, err = m.declineRequestToJoinCommunity(requestToJoin)
			if err == nil {
				err := state.Response.Merge(response)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			return err
		}

	case communities.RequestToJoinStateCanceled:
		// cancellation is handled by separate message
		fallthrough
	case communities.RequestToJoinStateAwaitingAddresses:
		// ownership changed request is handled only if owner kicked members and saved
		// temporary RequestToJoinStateAwaitingAddresses request
		fallthrough
	case communities.RequestToJoinStateAcceptedPending, communities.RequestToJoinStateDeclinedPending:
		// request can be marked as pending only manually
		return errors.New("invalid request state")
	}

	return nil
}

// HandleCommunityEditSharedAddresses handles an edit a user has made to their shared addresses
func (m *Messenger) HandleCommunityEditSharedAddresses(ctx context.Context, state *ReceivedMessageState, editRevealedAddressesProto *protobuf.CommunityEditSharedAddresses, statusMessage *common.StatusMessage) error {
	signer := state.CurrentMessageState.PublicKey
	if editRevealedAddressesProto.CommunityId == nil {
		return ErrInvalidCommunityID
	}

	err := m.communitiesManager.HandleCommunityEditSharedAddresses(signer, editRevealedAddressesProto)
	if err != nil {
		return err
	}

	community, err := m.communitiesManager.GetByIDString(string(editRevealedAddressesProto.GetCommunityId()))
	if err != nil {
		return err
	}

	state.Response.AddCommunity(community)
	return nil
}

func (m *Messenger) HandleCommunityRequestToJoinResponse(ctx context.Context, state *ReceivedMessageState, requestToJoinResponseProto *protobuf.CommunityRequestToJoinResponse, statusMessage *common.StatusMessage) error {
	signer := state.CurrentMessageState.PublicKey
	if requestToJoinResponseProto.CommunityId == nil {
		return ErrInvalidCommunityID
	}

	myRequestToJoinId := communities.CalculateRequestID(m.IdentityPublicKeyString(), requestToJoinResponseProto.CommunityId)

	requestToJoin, err := m.communitiesManager.GetRequestToJoin(myRequestToJoinId)
	if err != nil {
		return err
	}

	if requestToJoin.State == communities.RequestToJoinStateCanceled {
		return nil
	}

	community, err := m.communitiesManager.GetByID(requestToJoinResponseProto.CommunityId)
	if err != nil {
		return err
	}

	// check if it is outdated approved request to join
	clockSeconds := requestToJoinResponseProto.Clock / 1000
	isClockOutdated := clockSeconds < requestToJoin.Clock
	isDuplicateAfterMemberLeaves := clockSeconds == requestToJoin.Clock &&
		requestToJoin.State == communities.RequestToJoinStateAccepted && !community.Joined()

	if requestToJoin.State != communities.RequestToJoinStatePending &&
		(isClockOutdated || isDuplicateAfterMemberLeaves) {
		m.logger.Error(ErrOutdatedCommunityRequestToJoin.Error(),
			zap.String("communityId", community.IDString()),
			zap.Bool("joined", community.Joined()),
			zap.Uint64("requestToJoinResponseProto.Clock", requestToJoinResponseProto.Clock),
			zap.Uint64("requestToJoin.Clock", requestToJoin.Clock),
			zap.Uint8("state", uint8(requestToJoin.State)))
		return ErrOutdatedCommunityRequestToJoin
	}

	updatedRequest, err := m.communitiesManager.HandleCommunityRequestToJoinResponse(signer, requestToJoinResponseProto)
	if err != nil {
		return err
	}

	if updatedRequest != nil {
		state.Response.AddRequestToJoinCommunity(updatedRequest)
	}

	community, err = m.communitiesManager.GetByID(requestToJoinResponseProto.CommunityId)
	if err != nil {
		return err
	}

	if requestToJoinResponseProto.Accepted && community != nil && community.HasMember(&m.identity.PublicKey) {
		// Note: we can't guarantee that REQUEST_TO_JOIN_RESPONSE msg will be delivered before
		// COMMUNITY_DESCRIPTION msg, so this msg can return an ErrOrgAlreadyJoined if we
		// have been joined during COMMUNITY_DESCRIPTION
		response, err := m.JoinCommunity(context.Background(), requestToJoinResponseProto.CommunityId, false)
		if err != nil && err != communities.ErrOrgAlreadyJoined {
			return err
		}

		var communitySettings *communities.CommunitySettings
		if response != nil {
			// we merge to include chats in response signal to joining a community
			err = state.Response.Merge(response)
			if err != nil {
				return err
			}

			if len(response.Communities()) > 0 {
				communitySettings = response.CommunitiesSettings()[0]
				community = response.Communities()[0]
			}
		}

		if communitySettings == nil {
			communitySettings, err = m.communitiesManager.GetCommunitySettingsByID(requestToJoinResponseProto.CommunityId)
			if err != nil {
				return nil
			}
		}

		archiveLink := requestToJoinResponseProto.ArchiveLink
		if m.archiveManager.IsStarted() && communitySettings != nil && communitySettings.HistoryArchiveSupportEnabled && archiveLink != "" {

			currentTask := m.archiveManager.GetHistoryArchiveDownloadTask(community.IDString())
			if err := m.communitiesManager.UpdateArchiveLinkMessageClock(requestToJoinResponseProto.CommunityId, requestToJoinResponseProto.Clock); err != nil {
				return err
			}
			go func(currentTask *archivetypes.HistoryArchiveDownloadTask) {
				defer gocommon.LogOnPanic()
				// Cancel ongoing download/import task
				if currentTask != nil && !currentTask.IsCancelled() {
					currentTask.Cancel()
					currentTask.Waiter.Wait()
				}

				task := &archivetypes.HistoryArchiveDownloadTask{
					CancelChan: make(chan struct{}),
					Waiter:     *new(sync.WaitGroup),
					Cancelled:  false,
				}
				m.archiveManager.AddHistoryArchiveDownloadTask(community.IDString(), task)

				task.Waiter.Add(1)
				defer task.Waiter.Done()
				defer m.archiveManager.RemoveHistoryArchiveDownloadTask(community.IDString())

				m.shutdownWaitGroup.Add(1)
				defer m.shutdownWaitGroup.Done()

				m.logger.Debug("[LogosStorage][handle_community_request_to_join_response] Starting download and import of history archives", zap.String("archiveLink", archiveLink))

				m.downloadAndImportHistoryArchives(community.ID(), archiveLink, task.CancelChan)
			}(currentTask)
		}
	}

	return nil
}

func (m *Messenger) HandleCommunityRequestToLeave(ctx context.Context, state *ReceivedMessageState, requestToLeaveProto *protobuf.CommunityRequestToLeave, statusMessage *common.StatusMessage) error {
	signer := state.CurrentMessageState.PublicKey
	if requestToLeaveProto.CommunityId == nil {
		return ErrInvalidCommunityID
	}

	err := m.communitiesManager.HandleCommunityRequestToLeave(signer, requestToLeaveProto)
	if err != nil {
		return err
	}

	response, err := m.RemoveUserFromCommunity(requestToLeaveProto.CommunityId, crypto.PubkeyToHex(signer))
	if err != nil {
		return err
	}

	if len(response.Communities()) > 0 {
		state.Response.AddCommunity(response.Communities()[0])
	}

	return nil
}

func (m *Messenger) handleEditMessage(state *ReceivedMessageState, editMessage EditMessage) error {
	if err := ValidateEditMessage(editMessage.EditMessage); err != nil {
		return err
	}
	messageID := editMessage.MessageId

	originalMessage, err := m.getMessageFromResponseOrDatabase(state.Response, messageID)

	if err == common.ErrRecordNotFound {
		return m.persistence.SaveEdit(&editMessage)
	} else if err != nil {
		return err
	}

	originalMessageMentioned := originalMessage.Mentioned

	chat, ok := m.allChats.Load(originalMessage.LocalChatID)
	if !ok {
		return errors.New("chat not found")
	}

	// Check edit is valid
	if originalMessage.From != editMessage.From {
		return errors.New("invalid edit, not the right author")
	}

	// Check that edit should be applied
	if originalMessage.EditedAt >= editMessage.Clock {
		return m.persistence.SaveEdit(&editMessage)
	}

	// applyEditMessage modifies the message. Changing the variable name to make it clearer
	editedMessage := originalMessage
	// Update message and return it
	err = m.applyEditMessage(editMessage.EditMessage, editedMessage)
	if err != nil {
		return err
	}

	needToSaveChat := false
	if chat.LastMessage != nil && chat.LastMessage.ID == editedMessage.ID {
		chat.LastMessage = editedMessage
		needToSaveChat = true
	}
	responseTo, err := m.persistence.MessageByID(editedMessage.ResponseTo)

	if err != nil && err != common.ErrRecordNotFound {
		return err
	}

	err = state.updateExistingActivityCenterNotification(m.identity.PublicKey, m, editedMessage, responseTo)
	if err != nil {
		return err
	}

	editedMessageHasMentions := editedMessage.Mentioned

	// Messages in OneToOne chats increase the UnviewedMentionsCount whether or not they include a Mention
	if !chat.OneToOne() {
		if editedMessageHasMentions && !originalMessageMentioned && !editedMessage.Seen {
			// Increase unviewed count when the edited message has a mention and didn't have one before
			chat.UnviewedMentionsCount++
			needToSaveChat = true
		} else if !editedMessageHasMentions && originalMessageMentioned && !editedMessage.Seen {
			// Opposite of above, the message had a mention, but no longer does, so we reduce the count
			chat.UnviewedMentionsCount--
			needToSaveChat = true
		}
	}

	if needToSaveChat {
		err := m.saveChat(chat)
		if err != nil {
			return err
		}
	}

	state.Response.AddMessage(editedMessage)

	// pull updated messages
	updatedMessages, err := m.persistence.MessagesByResponseTo(messageID)
	if err != nil {
		return err
	}
	state.Response.AddMessages(updatedMessages)

	state.Response.AddChat(chat)

	return nil
}

func (m *Messenger) HandleEditMessage(ctx context.Context, state *ReceivedMessageState, editProto *protobuf.EditMessage, statusMessage *common.StatusMessage) error {
	return m.handleEditMessage(state, EditMessage{
		EditMessage: editProto,
		From:        state.CurrentMessageState.Contact.ID,
		ID:          state.CurrentMessageState.MessageID,
		SigPubKey:   state.CurrentMessageState.PublicKey,
	})
}

func (m *Messenger) handleDeleteMessage(ctx context.Context, state *ReceivedMessageState, deleteMessage *DeleteMessage) error {
	if deleteMessage == nil {
		return nil
	}
	if err := ValidateDeleteMessage(deleteMessage.DeleteMessage); err != nil {
		return err
	}

	messageID := deleteMessage.MessageId
	// Check if it's already in the response
	originalMessage := state.Response.GetMessage(messageID)
	// otherwise pull from database
	if originalMessage == nil {
		var err error
		originalMessage, err = m.persistence.MessageByID(messageID)

		if err != nil && err != common.ErrRecordNotFound {
			return err
		}
	}

	if originalMessage == nil {
		return m.persistence.SaveDelete(deleteMessage)
	}

	chat, ok := m.allChats.Load(originalMessage.LocalChatID)
	if !ok {
		return errors.New("chat not found")
	}

	var canDeleteMessageForEveryone = false
	if originalMessage.From != deleteMessage.From {
		fromPublicKey, err := crypto.HexToPubkey(deleteMessage.From)
		if err != nil {
			return err
		}
		if chat.ChatType == ChatTypeCommunityChat {
			canDeleteMessageForEveryone = m.CanDeleteMessageForEveryoneInCommunity(chat.CommunityID, fromPublicKey)
			if !canDeleteMessageForEveryone {
				return ErrInvalidDeletePermission
			}
		} else if chat.ChatType == ChatTypePrivateGroupChat {
			canDeleteMessageForEveryone = m.CanDeleteMessageForEveryoneInPrivateGroupChat(chat, fromPublicKey)
			if !canDeleteMessageForEveryone {
				return ErrInvalidDeletePermission
			}
		}

		// Check edit is valid
		if !canDeleteMessageForEveryone {
			return errors.New("invalid delete, not the right author")
		}
	}

	messagesToDelete, err := m.getOtherMessagesInAlbum(originalMessage, originalMessage.LocalChatID)
	if err != nil {
		return err
	}

	unreadCountDecreased := false
	for _, messageToDelete := range messagesToDelete {
		messageToDelete.Deleted = true
		messageToDelete.DeletedBy = deleteMessage.DeleteMessage.DeletedBy
		err := m.persistence.SaveMessages([]*common.Message{messageToDelete})
		if err != nil {
			return err
		}

		// we shouldn't sync deleted notification here,
		// as the same user on different devices will receive the same message(DeleteMessage) ?
		m.logger.Debug("deleting activity center notification for message", zap.String("chatID", chat.ID), zap.String("messageID", messageToDelete.ID))
		_, err = m.persistence.DeleteActivityCenterNotificationForMessage(chat.ID, messageToDelete.ID, m.GetCurrentTimeInMillis())

		if err != nil {
			m.logger.Warn("failed to delete notifications for deleted message", zap.Error(err))
			return err
		}

		// Reduce chat mention count and unread count if unread
		if !messageToDelete.Seen && !unreadCountDecreased {
			unreadCountDecreased = true
			if chat.UnviewedMessagesCount > 0 {
				chat.UnviewedMessagesCount--
			}
			if chat.UnviewedMentionsCount > 0 && (messageToDelete.Mentioned || messageToDelete.Replied) {
				chat.UnviewedMentionsCount--
			}
			err := m.saveChat(chat)
			if err != nil {
				return err
			}
		}

		state.Response.AddRemovedMessage(&RemovedMessage{MessageID: messageToDelete.ID, ChatID: chat.ID, DeletedBy: deleteMessage.DeleteMessage.DeletedBy})
		state.Response.AddNotification(DeletedMessageNotification(messageToDelete.ID, chat))
		state.Response.AddActivityCenterNotification(&ActivityCenterNotification{
			ID:      types3.FromHex(messageToDelete.ID),
			Deleted: true,
		})

		if chat.LastMessage != nil && chat.LastMessage.ID == messageToDelete.ID {
			chat.LastMessage = messageToDelete
			err = m.saveChat(chat)
			if err != nil {
				return nil
			}
		}

		messages, err := m.persistence.LatestMessageByChatID(chat.ID)
		if err != nil {
			return err
		}
		if len(messages) > 0 {
			previousNotDeletedMessage := messages[0]
			if previousNotDeletedMessage != nil && !previousNotDeletedMessage.Seen && chat.OneToOne() && !chat.Active {
				m.createMessageNotification(chat, state, previousNotDeletedMessage)
			}
		}

		// pull updated messages
		updatedMessages, err := m.persistence.MessagesByResponseTo(messageToDelete.ID)
		if err != nil {
			return err
		}
		state.Response.AddMessages(updatedMessages)
	}

	state.Response.AddChat(chat)

	return nil
}

func (m *Messenger) HandleDeleteMessage(ctx context.Context, state *ReceivedMessageState, deleteProto *protobuf.DeleteMessage, statusMessage *common.StatusMessage) error {
	return m.handleDeleteMessage(ctx, state, &DeleteMessage{
		DeleteMessage: deleteProto,
		From:          state.CurrentMessageState.Contact.ID,
		ID:            state.CurrentMessageState.MessageID,
		SigPubKey:     state.CurrentMessageState.PublicKey,
	})
}

func (m *Messenger) getMessageFromResponseOrDatabase(response *MessengerResponse, messageID string) (*common.Message, error) {
	originalMessage := response.GetMessage(messageID)
	// otherwise pull from database
	if originalMessage != nil {
		return originalMessage, nil
	}

	return m.persistence.MessageByID(messageID)
}

func (m *Messenger) HandleSyncDeleteForMeMessage(ctx context.Context, state *ReceivedMessageState, deleteForMeMessage *protobuf.SyncDeleteForMeMessage, statusMessage *common.StatusMessage) error {
	if err := ValidateDeleteForMeMessage(deleteForMeMessage); err != nil {
		return err
	}

	messageID := deleteForMeMessage.MessageId
	// Check if it's already in the response
	originalMessage, err := m.getMessageFromResponseOrDatabase(state.Response, messageID)

	if err == common.ErrRecordNotFound {
		return m.persistence.SaveOrUpdateDeleteForMeMessage(deleteForMeMessage)
	} else if err != nil {
		return err
	}

	chat, ok := m.allChats.Load(originalMessage.LocalChatID)
	if !ok {
		return errors.New("chat not found")
	}

	messagesToDelete, err := m.getOtherMessagesInAlbum(originalMessage, originalMessage.LocalChatID)
	if err != nil {
		return err
	}

	for _, messageToDelete := range messagesToDelete {
		messageToDelete.DeletedForMe = true

		err := m.persistence.SaveMessages([]*common.Message{messageToDelete})
		if err != nil {
			return err
		}

		// we shouldn't sync deleted notification here,
		// as the same user on different devices will receive the same message(DeleteForMeMessage) ?
		m.logger.Debug("deleting activity center notification for message", zap.String("chatID", chat.ID), zap.String("messageID", messageToDelete.ID))
		_, err = m.persistence.DeleteActivityCenterNotificationForMessage(chat.ID, messageToDelete.ID, m.GetCurrentTimeInMillis())
		if err != nil {
			m.logger.Warn("failed to delete notifications for deleted message", zap.Error(err))
			return err
		}

		if chat.LastMessage != nil && chat.LastMessage.ID == messageToDelete.ID {
			chat.LastMessage = messageToDelete
			err = m.saveChat(chat)
			if err != nil {
				return nil
			}
		}

		state.Response.AddMessage(messageToDelete)
	}
	state.Response.AddChat(chat)

	return nil
}

func handleContactRequestChatMessage(receivedMessage *common.Message, contact *contacts.Contact, outgoing bool, logger *zap.Logger) (bool, error) {
	receivedMessage.ContactRequestState = common.ContactRequestStatePending

	var response contacts.ContactRequestProcessingResponse

	if outgoing {
		response = contact.ContactRequestSent(receivedMessage.Clock)
	} else {
		response = contact.ContactRequestReceived(receivedMessage.Clock)
	}
	if !response.Processed() {
		logger.Info("not handling contact message since clock lower")
		return false, nil

	}

	if contact.Mutual() {
		receivedMessage.ContactRequestState = common.ContactRequestStateAccepted
	}

	return response.NewContactRequestReceived(), nil
}

func (m *Messenger) handleChatMessage(ctx context.Context, state *ReceivedMessageState, forceSeen bool) error {
	logger := m.logger.With(zap.String("site", "handleChatMessage"))
	if err := ValidateReceivedChatMessage(state.CurrentMessageState.Message, state.CurrentMessageState.WhisperTimestamp); err != nil {
		logger.Warn("failed to validate message", zap.Error(err))
		return err
	}

	receivedMessage := &common.Message{
		ID:               state.CurrentMessageState.MessageID,
		ChatMessage:      state.CurrentMessageState.Message,
		From:             state.CurrentMessageState.Contact.ID,
		Alias:            state.CurrentMessageState.Contact.Alias,
		SigPubKey:        state.CurrentMessageState.PublicKey,
		WhisperTimestamp: state.CurrentMessageState.WhisperTimestamp,
	}

	// is the message coming from us?
	isSyncMessage := crypto.IsPubKeyEqual(receivedMessage.SigPubKey, &m.identity.PublicKey)

	if forceSeen || isSyncMessage {
		receivedMessage.Seen = true
	}

	err := receivedMessage.PrepareContent(m.myHexIdentity())
	if err != nil {
		return fmt.Errorf("failed to prepare message content: %v", err)
	}

	// If the message is a reply, we check if it's a reply to one of own own messages
	if receivedMessage.ResponseTo != "" {
		repliedTo, err := m.persistence.MessageByID(receivedMessage.ResponseTo)
		if err != nil && (err == sql.ErrNoRows || err == common.ErrRecordNotFound) {
			logger.Error("failed to get quoted message", zap.Error(err))
		} else if err != nil {
			return err
		} else if repliedTo.From == m.myHexIdentity() {
			receivedMessage.Replied = true
		}
	}

	chat, err := m.matchChatEntity(receivedMessage, protobuf.ApplicationMetadataMessage_CHAT_MESSAGE)
	if err != nil {
		return err // matchChatEntity returns a descriptive error message
	}

	if chat.ReadMessagesAtClockValue >= receivedMessage.Clock {
		receivedMessage.Seen = true
	}

	allowed, err := m.isMessageAllowedFrom(state.CurrentMessageState.Contact.ID, chat)
	if err != nil {
		return err
	}

	if !allowed {
		return ErrMessageNotAllowed
	}

	if chat.ChatType == ChatTypeCommunityChat {
		communityID, err := types3.DecodeHex(chat.CommunityID)
		if err != nil {
			return err
		}

		community, err := m.communitiesManager.GetByIDReadonly(communityID)
		if err != nil {
			return err
		}

		if community == nil {
			logger.Warn("community not found for msg",
				zap.String("messageID", receivedMessage.ID),
				zap.String("from", receivedMessage.From),
				zap.String("communityID", chat.CommunityID))
			return communities.ErrOrgNotFound
		}

		pk, err := crypto.HexToPubkey(state.CurrentMessageState.Contact.ID)
		if err != nil {
			return err
		}

		if community.IsBanned(pk) {
			logger.Warn("skipping msg from banned user",
				zap.String("messageID", receivedMessage.ID),
				zap.String("from", receivedMessage.From),
				zap.String("communityID", chat.CommunityID))
			return errors.New("received a messaged from banned user")
		}
	}

	// It looks like status-mobile created profile chats as public chats
	// so for now we need to check for the presence of "@" in their chatID
	if chat.Public() && !chat.ProfileUpdates() {
		switch receivedMessage.ContentType {
		case protobuf.ChatMessage_IMAGE:
			return errors.New("images are not allowed in public chats")
		case protobuf.ChatMessage_AUDIO:
			return errors.New("audio messages are not allowed in public chats")
		}
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

	if err := m.updateChatFirstMessageTimestamp(chat, whisperToUnixTimestamp(receivedMessage.WhisperTimestamp), state.Response); err != nil {
		return err
	}

	// Our own message, mark as sent
	if isSyncMessage {
		receivedMessage.OutgoingStatus = common.OutgoingStatusSent
	} else if !receivedMessage.Seen {
		// Increase unviewed count
		skipUpdateUnviewedCountForAlbums := false
		if receivedMessage.ContentType == protobuf.ChatMessage_IMAGE {
			image := receivedMessage.GetImage()

			if image != nil && image.AlbumId != "" {
				// Skip unviewed counts increasing for other messages from album if we have it in memory
				for _, message := range state.Response.Messages() {
					if receivedMessage.ContentType == protobuf.ChatMessage_IMAGE {
						img := message.GetImage()
						if img != nil && img.AlbumId != "" && img.AlbumId == image.AlbumId {
							skipUpdateUnviewedCountForAlbums = true
							break
						}
					}
				}

				if !skipUpdateUnviewedCountForAlbums {
					messages, err := m.persistence.AlbumMessages(chat.ID, image.AlbumId)
					if err != nil {
						return err
					}

					// Skip unviewed counts increasing for other messages from album if we have it in db
					skipUpdateUnviewedCountForAlbums = len(messages) > 0
				}
			}
		}
		if !skipUpdateUnviewedCountForAlbums {
			m.updateUnviewedCounts(chat, receivedMessage)
		}
	}

	contact := state.CurrentMessageState.Contact

	if chat.ChatType == ChatTypeOneToOne && contact.Dismissed() {
		m.logger.Info("ignoring one on one messages dismissed contacts")
		return nil
	}

	if receivedMessage.ContentType == protobuf.ChatMessage_DISCORD_MESSAGE {
		discordMessage := receivedMessage.GetDiscordMessage()
		discordMessageAuthor := discordMessage.GetAuthor()
		discordMessageAttachments := discordMessage.GetAttachments()

		state.Response.AddDiscordMessage(discordMessage)
		state.Response.AddDiscordMessageAuthor(discordMessageAuthor)

		if len(discordMessageAttachments) > 0 {
			state.Response.AddDiscordMessageAttachments(discordMessageAttachments)
		}
	}

	err = m.checkForEdits(receivedMessage)
	if err != nil {
		return err
	}

	err = m.checkForDeletes(receivedMessage)
	if err != nil {
		return err
	}

	err = m.checkForDeleteForMes(receivedMessage)
	if err != nil {
		return err
	}

	if !receivedMessage.Deleted && !receivedMessage.DeletedForMe {
		err = chat.UpdateFromMessage(receivedMessage, m.getTimesource())
		if err != nil {
			return err
		}
	}
	// Set in the modified maps chat
	state.Response.AddChat(chat)
	// TODO(samyoul) remove storing of an updated reference pointer?
	m.allChats.Store(chat.ID, chat)

	if !isSyncMessage && receivedMessage.EnsName != "" {
		oldRecord, err := m.ensVerifier.Add(contact.ID, receivedMessage.EnsName, receivedMessage.Clock)
		if err != nil {
			m.logger.Warn("failed to verify ENS name", zap.Error(err))
		} else if oldRecord == nil {
			// If oldRecord is nil, a new verification process will take place
			// so we reset the record
			contact.ENSVerified = false
			state.ModifiedContacts.Store(contact.ID, true)
			state.AllContacts.Store(contact.ID, contact)
		}
	}

	if !isSyncMessage && contact.DisplayName != receivedMessage.DisplayName && len(receivedMessage.DisplayName) != 0 {
		contact.DisplayName = receivedMessage.DisplayName
		state.ModifiedContacts.Store(contact.ID, true)
	}

	if customizationColor := multiaccountscommon.IDToColorFallbackToBlue(receivedMessage.CustomizationColor); !isSyncMessage && receivedMessage.CustomizationColor != 0 && contact.CustomizationColor != customizationColor {
		contact.CustomizationColor = customizationColor
		state.ModifiedContacts.Store(contact.ID, true)
	}

	if receivedMessage.ContentType == protobuf.ChatMessage_COMMUNITY {
		m.logger.Debug("Handling community content type")

		signer, description, err := communities.UnwrapCommunityDescriptionMessage(receivedMessage.GetCommunity())
		if err != nil {
			return err
		}

		err = m.handleCommunityDescription(state, signer, description, receivedMessage.GetCommunity(), nil)
		if err != nil {
			return err
		}

		if len(description.ID) != 0 {
			receivedMessage.CommunityID = description.ID
		} else {
			// Backward compatibility
			receivedMessage.CommunityID = types3.EncodeHex(crypto.CompressPubkey(signer))
		}
	}

	receivedAContactRequest := false
	// If we receive some propagated state from someone who's not
	// our paired device, we handle it
	if receivedMessage.ContactRequestPropagatedState != nil && !isSyncMessage {
		result := contact.ContactRequestPropagatedStateReceived(receivedMessage.ContactRequestPropagatedState)
		if result.SendBackState() {
			_, err = m.sendContactUpdate(context.Background(), contact.ID, "", "", "", "", m.dispatchMessage)
			if err != nil {
				return err
			}
		}
		if result.NewContactRequestReceived() {
			receivedAContactRequest = true

			if contact.HasAddedUs() && !contact.Mutual() {
				receivedMessage.ContactRequestState = common.ContactRequestStatePending
			}

			// Add mutual state update message for outgoing contact request
			clock := receivedMessage.Clock - 1
			updateMessage, err := m.prepareMutualStateUpdateMessage(contact.ID, contacts.MutualStateUpdateTypeSent, clock, receivedMessage.Timestamp, false)
			if err != nil {
				return err
			}

			err = m.prepareMessage(updateMessage, m.httpServer)
			if err != nil {
				return err
			}
			err = m.persistence.SaveMessages([]*common.Message{updateMessage})
			if err != nil {
				return err
			}
			state.Response.AddMessage(updateMessage)

			if !contact.Mutual() {
				// Only create the notification if we are not mutual yet
				err = m.createIncomingContactRequestNotification(contact, state, receivedMessage, true)
				if err != nil {
					return err
				}
			} else {
				// We already sent a contact request, so we can mark the old notification as Accepted
				notification, err := m.persistence.GetActivityCenterNotificationByTypeAuthorAndChatID(ActivityCenterNotificationTypeContactRequest, m.myHexIdentity(), contact.ID)
				if err != nil {
					return err
				}
				if notification != nil && notification.Message.ContactRequestState != common.ContactRequestStateAccepted {
					notification.Message.ContactRequestState = common.ContactRequestStateAccepted
					notification.UpdatedAt = m.GetCurrentTimeInMillis()
					err = m.addActivityCenterNotification(state.Response, notification, nil)
					if err != nil {
						return err
					}
				}
			}
		}
		state.ModifiedContacts.Store(contact.ID, true)
		state.AllContacts.Store(contact.ID, contact)
	}

	if receivedMessage.ContentType == protobuf.ChatMessage_CONTACT_REQUEST && chat.OneToOne() {
		chatContact := contact
		if isSyncMessage {
			chatContact, err = m.BuildContact(&requests.BuildContact{PublicKey: chat.ID})
			if err != nil {
				return err
			}
		}

		if receivedMessage.CustomizationColor != 0 {
			chatContact.CustomizationColor = multiaccountscommon.IDToColorFallbackToBlue(receivedMessage.CustomizationColor)
		}

		if (!receivedAContactRequest && chatContact.Mutual()) || chatContact.Dismissed() {
			m.logger.Info("ignoring contact request message for a mutual or dismissed contact")
			return nil
		}

		sendNotification, err := handleContactRequestChatMessage(receivedMessage, chatContact, isSyncMessage, m.logger)
		if err != nil {
			m.logger.Error("failed to handle contact request message", zap.Error(err))
			return err
		}
		state.ModifiedContacts.Store(chatContact.ID, true)
		state.AllContacts.Store(chatContact.ID, chatContact)

		if sendNotification {
			err = m.createIncomingContactRequestNotification(chatContact, state, receivedMessage, true)
			if err != nil {
				return err
			}
		}
	} else if receivedMessage.ContentType == protobuf.ChatMessage_COMMUNITY {
		chat.Highlight = true
	}

	receivedMessage.New = true
	state.Response.AddMessage(receivedMessage)

	return nil
}

func (m *Messenger) HandleChatMessage(ctx context.Context, state *ReceivedMessageState, message *protobuf.ChatMessage, statusMessage *common.StatusMessage, fromArchive bool) error {
	state.CurrentMessageState.Message = message
	return m.handleChatMessage(ctx, state, fromArchive)
}

func (m *Messenger) HandleSyncSetting(ctx context.Context, messageState *ReceivedMessageState, message *protobuf.SyncSetting, statusMessage *common.StatusMessage) error {
	settingField, err := syncing.ExtractAndSaveSyncSetting(m.settings, m.logger, message)
	if err != nil {
		return err
	}

	if settingField == nil {
		return nil
	}

	switch message.GetType() {
	case protobuf.SyncSetting_DISPLAY_NAME:
		if newName := message.GetValueString(); newName != "" && m.account.Name != newName {
			m.account.Name = newName
			if err := m.multiAccounts.SaveAccount(*m.account); err != nil {
				return err
			}
		}
	case protobuf.SyncSetting_MNEMONIC_REMOVED:
		if message.GetValueBool() {
			if err := m.settings.DeleteMnemonic(); err != nil {
				return err
			}
			messageState.Response.AddSetting(&settings2.SyncSettingField{SettingField: settings2.Mnemonic})
		}
		return nil
	}
	messageState.Response.AddSetting(settingField)
	return nil
}

func (m *Messenger) HandleSyncAccountCustomizationColor(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncAccountCustomizationColor, statusMessage *common.StatusMessage) error {
	affected, err := m.multiAccounts.UpdateAccountCustomizationColor(message.GetKeyUid(), message.GetCustomizationColor(), message.GetUpdatedAt())
	if err != nil {
		return err
	}

	if affected > 0 {
		m.account.CustomizationColor = multiaccountscommon.CustomizationColor(message.GetCustomizationColor())
		state.Response.CustomizationColor = message.GetCustomizationColor()
	}
	return nil
}

func (m *Messenger) matchChatEntity(chatEntity ChatEntity, messageType protobuf.ApplicationMetadataMessage_Type) (*Chat, error) {
	if chatEntity.GetSigPubKey() == nil {
		m.logger.Error("public key can't be empty")
		return nil, errors.New("received a chatEntity with empty public key")
	}

	switch {
	case chatEntity.GetMessageType() == protobuf.MessageType_PUBLIC_GROUP:
		// For public messages, all outgoing and incoming messages have the same chatID
		// equal to a public chat name.
		chatID := chatEntity.GetChatId()
		chat, ok := m.allChats.Load(chatID)
		if !ok {
			return nil, errors.New("received a public chatEntity from non-existing chat")
		}
		if !chat.Public() && !chat.ProfileUpdates() && !chat.Timeline() {
			return nil, ErrMessageForWrongChatType
		}
		return chat, nil
	case chatEntity.GetMessageType() == protobuf.MessageType_ONE_TO_ONE && crypto.IsPubKeyEqual(chatEntity.GetSigPubKey(), &m.identity.PublicKey):
		// It's a private message coming from us so we rely on Message.ChatID
		// If chat does not exist, it should be created to support multidevice synchronization.
		chatID := chatEntity.GetChatId()
		chat, ok := m.allChats.Load(chatID)
		if !ok {
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

			chat = CreateOneToOneChat(chatID[:8], pubKey, m.getTimesource())
		}
		// if we are the sender, the chat must be active
		chat.Active = true
		return chat, nil
	case chatEntity.GetMessageType() == protobuf.MessageType_ONE_TO_ONE:
		// It's an incoming private chatEntity. ChatID is calculated from the signature.
		// If a chat does not exist, a new one is created and saved.
		chatID := contacts.ContactIDFromPublicKey(chatEntity.GetSigPubKey())
		chat, ok := m.allChats.Load(chatID)
		if !ok {
			// TODO: this should be a three-word name used in the mobile client
			chat = CreateOneToOneChat(chatID[:8], chatEntity.GetSigPubKey(), m.getTimesource())
			chat.Active = false
		}
		// We set the chat as inactive and will create a notification
		// if it's not coming from a contact
		contact, ok := m.allContacts.Load(chatID)
		chat.Active = chat.Active || (ok && contact.Added())
		return chat, nil
	case chatEntity.GetMessageType() == protobuf.MessageType_COMMUNITY_CHAT:
		chatID := chatEntity.GetChatId()
		chat, ok := m.allChats.Load(chatID)
		if !ok {
			return nil, errors.New("received community chat chatEntity for non-existing chat")
		}

		if chat.CommunityID == "" || chat.ChatType != ChatTypeCommunityChat {
			return nil, errors.New("not an community chat")
		}

		canPost, err := m.communitiesManager.CanPost(chatEntity.GetSigPubKey(), chat.CommunityID, chat.CommunityChatID(), messageType)
		if err != nil {
			return nil, err
		}

		if !canPost {
			return nil, errors.New("user can't post in community")
		}

		return chat, nil

	case chatEntity.GetMessageType() == protobuf.MessageType_PRIVATE_GROUP:
		// In the case of a group chatEntity, ChatID is the same for all messages belonging to a group.
		// It needs to be verified if the signature public key belongs to the chat.
		chatID := chatEntity.GetChatId()
		chat, ok := m.allChats.Load(chatID)
		if !ok {
			return nil, errors.New("received group chat chatEntity for non-existing chat")
		}

		senderKeyHex := contacts.ContactIDFromPublicKey(chatEntity.GetSigPubKey())
		myKeyHex := contacts.ContactIDFromPublicKey(&m.identity.PublicKey)
		senderIsMember := false
		iAmMember := false
		for _, member := range chat.Members {
			if member.ID == senderKeyHex {
				senderIsMember = true
			}
			if member.ID == myKeyHex {
				iAmMember = true
			}
		}

		if senderIsMember && iAmMember {
			return chat, nil
		}

		return nil, errors.New("did not find a matching group chat")
	default:
		return nil, errors.New("can not match a chat because there is no valid case")
	}
}

func (m *Messenger) messageExists(messageID string, existingMessagesMap map[string]bool) (bool, error) {
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

func (m *Messenger) HandleEmojiReaction(ctx context.Context, state *ReceivedMessageState, pbEmojiR *protobuf.EmojiReaction, statusMessage *common.StatusMessage) error {
	logger := m.logger.With(zap.String("site", "HandleEmojiReaction"))
	if err := ValidateReceivedEmojiReaction(pbEmojiR, state.Timesource.GetCurrentTime()); err != nil {
		logger.Error("invalid emoji reaction", zap.Error(err))
		return err
	}

	from := state.CurrentMessageState.Contact.ID

	// Backwards compatibility: old versions will not have the string Emoji, convert it
	if pbEmojiR.Emoji == "" {
		pbEmojiR.Emoji = ConvertEmojiIDToString(pbEmojiR.Type)
	}

	// Validate that the emoji is valid
	if !emojiRegex.MatchString(pbEmojiR.Emoji) {
		return errors.New("invalid emoji")
	}

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

	err = m.CheckMaxNumberOfEmojiReactionsPerMessage(emojiReaction.ChatId, emojiReaction.MessageId, pbEmojiR.Emoji)
	if err != nil {
		return err
	}

	chat, err := m.matchChatEntity(emojiReaction, protobuf.ApplicationMetadataMessage_EMOJI_REACTION)
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
	// TODO(samyoul) remove storing of an updated reference pointer?
	state.AllChats.Store(chat.ID, chat)

	// save emoji reaction
	err = m.persistence.SaveEmojiReaction(emojiReaction)
	if err != nil {
		return err
	}

	state.EmojiReactions[emojiReaction.ID()] = emojiReaction

	return nil
}

func (m *Messenger) HandleGroupChatInvitation(ctx context.Context, state *ReceivedMessageState, pbGHInvitations *protobuf.GroupChatInvitation, statusMessage *common.StatusMessage) error {
	allowed, err := m.isMessageAllowedFrom(state.CurrentMessageState.Contact.ID, nil)
	if err != nil {
		return err
	}

	if !allowed {
		return ErrMessageNotAllowed
	}
	logger := m.logger.With(zap.String("site", "HandleGroupChatInvitation"))
	if err := ValidateReceivedGroupChatInvitation(pbGHInvitations); err != nil {
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
		groupChatInvitation.From = types3.EncodeHex(crypto.FromECDSAPub(&m.identity.PublicKey))
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

func (m *Messenger) HandleContactCodeAdvertisement(ctx context.Context, state *ReceivedMessageState, cca *protobuf.ContactCodeAdvertisement, statusMessage *common.StatusMessage) error {
	publicKey := state.CurrentMessageState.PublicKey

	if m.pushNotificationClient != nil {
		if err := m.pushNotificationClient.HandleContactCodeAdvertisement(publicKey, cca); err != nil {
			m.logger.Warn("failed to handle ContactCodeAdvertisement push notification info", zap.Error(err))
		}
	}

	if cca.ChatIdentity == nil {
		return nil
	}
	return m.HandleChatIdentity(ctx, state, cca.ChatIdentity, nil)
}

// HandleChatIdentity handles an incoming protobuf.ChatIdentity
// extracts contact information stored in the protobuf and adds it to the user's contact for update.
func (m *Messenger) HandleChatIdentity(ctx context.Context, state *ReceivedMessageState, ci *protobuf.ChatIdentity, statusMessage *common.StatusMessage) error {
	s, err := m.settings.GetSettings()
	if err != nil {
		return err
	}

	contact := state.CurrentMessageState.Contact
	viewFromContacts := s.ProfilePicturesVisibility == settings2.ProfilePicturesVisibilityContactsOnly
	viewFromNoOne := s.ProfilePicturesVisibility == settings2.ProfilePicturesVisibilityNone

	m.logger.Debug("settings found",
		zap.Bool("viewFromContacts", viewFromContacts),
		zap.Bool("viewFromNoOne", viewFromNoOne),
	)

	// If we don't want to view profile images from anyone, don't process identity images.
	// We don't want to store the profile images of other users, even if we don't display images.
	inOurContacts, ok := m.allContacts.Load(state.CurrentMessageState.Contact.ID)

	isContact := ok && inOurContacts.Added()
	if viewFromNoOne && !isContact {
		return nil
	}

	// If there are no images attached to a ChatIdentity, check if message is allowed
	// Or if there are images and visibility is set to from contacts only, check if message is allowed
	// otherwise process the images without checking if the message is allowed
	if len(ci.Images) == 0 || (len(ci.Images) > 0 && (viewFromContacts)) {
		allowed, err := m.isMessageAllowedFrom(state.CurrentMessageState.Contact.ID, nil)
		if err != nil {
			return err
		}

		if !allowed {
			return ErrMessageNotAllowed
		}
	}

	err = DecryptIdentityImagesWithIdentityPrivateKey(ci.Images, m.identity, state.CurrentMessageState.PublicKey)
	if err != nil {
		return err
	}

	// Remove any images still encrypted after the decryption process
	for name, image := range ci.Images {
		if image.Encrypted {
			delete(ci.Images, name)
		}
	}

	if len(ci.Images) == 0 {
		contact.Images = nil
	}

	clockChanged, imagesChanged, err := m.persistence.UpdateContactChatIdentity(contact.ID, ci)
	if err != nil {
		return err
	}
	contactModified := false

	if imagesChanged {
		for imageType, image := range ci.Images {
			if contact.Images == nil {
				contact.Images = make(map[string]images.IdentityImage)
			}
			contact.Images[imageType] = images.IdentityImage{Name: imageType, Payload: image.Payload, Clock: ci.Clock}

		}
		if err = m.updateContactImagesURL(contact); err != nil {
			return err
		}

		contactModified = true
	}

	if clockChanged {
		if err = gocommon.ValidateDisplayName(&ci.DisplayName); err != nil {
			return err
		}

		if contact.DisplayName != ci.DisplayName && len(ci.DisplayName) != 0 {
			contact.DisplayName = ci.DisplayName
			contactModified = true
		}

		if customizationColor := multiaccountscommon.IDToColorFallbackToBlue(ci.CustomizationColor); contact.CustomizationColor != customizationColor {
			contact.CustomizationColor = customizationColor
			contactModified = true
		}

		if err = ValidateBio(&ci.Description); err != nil {
			return err
		}

		if contact.Bio != ci.Description {
			contact.Bio = ci.Description
			contactModified = true
		}

		if ci.ProfileShowcase != nil {
			err := m.BuildProfileShowcaseFromIdentity(state, ci.ProfileShowcase)
			if err != nil {
				return err
			}
			state.Response.AddUpdatedProfileShowcaseContactID(contact.ID)
		}
	}

	if contactModified {
		state.ModifiedContacts.Store(contact.ID, true)
		state.AllContacts.Store(contact.ID, contact)
	}

	return nil
}

func (m *Messenger) HandleAnonymousMetricBatch(ctx context.Context, state *ReceivedMessageState, amb *protobuf.AnonymousMetricBatch, statusMessage *common.StatusMessage) error {

	// TODO
	return nil
}

func (m *Messenger) checkForEdits(message *common.Message) error {
	// Check for any pending edit
	// If any pending edits are available and valid, apply them
	edits, err := m.persistence.GetEdits(message.ID, message.From)
	if err != nil {
		return err
	}

	if len(edits) == 0 {
		return nil
	}

	// Apply the first edit that is valid
	for _, e := range edits {
		if e.Clock >= message.Clock {
			// Update message and return it
			err := m.applyEditMessage(e.EditMessage, message)
			if err != nil {
				return err
			}
			return nil
		}
	}

	return nil
}

func (m *Messenger) getMessagesToCheckForDelete(message *common.Message) ([]*common.Message, error) {
	var messagesToCheck []*common.Message
	if message.ContentType == protobuf.ChatMessage_IMAGE {
		image := message.GetImage()
		if image != nil && image.AlbumId != "" {
			messagesInTheAlbum, err := m.persistence.albumMessages(message.ChatId, image.GetAlbumId())
			if err != nil {
				return nil, err
			}
			messagesToCheck = append(messagesToCheck, messagesInTheAlbum...)
		}
	}
	messagesToCheck = append(messagesToCheck, message)
	return messagesToCheck, nil
}

func (m *Messenger) checkForDeletes(message *common.Message) error {
	// Get all messages part of the album
	messagesToCheck, err := m.getMessagesToCheckForDelete(message)
	if err != nil {
		return err
	}

	var messageDeletes []*DeleteMessage
	applyDelete := false
	// Loop all messages part of the album, if one of them is marked as deleted, we delete them all
	for _, messageToCheck := range messagesToCheck {
		// Check for any pending deletes
		// If any pending deletes are available and valid, apply them
		messageDeletes, err = m.persistence.GetDeletes(messageToCheck.ID, messageToCheck.From)
		if err != nil {
			return err
		}

		if len(messageDeletes) == 0 {
			continue
		}
		// Once one messageDelete has been found, we apply it to all the images in the album
		applyDelete = true
		break
	}
	if applyDelete {
		for _, messageToCheck := range messagesToCheck {
			err := m.applyDeleteMessage(messageDeletes, messageToCheck)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Messenger) checkForDeleteForMes(message *common.Message) error {
	messagesToCheck, err := m.getMessagesToCheckForDelete(message)
	if err != nil {
		return err
	}

	var messageDeleteForMes []*protobuf.SyncDeleteForMeMessage
	applyDelete := false
	for _, messageToCheck := range messagesToCheck {
		if !applyDelete {
			// Check for any pending delete for mes
			// If any pending deletes are available and valid, apply them
			messageDeleteForMes, err = m.persistence.GetDeleteForMeMessagesByMessageID(messageToCheck.ID)
			if err != nil {
				return err
			}

			if len(messageDeleteForMes) == 0 {
				continue
			}
		}
		// Once one messageDeleteForMes has been found, we apply it to all the images in the album
		applyDelete = true

		err := m.applyDeleteForMeMessage(messageToCheck)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Messenger) isMessageAllowedFrom(publicKey string, chat *Chat) (bool, error) {
	onlyFromContacts, err := m.settings.GetMessagesFromContactsOnly()
	if err != nil {
		return false, err
	}

	if !onlyFromContacts {
		return true, nil
	}

	// if it's from us, it's allowed
	if m.myHexIdentity() == publicKey {
		return true, nil
	}

	// If the chat is public, we allow it
	if chat != nil && chat.Public() {
		return true, nil
	}

	contact, contactOk := m.allContacts.Load(publicKey)

	// If the chat is active, we allow it
	if chat != nil && chat.Active {
		if contactOk {
			// If the chat is active and it is a 1x1 chat, we need to make sure the contact is added and not removed
			return contact.Added(), nil
		}
		return true, nil
	}

	if !contactOk {
		// If it's not in contacts, we don't allow it
		return false, nil
	}

	// Otherwise we check if we added it
	return contact.Added(), nil
}

func (m *Messenger) updateUnviewedCounts(chat *Chat, message *common.Message) {
	chat.UnviewedMessagesCount++
	if message.Mentioned || message.Replied || chat.OneToOne() {
		chat.UnviewedMentionsCount++
	}
}

func (m *Messenger) resolveAccountOperability(syncAcc *protobuf.SyncAccount,
	syncKpMigratedToKeycard bool, dbKpMigratedToKeycard bool, accountReceivedFromLocalPairing bool) (accsmanagementtypes.AccountOperable, error) {
	if accountReceivedFromLocalPairing {
		return accsmanagementtypes.AccountOperable(syncAcc.Operable), nil
	}

	if syncKpMigratedToKeycard && m.account.KeyUID == syncAcc.KeyUid {
		return accsmanagementtypes.AccountFullyOperable, nil
	}

	accountsOperability := accsmanagementtypes.AccountNonOperable
	dbAccount, err := m.settings.GetAccountByAddress(types3.BytesToAddress(syncAcc.Address))
	if err != nil && err != accounts.ErrDbAccountNotFound {
		return accountsOperability, err
	}
	if dbAccount != nil {
		// We're here when we receive a keypair from the paired device which has just migrated from keycard to app.
		if !syncKpMigratedToKeycard && dbKpMigratedToKeycard {
			return accsmanagementtypes.AccountNonOperable, nil
		}
		return dbAccount.Operable, nil
	}

	if !syncKpMigratedToKeycard && dbKpMigratedToKeycard {
		// We're here when we receive a keypair from the paired device which was just converted from a cold wallet
		// to a regular keypair so we need to mark all accounts for this keypair non operable
		return accsmanagementtypes.AccountNonOperable, nil
	}

	if syncAcc.Chat || syncAcc.Wallet {
		accountsOperability = accsmanagementtypes.AccountFullyOperable
	} else {
		partiallyOrFullyOperable, err := m.settings.IsAnyAccountPartiallyOrFullyOperableForKeyUID(syncAcc.KeyUid)
		if err != nil {
			if err == accsmanagementtypes.ErrDbKeypairNotFound {
				return accsmanagementtypes.AccountNonOperable, nil
			}
			return accsmanagementtypes.AccountNonOperable, err
		}
		if partiallyOrFullyOperable {
			accountsOperability = accsmanagementtypes.AccountPartiallyOperable
		}
	}

	return accountsOperability, nil
}

func (m *Messenger) handleSyncTokenPreferences(message *protobuf.SyncTokenPreferences) ([]walletsettings.TokenPreferences, error) {
	if len(message.Preferences) == 0 {
		return nil, nil
	}

	dbLastUpdate, err := m.settings.GetClockOfLastTokenPreferencesChange()
	if err != nil {
		return nil, err
	}

	groupByCommunity, err := m.settings.GetTokenGroupByCommunity()
	if err != nil {
		return nil, err
	}

	// Since adding new token preferences updates `ClockOfLastTokenPreferencesChange` we should handle token preferences changes
	// even they are with the same clock, that ensures the correct order in case of syncing devices.
	if message.Clock < dbLastUpdate {
		return nil, ErrTryingToApplyOldTokenPreferences
	}

	var tokenPreferences []walletsettings.TokenPreferences
	for _, pref := range message.Preferences {
		tokenPref := walletsettings.TokenPreferences{
			Key:           pref.Key,
			Position:      int(pref.Position),
			GroupPosition: int(pref.GroupPosition),
			Visible:       pref.Visible,
			CommunityID:   pref.CommunityId,
		}
		tokenPreferences = append(tokenPreferences, tokenPref)
	}

	err = m.settings.UpdateTokenPreferences(tokenPreferences, groupByCommunity, message.Testnet, message.Clock)
	if err != nil {
		return nil, err
	}
	return tokenPreferences, nil
}

func (m *Messenger) handleSyncCollectiblePreferences(message *protobuf.SyncCollectiblePreferences) ([]walletsettings.CollectiblePreferences, error) {
	if len(message.Preferences) == 0 {
		return nil, nil
	}

	dbLastUpdate, err := m.settings.GetClockOfLastCollectiblePreferencesChange()
	if err != nil {
		return nil, err
	}

	groupByCommunity, err := m.settings.GetCollectibleGroupByCommunity()
	if err != nil {
		return nil, err
	}

	groupByCollection, err := m.settings.GetCollectibleGroupByCollection()
	if err != nil {
		return nil, err
	}

	// Since adding new collectible preferences updates `ClockOfLastCollectiblePreferencesChange` we should handle collectible
	// preferences changes even they are with the same clock, that ensures the correct order in case of syncing devices.
	if message.Clock < dbLastUpdate {
		return nil, ErrTryingToApplyOldCollectiblePreferences
	}

	var collectiblePreferences []walletsettings.CollectiblePreferences
	for _, pref := range message.Preferences {
		collectiblePref := walletsettings.CollectiblePreferences{
			Type:     walletsettings.CollectiblePreferencesType(pref.Type),
			Key:      pref.Key,
			Position: int(pref.Position),
			Visible:  pref.Visible,
		}
		collectiblePreferences = append(collectiblePreferences, collectiblePref)
	}

	err = m.settings.UpdateCollectiblePreferences(collectiblePreferences, groupByCommunity, groupByCollection, message.Testnet, message.Clock)
	if err != nil {
		return nil, err
	}
	return collectiblePreferences, nil
}

func (m *Messenger) handleSyncAccountsPositions(message *protobuf.SyncAccountsPositions) ([]*accsmanagementtypes.Account, error) {
	if len(message.Accounts) == 0 {
		return nil, nil
	}

	dbLastUpdate, err := m.settings.GetClockOfLastAccountsPositionChange()
	if err != nil {
		return nil, err
	}

	// Since adding new account updates `ClockOfLastAccountsPositionChange` we should handle account order changes
	// even they are with the same clock, that ensures the correct order in case of syncing devices.
	if message.Clock < dbLastUpdate {
		return nil, ErrTryingToApplyOldWalletAccountsOrder
	}

	var accs []*accsmanagementtypes.Account
	for _, sAcc := range message.Accounts {
		acc := &accsmanagementtypes.Account{
			Address:  types3.BytesToAddress(sAcc.Address),
			KeyUID:   sAcc.KeyUid,
			Position: sAcc.Position,
		}
		accs = append(accs, acc)
	}

	err = m.settings.SetWalletAccountsPositions(accs, message.Clock)
	if err != nil {
		return nil, err
	}

	return accs, nil
}

func (m *Messenger) handleProfileKeypairMigration(state *ReceivedMessageState, fromLocalPairing bool, message *protobuf.SyncKeypair) (migrationNeeded bool, err error) {
	if message == nil {
		err = errors.New("handleProfileKeypairMigration receive a nil message")
		return
	}

	if fromLocalPairing {
		return
	}

	if m.account.KeyUID != message.KeyUid {
		return
	}

	var dbKeypair *accsmanagementtypes.Keypair
	dbKeypair, err = m.settings.GetKeypairByKeyUID(message.KeyUid)
	if err != nil {
		return
	}

	if dbKeypair.Clock >= message.Clock {
		return
	}

	migrationNeeded = dbKeypair.MigratedToColdWallet() && message.ColdWallet == "" || // `true` if profile keypair cold wallet was removed on one of paired devices
		!dbKeypair.MigratedToColdWallet() && message.ColdWallet != "" // `true` if profile keypair was migrated to a cold wallet on one of paired devices
	err = m.settings.SaveSettingField(settings2.ProfileMigrationNeeded, migrationNeeded)
	if err != nil {
		return
	}

	state.Response.AddSetting(&settings2.SyncSettingField{SettingField: settings2.ProfileMigrationNeeded, Value: migrationNeeded})
	return
}

func (m *Messenger) handleSyncKeypair(message *protobuf.SyncKeypair, fromLocalPairing bool, applyColdWalletState bool, acNofificationCallback func() error) (*accsmanagementtypes.Keypair, error) {
	if message == nil {
		return nil, errors.New("handleSyncKeypair receive a nil message")
	}
	dbKeypair, err := m.settings.GetKeypairByKeyUID(message.KeyUid)
	if err != nil && err != accsmanagementtypes.ErrDbKeypairNotFound {
		return nil, err
	}

	kp := &accsmanagementtypes.Keypair{
		KeyUID:                  message.KeyUid,
		Name:                    message.Name,
		Type:                    accsmanagementtypes.KeypairType(message.Type),
		DerivedFrom:             message.DerivedFrom,
		LastUsedDerivationIndex: message.LastUsedDerivationIndex,
		SyncedFrom:              message.SyncedFrom,
		Clock:                   message.Clock,
		Removed:                 message.Removed,
		XPub:                    message.Xpub,
		ColdWallet:              accsmanagementtypes.ColdWalletType(message.ColdWallet),
	}

	oldAddresses := make(map[types3.Address]bool)

	if dbKeypair != nil {
		if dbKeypair.Clock >= kp.Clock {
			return nil, ErrTryingToStoreOldKeypair
		}
		// in case of keypair update, we need to keep `synced_from` field as it was when keypair was introduced to this device for the first time
		// but in case if keypair on this device came from the backup (e.g. device A recovered from waku, then device B paired with the device A
		// via local pairing, before device A made its keypairs fully operable) we need to update syncedFrom when user on this device when that
		// keypair becomes operable on any of other paired devices
		kp.SyncedFrom = dbKeypair.SyncedFrom
		for _, acc := range dbKeypair.Accounts {
			oldAddresses[acc.Address] = !acc.Removed
		}

		// Keep this device's signing method (auto-apply disabled, or a profile conversion
		// flow is pending): preserve the local cold wallet state instead of adopting the
		// wire value; all other fields still apply. For keypairs unknown locally there is
		// no state to preserve, the wire value is used.
		if !applyColdWalletState {
			kp.ColdWallet = dbKeypair.ColdWallet
		}
	}

	// Intentionally derived from the effective (possibly preserved) value, not the wire value,
	// so the operability transitions below stay consistent with what will be stored.
	syncKpMigratedToKeycard := kp.ColdWallet != ""

	for _, sAcc := range message.Accounts {
		accountOperability, err := m.resolveAccountOperability(sAcc,
			syncKpMigratedToKeycard,
			dbKeypair != nil && dbKeypair.MigratedToColdWallet(),
			fromLocalPairing)
		if err != nil {
			return nil, err
		}
		acc := syncing.MapSyncAccountToAccount(sAcc, accountOperability, accsmanagementtypes.GetAccountTypeForKeypairType(kp.Type))

		kp.Accounts = append(kp.Accounts, acc)
	}

	if !fromLocalPairing &&
		applyColdWalletState &&
		dbKeypair != nil &&
		!dbKeypair.MigratedToColdWallet() &&
		syncKpMigratedToKeycard {
		err = m.settings.MarkKeypairFullyOperable(dbKeypair.KeyUID, 0, false)
		if err != nil {
			return nil, err
		}
	}

	// deleting keypair will delete related keycards as well
	err = m.settings.RemoveKeypair(message.KeyUid, message.Clock)
	if err != nil && err != accsmanagementtypes.ErrDbKeypairNotFound {
		return nil, err
	}

	// if entire keypair was removed and keypair is already in db, there is no point to continue
	if kp.Removed && dbKeypair != nil {
		err = m.settings.ResolveAccountsPositions(message.Clock)
		if err != nil {
			return nil, err
		}
		return kp, nil
	}

	// save keypair first
	err = m.settings.SaveOrUpdateKeypair(kp)
	if err != nil {
		return nil, err
	}

	// then resolve accounts positions, cause some accounts might be removed
	err = m.settings.ResolveAccountsPositions(message.Clock)
	if err != nil {
		return nil, err
	}

	// if keypair is coming from paired device (means not from backup) and it's not among known, active keypairs,
	// we need to add an activity center notification
	if !kp.Removed && dbKeypair == nil {
		defer func() {
			err = acNofificationCallback()
		}()
	}

	// getting keypair form the db, cause keypair related accounts positions might be changed
	dbKeypair, err = m.settings.GetKeypairByKeyUID(message.KeyUid)
	if err != nil {
		return nil, err
	}

	if m.config.accountsPublisher != nil {
		addedAddresses := []gethcommon.Address{}
		removedAddresses := []gethcommon.Address{}
		if dbKeypair.Removed {
			for _, acc := range dbKeypair.Accounts {
				removedAddresses = append(removedAddresses, gethcommon.Address(acc.Address))
			}
		} else {
			for _, acc := range dbKeypair.Accounts {
				if acc.Chat {
					continue
				}
				// We call the signals only if there is a change in the account list
				// i.e. an account is unknown (not part of the Accounts list before)
				// or an account was removed and is now back (Removed flag changed)
				stillPresent, ok := oldAddresses[acc.Address]
				if acc.Removed && (!ok || stillPresent) {
					removedAddresses = append(removedAddresses, gethcommon.Address(acc.Address))
				} else if !ok || !stillPresent {
					addedAddresses = append(addedAddresses, gethcommon.Address(acc.Address))
				}
			}
		}

		if len(addedAddresses) > 0 {
			pubsub.Publish(m.config.accountsPublisher, accountsevent.AccountsAddedEvent{
				Accounts: addedAddresses,
			})
		}
		if len(removedAddresses) > 0 {
			pubsub.Publish(m.config.accountsPublisher, accountsevent.AccountsRemovedEvent{
				Accounts: removedAddresses,
			})
		}
	}

	return dbKeypair, nil
}

func (m *Messenger) HandleSyncAccountsPositions(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncAccountsPositions, statusMessage *common.StatusMessage) error {
	accs, err := m.handleSyncAccountsPositions(message)
	if err != nil {
		if err == ErrTryingToApplyOldWalletAccountsOrder ||
			err == accounts.ErrAccountWrongPosition ||
			err == accounts.ErrNotTheSameNumberOdAccountsToApplyReordering ||
			err == accounts.ErrNotTheSameAccountsToApplyReordering {
			m.logger.Warn("syncing accounts order issue", zap.Error(err))
			return nil
		}
		return err
	}

	state.Response.AccountsPositions = append(state.Response.AccountsPositions, accs...)

	return nil
}

func (m *Messenger) HandleSyncTokenPreferences(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncTokenPreferences, statusMessage *common.StatusMessage) error {
	tokenPreferences, err := m.handleSyncTokenPreferences(message)
	if err != nil {
		if err == ErrTryingToApplyOldTokenPreferences {
			m.logger.Warn("syncing token preferences issue", zap.Error(err))
			return nil
		}
		return err
	}

	state.Response.TokenPreferences = append(state.Response.TokenPreferences, tokenPreferences...)

	return nil
}

func (m *Messenger) HandleSyncCollectiblePreferences(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncCollectiblePreferences, statusMessage *common.StatusMessage) error {
	collectiblePreferences, err := m.handleSyncCollectiblePreferences(message)
	if err != nil {
		if err == ErrTryingToApplyOldCollectiblePreferences {
			m.logger.Warn("syncing collectible preferences issue", zap.Error(err))
			return nil
		}
		return err
	}

	state.Response.CollectiblePreferences = append(state.Response.CollectiblePreferences, collectiblePreferences...)

	return nil
}

func (m *Messenger) HandleSyncAccount(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncAccount, statusMessage *common.StatusMessage) error {
	acc, err := syncing.HandleSyncWatchOnlyAccount(m.settings, message, m.config.accountsPublisher)
	if err != nil {
		if err == syncing.ErrTryingToStoreOldWalletAccount {
			return nil
		}
		return err
	}

	state.Response.WatchOnlyAccounts = append(state.Response.WatchOnlyAccounts, acc)

	return nil
}

func (m *Messenger) HandleSyncKeypair(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncKeypair, statusMessage *common.StatusMessage) error {
	return m.handleSyncKeypairInternal(state, message, false)
}

func (m *Messenger) handleSyncKeypairInternal(state *ReceivedMessageState, message *protobuf.SyncKeypair, fromLocalPairing bool) error {
	if message == nil {
		return errors.New("handleSyncKeypairInternal receive a nil message")
	}

	if m.walletAPI != nil {
		err := m.walletAPI.SetPairingsJSONFileContent(message.KeycardPairings)
		if err != nil {
			return err
		}
	}

	// The auto-apply setting is device-local: when disabled, incoming syncs update keypair metadata only and never change this device's signing method (cold wallet state).
	// Local pairing bypasses it — the initial transfer must carry the full state.
	autoApply := true
	if !fromLocalPairing {
		var err error
		autoApply, err = m.settings.AutoApplyKeypairMigrations()
		if err != nil {
			return err
		}
	}

	applyColdWalletState := autoApply
	if autoApply {
		// check for the profile keypair migration first on paired device
		migrationNeeded, err := m.handleProfileKeypairMigration(state, fromLocalPairing, message)
		if err != nil {
			return err
		}

		// Even the migration is needed, the metadata (name, accounts, colors, ...) still applies below, but the local
		// cold wallet state is preserved until the user completes the migration flow
		if migrationNeeded {
			applyColdWalletState = false
		}
	}

	kp, err := m.handleSyncKeypair(message, fromLocalPairing, applyColdWalletState, func() error {
		return m.addNewKeypairAddedOnPairedDeviceACNotification(message.KeyUid, state.Response)
	})
	if err != nil {
		if err == ErrTryingToStoreOldKeypair {
			return nil
		}
		return err
	}

	state.Response.Keypairs = append(state.Response.Keypairs, kp)

	return nil
}

func (m *Messenger) HandleSyncContactRequestDecision(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncContactRequestDecision, statusMessage *common.StatusMessage) error {
	var err error
	var response *MessengerResponse

	if message.DecisionStatus == protobuf.SyncContactRequestDecision_ACCEPTED {
		response, err = m.updateAcceptedContactRequest(nil, message.RequestId, message.ContactId, true)
	} else {
		response, err = m.declineContactRequest(message.RequestId, message.ContactId)
	}
	if err != nil {
		return err
	}

	return state.Response.Merge(response)
}

func (m *Messenger) HandlePushNotificationRegistration(ctx context.Context, state *ReceivedMessageState, encryptedRegistration []byte, statusMessage *common.StatusMessage) error {
	if m.pushNotificationServer == nil {
		return nil
	}
	publicKey := state.CurrentMessageState.PublicKey

	return m.pushNotificationServer.HandlePushNotificationRegistration(publicKey, encryptedRegistration)
}

func (m *Messenger) HandlePushNotificationResponse(ctx context.Context, state *ReceivedMessageState, message *protobuf.PushNotificationResponse, statusMessage *common.StatusMessage) error {
	if m.pushNotificationClient == nil {
		return nil
	}
	publicKey := state.CurrentMessageState.PublicKey

	return m.pushNotificationClient.HandlePushNotificationResponse(publicKey, message)
}

func (m *Messenger) HandlePushNotificationRegistrationResponse(ctx context.Context, state *ReceivedMessageState, message *protobuf.PushNotificationRegistrationResponse, statusMessage *common.StatusMessage) error {
	if m.pushNotificationClient == nil {
		return nil
	}
	publicKey := state.CurrentMessageState.PublicKey

	return m.pushNotificationClient.HandlePushNotificationRegistrationResponse(publicKey, message)
}

func (m *Messenger) HandlePushNotificationQuery(ctx context.Context, state *ReceivedMessageState, message *protobuf.PushNotificationQuery, statusMessage *common.StatusMessage) error {
	if m.pushNotificationServer == nil {
		return nil
	}
	publicKey := state.CurrentMessageState.PublicKey

	return m.pushNotificationServer.HandlePushNotificationQuery(publicKey, statusMessage.ApplicationLayer.ID, message)
}

func (m *Messenger) HandlePushNotificationQueryResponse(ctx context.Context, state *ReceivedMessageState, message *protobuf.PushNotificationQueryResponse, statusMessage *common.StatusMessage) error {
	if m.pushNotificationClient == nil {
		return nil
	}
	publicKey := state.CurrentMessageState.PublicKey

	return m.pushNotificationClient.HandlePushNotificationQueryResponse(publicKey, message)
}

func (m *Messenger) HandlePushNotificationRequest(ctx context.Context, state *ReceivedMessageState, message *protobuf.PushNotificationRequest, statusMessage *common.StatusMessage) error {
	if m.pushNotificationServer == nil {
		return nil
	}
	publicKey := state.CurrentMessageState.PublicKey

	return m.pushNotificationServer.HandlePushNotificationRequest(publicKey, statusMessage.ApplicationLayer.ID, message)
}

func (m *Messenger) HandleCommunityDescription(ctx context.Context, state *ReceivedMessageState, message *protobuf.CommunityDescription, statusMessage *common.StatusMessage) error {
	err := m.handleCommunityDescription(state, state.CurrentMessageState.PublicKey, message, statusMessage.EncryptionLayer.Payload, nil)
	if err != nil {
		m.logger.Warn("failed to handle CommunityDescription", zap.Error(err))
		return err
	}
	return nil
}

func (m *Messenger) HandleSyncBookmark(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncBookmark, statusMessage *common.StatusMessage) error {
	bookmark := &browsers.Bookmark{
		URL:      message.Url,
		Name:     message.Name,
		ImageURL: message.ImageUrl,
		Removed:  message.Removed,
		Clock:    message.Clock,
	}
	state.AllBookmarks[message.Url] = bookmark
	return nil
}

func (m *Messenger) HandleSyncClearHistory(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncClearHistory, statusMessage *common.StatusMessage) error {
	chatID := message.ChatId
	existingChat, ok := state.AllChats.Load(chatID)
	if !ok {
		return ErrChatNotFound
	}

	if existingChat.DeletedAtClockValue >= message.ClearedAt {
		return nil
	}

	err := m.persistence.ClearHistoryFromSyncMessage(existingChat, message.ClearedAt)
	if err != nil {
		return err
	}

	if existingChat.Public() {
		err = m.messaging.ClearProcessedMessageIDsCache()
		if err != nil {
			return err
		}
	}

	state.AllChats.Store(chatID, existingChat)
	state.Response.AddChat(existingChat)
	state.Response.AddClearedHistory(&ClearedHistory{
		ClearedAt: message.ClearedAt,
		ChatID:    chatID,
	})
	return nil
}

func (m *Messenger) HandleSyncTrustedUser(ctx context.Context, state *ReceivedMessageState, message *protobuf.SyncTrustedUser, statusMessage *common.StatusMessage) error {
	updated, err := m.verificationDatabase.UpsertTrustStatus(message.Id, verification.TrustStatus(message.Status), message.Clock)
	if err != nil {
		return err
	}

	if updated {
		state.AllTrustStatus[message.Id] = verification.TrustStatus(message.Status)

		contact, ok := m.allContacts.Load(message.Id)
		if !ok {
			m.logger.Info("contact not found")
			return nil
		}

		contact.TrustStatus = verification.TrustStatus(message.Status)
		m.allContacts.Store(contact.ID, contact)
		state.ModifiedContacts.Store(contact.ID, true)
	}

	return nil
}

func (m *Messenger) HandleCommunityMessageArchiveLink(ctx context.Context, state *ReceivedMessageState, message *protobuf.CommunityMessageArchiveLink, statusMessage *common.StatusMessage) error {
	m.logger.Debug("[LogosStorage][HandleCommunityMessageArchiveLink] received CommunityMessageArchiveLink", zap.String("archiveLink", message.ArchiveLink))

	return m.HandleHistoryArchiveLinkMessage(state, state.CurrentMessageState.PublicKey, message.ArchiveLink, message.Clock)
}

func (m *Messenger) addNewKeypairAddedOnPairedDeviceACNotification(keyUID string, response *MessengerResponse) error {
	kp, err := m.settings.GetKeypairByKeyUID(keyUID)
	if err != nil {
		return err
	}

	notification := &ActivityCenterNotification{
		ID:        types3.FromHex(uuid.New().String()),
		Type:      ActivityCenterNotificationTypeNewKeypairAddedToPairedDevice,
		Timestamp: m.getTimesource().GetCurrentTime(),
		Read:      false,
		UpdatedAt: m.GetCurrentTimeInMillis(),
		Message: &common.Message{
			ChatMessage: &protobuf.ChatMessage{
				Text: kp.Name,
			},
			ID: kp.KeyUID,
		},
	}

	err = m.addActivityCenterNotification(response, notification, nil)
	if err != nil {
		m.logger.Warn("failed to create activity center notification", zap.Error(err))
		return err
	}
	return nil
}

func (m *Messenger) HandleSyncProfileShowcasePreferences(ctx context.Context, state *ReceivedMessageState, p *protobuf.SyncProfileShowcasePreferences, statusMessage *common.StatusMessage) error {
	_, err := m.saveProfileShowcasePreferencesProto(p, false)
	return err
}
