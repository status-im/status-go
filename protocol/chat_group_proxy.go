package protocol

import (
	v1protocol "github.com/status-im/status-go/v10/protocol/v1"
)

func newProtocolGroupFromChat(chat *Chat) (*v1protocol.Group, error) {
	return v1protocol.NewGroupWithEvents(chat.ID, chat.MembershipUpdates)
}
