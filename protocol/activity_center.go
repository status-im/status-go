package protocol

import (
	"crypto/ecdsa"
	"errors"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
)

// The activity center is a place where we store incoming notifications before
// they are shown to the users as new chats, in order to mitigate the impact of spam
// on the messenger

type ActivityCenterType int

const (
	ActivityCenterNotificationNoType ActivityCenterType = iota
	ActivityCenterNotificationTypeNewOneToOne
	ActivityCenterNotificationTypeNewPrivateGroupChat
	ActivityCenterNotificationTypeMention
	ActivityCenterNotificationTypeReply
        ActivityCenterNotificationTypeContactRequest
        ActivityCenterNotificationTypeContactRequestRetracted
)

var ErrInvalidActivityCenterNotification = errors.New("invalid activity center notification")

type ActivityCenterNotification struct {
	ID           types.HexBytes     `json:"id"`
	ChatID       string             `json:"chatId"`
	Name         string             `json:"name"`
	Author       string             `json:"author"`
	Type         ActivityCenterType `json:"type"`
	LastMessage  *common.Message    `json:"lastMessage"`
	Message      *common.Message    `json:"message"`
	ReplyMessage *common.Message    `json:"replyMessage"`
	Timestamp    uint64             `json:"timestamp"`
	Read         bool               `json:"read"`
	Dismissed    bool               `json:"dismissed"`
	Accepted     bool               `json:"accepted"`
}

type ActivityCenterPaginationResponse struct {
	Cursor        string                        `json:"cursor"`
	Notifications []*ActivityCenterNotification `json:"notifications"`
}

func (n *ActivityCenterNotification) Valid() error {
	if len(n.ID) == 0 || n.Type == 0 || n.Timestamp == 0 {
		return ErrInvalidActivityCenterNotification
	}
	return nil

}

func showMentionOrReplyActivityCenterNotification(publicKey ecdsa.PublicKey, message *common.Message, chat *Chat, responseTo *common.Message) (bool, ActivityCenterType) {
	if chat == nil || !chat.Active || (!chat.CommunityChat() && !chat.PrivateGroupChat()) {
		return false, ActivityCenterNotificationNoType
	}

	if message.Mentioned {
		return true, ActivityCenterNotificationTypeMention
	}

	publicKeyString := common.PubkeyToHex(&publicKey)
	if responseTo != nil && responseTo.From == publicKeyString {
		return true, ActivityCenterNotificationTypeReply
	}

	return false, ActivityCenterNotificationNoType
}
