package statusproto

import (
	protocol "github.com/status-im/status-protocol-go/v1"
)

func newProtocolGroupFromChat(chat *Chat) (*protocol.Group, error) {
	return protocol.NewGroup(chat.ID, chatToFlattenMembershipUpdate(chat))
}

func chatToFlattenMembershipUpdate(chat *Chat) []protocol.MembershipUpdateFlat {
	result := make([]protocol.MembershipUpdateFlat, len(chat.MembershipUpdates))
	for idx, update := range chat.MembershipUpdates {
		result[idx] = protocol.MembershipUpdateFlat{
			ChatID:    chat.ID,
			From:      update.From,
			Signature: update.Signature,
			MembershipUpdateEvent: protocol.MembershipUpdateEvent{
				Name:       update.Name,
				Type:       update.Type,
				ClockValue: int64(update.ClockValue), // TODO: remove type difference
				Member:     update.Member,
				Members:    update.Members,
			},
		}
	}
	return result
}
