package protocol

import (
	"crypto/ecdsa"
	"encoding/json"
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/internal/images"
	"github.com/status-im/status-go/internal/protocol/common"
	"github.com/status-im/status-go/internal/protocol/communities"
	"github.com/status-im/status-go/internal/protocol/contacts"
	"github.com/status-im/status-go/internal/protocol/identity/identicon"
	localnotifications "github.com/status-im/status-go/pkg/services/local-notifications"
)

// Notification setting values (must match settings_notifications constants)
const (
	notifValueSendAlerts = "SendAlerts"
	notifValueTurnOff    = "TurnOff"
)

// MessagePreview values (must match Constants.settingsSection.notificationsBubble)
const (
	messagePreviewAnonymous      = 0
	messagePreviewNameOnly       = 1
	messagePreviewNameAndMessage = 2
)

const messagePreviewDefaultMessage = "You have a new message"
const messagePreviewAnonymousTitle = "Status"

// communityIconDataURI returns a data URI for the community's avatar (thumbnail or banner).
func communityIconDataURI(c *communities.Community) string {
	if c == nil {
		return ""
	}
	imgs := c.Images()
	if img, ok := imgs[images.SmallDimName]; ok && img != nil && len(img.Payload) > 0 {
		uri, err := images.GetPayloadDataURI(img.Payload)
		if err == nil {
			return uri
		}
	}
	if img, ok := imgs[images.BannerIdentityName]; ok && img != nil && len(img.Payload) > 0 {
		uri, err := images.GetPayloadDataURI(img.Payload)
		if err == nil {
			return uri
		}
	}
	return ""
}

// authorIconDataURI returns a data URI for the notification author (sender) avatar.
// Uses contact's profile image or identicon fallback so notifications always have an icon.
// Profile picture visibility is always treated as Everyone for local notifications—the
// notification is private to the recipient who has already opted into receiving it.
func authorIconDataURI(contact *contacts.Contact) string {
	if contact == nil {
		return ""
	}
	// Always prefer profile image when available; notification recipient already opted into receiving it
	icon := contact.CanonicalImage(settings.ProfilePicturesVisibilityEveryone)
	if icon != "" {
		return icon
	}
	if contact.ID == "" {
		return ""
	}
	dataURI, err := identicon.GenerateBase64(contact.ID)
	if err != nil {
		return ""
	}
	return dataURI
}

// chatIconDataURI returns a data URI for the chat/group avatar.
func chatIconDataURI(chat *Chat) string {
	if chat == nil {
		return ""
	}
	if chat.Base64Image != "" {
		if strings.HasPrefix(chat.Base64Image, "data:") {
			return chat.Base64Image
		}
		return "data:image/png;base64," + chat.Base64Image
	}
	if chat.Identicon != "" {
		if strings.HasPrefix(chat.Identicon, "data:") {
			return chat.Identicon
		}
		return "data:image/png;base64," + chat.Identicon
	}
	return ""
}

// applyMessagePreview returns displayTitle and displayMessage for lock screen/OS use based on preview mode.
// 0=Anonymous (Status, You have a new message), 1=Name only (keep title, generic message), 2=Full (unchanged).
func applyMessagePreview(title, message string, messagePreview int) (displayTitle, displayMessage string) {
	switch messagePreview {
	case messagePreviewAnonymous:
		return messagePreviewAnonymousTitle, messagePreviewDefaultMessage
	case messagePreviewNameOnly:
		return title, messagePreviewDefaultMessage
	default:
		return title, message
	}
}

// getMessagePreviewText returns a simplified, notification-safe preview of the message text.
// Uses GetSimplifiedText when possible, falls back to raw Text. Returns empty string if message is nil or extraction fails.
func getMessagePreviewText(m *common.Message, canonicalNames map[string]string) string {
	if m == nil {
		return ""
	}
	text, err := m.GetSimplifiedText("", canonicalNames)
	if err != nil {
		return strings.TrimSpace(m.Text)
	}
	return strings.TrimSpace(text)
}

// timestampOrZero returns the message's WhisperTimestamp, or 0 if the message is nil.
func timestampOrZero(m *common.Message) uint64 {
	if m == nil {
		return 0
	}
	return m.WhisperTimestamp
}

// applyAuthorPrivacy clears identity-revealing fields from a notification when the privacy
// level is anonymous. In anonymous mode the OS notification must not expose who sent
// the message: no sender name, no sender/community/chat icon.
func applyAuthorPrivacy(notif *localnotifications.Notification, messagePreview int) {
	if messagePreview == messagePreviewAnonymous {
		notif.Author = localnotifications.NotificationAuthor{}
		notif.CommunityIcon = ""
		notif.ChatIcon = ""
	}
}

// NotificationSettingsProvider allows querying notification preferences for local notifications.
// Implemented by *accounts.Database which embeds *NotificationsSettings.
type NotificationSettingsProvider interface {
	GetOneToOneChats() (string, error)
	GetGroupChats() (string, error)
	GetPersonalMentions() (string, error)
	GetGlobalMentions() (string, error)
	GetAllMessages() (string, error)
	GetMessagePreview() (int, error)
	GetContactRequests() (string, error)
	HasExemption(id string) (bool, error)
	GetExMuteAllMessages(id string) (bool, error)
	GetExPersonalMentions(id string) (string, error)
	GetExGlobalMentions(id string) (string, error)
	GetExOtherMessages(id string) (string, error)
}

func isNotificationAllowed(value string) bool {
	return value != notifValueTurnOff
}

type NotificationBody struct {
	Message   *common.Message        `json:"message"`
	Contact   *contacts.Contact      `json:"contact"`
	Chat      *Chat                  `json:"chat"`
	Community *communities.Community `json:"community"`
}

func showMessageNotification(settings NotificationSettingsProvider, publicKey ecdsa.PublicKey, message *common.Message, chat *Chat, responseTo *common.Message) bool {
	if message != nil && message.From == crypto.PubkeyToHex(&publicKey) {
		return false
	}

	// For public/community chats, skip inactive (soft-deleted). For 1:1 and group,
	// still notify — e.g. contact requests or new messages can arrive when Active=false.
	if chat != nil && !chat.Active && !chat.OneToOne() && !chat.PrivateGroupChat() {
		return false
	}

	if settings == nil {
		// Fallback when settings unavailable: legacy behavior
		if chat != nil && (chat.OneToOne() || chat.PrivateGroupChat()) {
			return true
		}
		if message.Mentioned {
			return true
		}
		if responseTo != nil {
			return responseTo.From == crypto.PubkeyToHex(&publicKey)
		}
		return false
	}

	chatID := chat.ID
	if chatID == "" {
		chatID = chat.CommunityID
	}

	// 1. Check per-chat/community exemptions first
	exMuteAll, err := settings.GetExMuteAllMessages(chatID)
	if err == nil && exMuteAll {
		return false
	}

	// Classify message: is it a reply to own, a mention, or other?
	isReplyToOwn := responseTo != nil && responseTo.From == crypto.PubkeyToHex(&publicKey)

	// 2. One-to-one chats
	if chat.OneToOne() {
		if message.ContactRequestState == common.ContactRequestStatePending {
			val, err := settings.GetContactRequests()
			if err == nil && !isNotificationAllowed(val) {
				return false
			}
			return true
		}

		val, err := settings.GetOneToOneChats()
		if err == nil && !isNotificationAllowed(val) {
			return false
		}
		return true
	}

	// 3. Private group chats
	if chat.PrivateGroupChat() {
		val, err := settings.GetGroupChats()
		if err == nil && !isNotificationAllowed(val) {
			return false
		}
		return true
	}

	// 4. Public / community chats: check mentions, reply, or all messages
	// Use exemption values when HasExemption; otherwise use global settings.
	hasEx, _ := settings.HasExemption(chatID)

	if message.Mentioned {
		personalVal, _ := settings.GetPersonalMentions()
		globalVal, _ := settings.GetGlobalMentions()
		effPersonal, effGlobal := personalVal, globalVal
		if hasEx {
			exP, _ := settings.GetExPersonalMentions(chatID)
			exG, _ := settings.GetExGlobalMentions(chatID)
			effPersonal, effGlobal = exP, exG
		}
		return isNotificationAllowed(effPersonal) || isNotificationAllowed(effGlobal)
	}

	if isReplyToOwn {
		val, _ := settings.GetAllMessages()
		eff := val
		if hasEx {
			exOther, _ := settings.GetExOtherMessages(chatID)
			eff = exOther
		}
		return isNotificationAllowed(eff)
	}

	// 5. Other messages in public/community chats
	val, _ := settings.GetAllMessages()
	eff := val
	if hasEx {
		exOther, _ := settings.GetExOtherMessages(chatID)
		eff = exOther
	}
	return isNotificationAllowed(eff)
}

func (n NotificationBody) MarshalJSON() ([]byte, error) {
	type Alias NotificationBody
	item := struct{ *Alias }{Alias: (*Alias)(&n)}
	return json.Marshal(item)
}

func NewMessageNotification(id string, message *common.Message, chat *Chat, contact *contacts.Contact, community *communities.Community, resolvePrimaryName func(string) (string, error), profilePicturesVisibility int, messagePreview int) (*localnotifications.Notification, error) {
	body := &NotificationBody{
		Message:   message,
		Chat:      chat,
		Contact:   contact,
		Community: community,
	}

	// Contact requests in 1:1 chats should use a dedicated notification type with Accept/Reject actions
	if chat.OneToOne() && message.ContactRequestState == common.ContactRequestStatePending {
		return body.toContactRequestNotification(id, profilePicturesVisibility, messagePreview)
	}

	return body.toMessageNotification(id, resolvePrimaryName, profilePicturesVisibility, messagePreview)
}

func DeletedMessageNotification(id string, chat *Chat) *localnotifications.Notification {
	return &localnotifications.Notification{
		BodyType:       localnotifications.TypeMessage,
		ID:             gethcommon.HexToHash(id),
		IsConversation: true,
		ConversationID: chat.ID,
		Deeplink:       chat.DeepLink(),
		Deleted:        true,
	}
}

func NewCommunityRequestToJoinNotification(id string, community *communities.Community, contact *contacts.Contact, profilePicturesVisibility int, messagePreview int) *localnotifications.Notification {
	body := &NotificationBody{
		Community: community,
		Contact:   contact,
	}

	return body.toCommunityRequestToJoinNotification(id, profilePicturesVisibility, messagePreview)
}

// NewOutgoingMessageNotification creates a local notification for a message sent by the current user.
func NewOutgoingMessageNotification(id string, message *common.Message, chat *Chat, community *communities.Community, authorName string, authorIcon string, authorID string) *localnotifications.Notification {
	title := chat.Name
	simplifiedText := getMessagePreviewText(message, nil)
	notif := &localnotifications.Notification{
		ID:                  gethcommon.HexToHash(id),
		BodyType:            localnotifications.TypeMessage,
		Category:            localnotifications.CategoryMessage,
		Deeplink:            chat.DeepLink(),
		Title:               title,
		Message:             simplifiedText,
		DisplayTitle:        title,
		DisplayMessage:      simplifiedText,
		IsConversation:      true,
		IsGroupConversation: chat.PrivateGroupChat() || chat.Public() || chat.CommunityChat(),
		Author: localnotifications.NotificationAuthor{
			Name: authorName,
			Icon: authorIcon,
			ID:   authorID,
		},
		Timestamp:      message.WhisperTimestamp,
		ConversationID: chat.ID,
		Image:          "",
		IsFromMe:       true,
	}
	if chat.CommunityChat() && community != nil {
		notif.CommunityIcon = communityIconDataURI(community)
	}
	if chat.PrivateGroupChat() {
		notif.ChatIcon = chatIconDataURI(chat)
	}
	return notif
}

func NewPrivateGroupInviteNotification(id string, chat *Chat, contact *contacts.Contact, profilePicturesVisibility int, messagePreview int) *localnotifications.Notification {
	body := &NotificationBody{
		Chat:    chat,
		Contact: contact,
	}

	return body.toPrivateGroupInviteNotification(id, profilePicturesVisibility, messagePreview)
}

func (n NotificationBody) toMessageNotification(id string, resolvePrimaryName func(string) (string, error), profilePicturesVisibility int, messagePreview int) (*localnotifications.Notification, error) {
	var title string
	if n.Chat.PrivateGroupChat() || n.Chat.Public() || n.Chat.CommunityChat() {
		title = n.Chat.Name
	} else if n.Chat.OneToOne() {
		title = n.Contact.PrimaryName()

	}

	canonicalNames := make(map[string]string)
	for _, mentionID := range n.Message.Mentions {
		name, err := resolvePrimaryName(mentionID)
		if err != nil {
			canonicalNames[mentionID] = mentionID
		} else {
			canonicalNames[mentionID] = name
		}
	}

	// We don't pass idenity as only interested in the simplified text
	simplifiedText, err := n.Message.GetSimplifiedText("", canonicalNames)
	if err != nil {
		return nil, err
	}

	displayTitle, displayMessage := applyMessagePreview(title, simplifiedText, messagePreview)

	notif := &localnotifications.Notification{
		Body:                n,
		ID:                  gethcommon.HexToHash(id),
		BodyType:            localnotifications.TypeMessage,
		Category:            localnotifications.CategoryMessage,
		Deeplink:            n.Chat.DeepLink(),
		Title:               title,
		Message:             simplifiedText,
		DisplayTitle:        displayTitle,
		DisplayMessage:      displayMessage,
		IsConversation:      true,
		IsGroupConversation: n.Chat.PrivateGroupChat() || n.Chat.Public() || n.Chat.CommunityChat(),
		Author: localnotifications.NotificationAuthor{
			Name: n.Contact.PrimaryName(),
			Icon: authorIconDataURI(n.Contact),
			ID:   n.Contact.ID,
		},
		Timestamp:      n.Message.WhisperTimestamp,
		ConversationID: n.Chat.ID,
		Image:          "",
	}

	// Community chat: community icon
	if n.Chat.CommunityChat() && n.Community != nil {
		notif.CommunityIcon = communityIconDataURI(n.Community)
	}
	// Group chat: chat/group icon
	if n.Chat.PrivateGroupChat() {
		notif.ChatIcon = chatIconDataURI(n.Chat)
	}

	applyAuthorPrivacy(notif, messagePreview)
	return notif, nil
}

func (n NotificationBody) toContactRequestNotification(id string, profilePicturesVisibility int, messagePreview int) (*localnotifications.Notification, error) {
	title := n.Contact.PrimaryName() + " wants to connect"
	message := getMessagePreviewText(n.Message, nil)
	if message == "" {
		message = n.Contact.PrimaryName() + " sent you a contact request"
	}
	displayTitle, displayMessage := applyMessagePreview(title, message, messagePreview)
	notif := &localnotifications.Notification{
		ID:             gethcommon.HexToHash(id),
		Body:           n,
		Title:          title,
		Message:        message,
		DisplayTitle:   displayTitle,
		DisplayMessage: displayMessage,
		BodyType:       localnotifications.TypeMessage,
		Category:       localnotifications.CategoryContactRequest,
		Deeplink:       n.Chat.DeepLink(),
		Author: localnotifications.NotificationAuthor{
			Name: n.Contact.PrimaryName(),
			Icon: authorIconDataURI(n.Contact),
			ID:   n.Contact.ID,
		},
		ConversationID: n.Chat.ID,
		Timestamp:      timestampOrZero(n.Message),
		IsConversation: true,
		Image:          "",
	}
	applyAuthorPrivacy(notif, messagePreview)
	return notif, nil
}

func (n NotificationBody) toPrivateGroupInviteNotification(id string, profilePicturesVisibility int, messagePreview int) *localnotifications.Notification {
	title := n.Contact.PrimaryName() + " invited you to " + n.Chat.Name
	message := getMessagePreviewText(n.Message, nil)
	if message == "" {
		message = n.Contact.PrimaryName() + " wants you to join group " + n.Chat.Name
	}
	displayTitle, displayMessage := applyMessagePreview(title, message, messagePreview)
	notif := &localnotifications.Notification{
		ID:             gethcommon.HexToHash(id),
		Body:           n,
		Title:          title,
		Message:        message,
		DisplayTitle:   displayTitle,
		DisplayMessage: displayMessage,
		BodyType:       localnotifications.TypeMessage,
		Category:       localnotifications.CategoryGroupInvite,
		Deeplink:       n.Chat.DeepLink(),
		Author: localnotifications.NotificationAuthor{
			Name: n.Contact.PrimaryName(),
			Icon: authorIconDataURI(n.Contact),
			ID:   n.Contact.ID,
		},
		ChatIcon: chatIconDataURI(n.Chat),
		Image:    "",
	}
	applyAuthorPrivacy(notif, messagePreview)
	return notif
}

func (n NotificationBody) toCommunityRequestToJoinNotification(id string, profilePicturesVisibility int, messagePreview int) *localnotifications.Notification {
	title := n.Contact.PrimaryName() + " wants to join " + n.Community.Name()
	message := getMessagePreviewText(n.Message, nil)
	displayTitle, displayMessage := applyMessagePreview(title, message, messagePreview)
	notif := &localnotifications.Notification{
		ID:             gethcommon.HexToHash(id),
		Body:           n,
		Title:          title,
		Message:        message,
		DisplayTitle:   displayTitle,
		DisplayMessage: displayMessage,
		BodyType:       localnotifications.TypeMessage,
		Category:       localnotifications.CategoryCommunityRequestToJoin,
		Deeplink:       "status-app://cr/" + n.Community.IDString(),
		Author: localnotifications.NotificationAuthor{
			Name: n.Contact.PrimaryName(),
			Icon: authorIconDataURI(n.Contact),
			ID:   n.Contact.ID,
		},
		CommunityIcon: communityIconDataURI(n.Community),
		Image:         "",
	}
	applyAuthorPrivacy(notif, messagePreview)
	return notif
}
