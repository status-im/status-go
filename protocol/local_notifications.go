package protocol

import (
	"crypto/ecdsa"
	"encoding/json"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
	localnotifications "github.com/status-im/status-go/services/local-notifications"
)

type NotificationBody struct {
	Message   *common.Message        `json:"message"`
	Contact   *Contact               `json:"contact"`
	Chat      *Chat                  `json:"chat"`
	Community *communities.Community `json:"community"`
}

func showMessageNotification(publicKey ecdsa.PublicKey, message *common.Message, chat *Chat, responseTo *common.Message) bool {
	if chat != nil && (chat.OneToOne() || chat.PrivateGroupChat()) {
		return true
	}

	publicKeyString := common.PubkeyToHex(&publicKey)
	mentioned := false
	for _, mention := range message.Mentions {
		if publicKeyString == mention {
			mentioned = true
		}
	}

	if mentioned {
		return true
	}

	if responseTo != nil {
		return responseTo.From == publicKeyString
	}

	return false
}

func (n NotificationBody) MarshalJSON() ([]byte, error) {
	type Alias NotificationBody
	item := struct{ *Alias }{Alias: (*Alias)(&n)}
	return json.Marshal(item)
}

func NewMessageNotification(id string, message *common.Message, chat *Chat, contact *Contact, contacts map[string]*Contact) (*localnotifications.Notification, error) {
	body := &NotificationBody{
		Message: message,
		Chat:    chat,
		Contact: contact,
	}

	return body.toMessageNotification(id, contacts)
}

func NewCommunityRequestToJoinNotification(id string, community *communities.Community, contact *Contact) *localnotifications.Notification {
	body := &NotificationBody{
		Community: community,
		Contact:   contact,
	}

	return body.toCommunityRequestToJoinNotification(id)
}

func (n NotificationBody) toMessageNotification(id string, contacts map[string]*Contact) (*localnotifications.Notification, error) {
	var title string
	if n.Chat.PrivateGroupChat() || n.Chat.Public() || n.Chat.CommunityChat() {
		title = n.Chat.Name
	} else if n.Chat.OneToOne() {
		title = n.Contact.CanonicalName()

	}

	canonicalNames := make(map[string]string)
	for _, id := range n.Message.Mentions {
		contact, ok := contacts[id]
		if !ok {
			var err error
			contact, err = buildContactFromPkString(id)
			if err != nil {
				return nil, err
			}
		}
		canonicalNames[id] = contact.CanonicalName()
	}

	simplifiedText, err := n.Message.GetSimplifiedText(canonicalNames)
	if err != nil {
		return nil, err
	}

	return &localnotifications.Notification{
		Body:                n,
		ID:                  gethcommon.HexToHash(id),
		BodyType:            localnotifications.TypeMessage,
		Category:            localnotifications.CategoryMessage,
		Deeplink:            n.Chat.DeepLink(),
		Title:               title,
		Message:             simplifiedText,
		IsConversation:      true,
		IsGroupConversation: true,
		Author: localnotifications.NotificationAuthor{
			Name: n.Contact.CanonicalName(),
			Icon: n.Contact.CanonicalImage(),
			ID:   n.Contact.ID,
		},
		Timestamp:      n.Message.WhisperTimestamp,
		ConversationID: n.Chat.ID,
		Image:          "",
	}, nil
}

func (n NotificationBody) toCommunityRequestToJoinNotification(id string) *localnotifications.Notification {
	return &localnotifications.Notification{
		ID:       gethcommon.HexToHash(id),
		Body:     n,
		Title:    n.Contact.CanonicalName() + " wants to join " + n.Community.Name(),
		Message:  n.Contact.CanonicalName() + " wants to join  message " + n.Community.Name(),
		BodyType: localnotifications.TypeMessage,
		Category: localnotifications.CategoryCommunityRequestToJoin,
		Deeplink: "status-im://cr/" + n.Community.IDString(),
		Image:    "",
	}
}
