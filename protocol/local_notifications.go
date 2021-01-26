package protocol

import "github.com/status-im/status-go/protocol/common"

type MessageNotificationBody struct {
	Message *common.Message `json:"message"`
	Contact *Contact        `json:"contact"`
	Chat    *Chat           `json:"chat"`
}
