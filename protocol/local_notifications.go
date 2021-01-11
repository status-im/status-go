package protocol

import "github.com/status-im/status-go/protocol/common"

type MessageNotificationBody struct {
	Message *common.Message
	Contact *Contact
	Chat    *Chat
}
