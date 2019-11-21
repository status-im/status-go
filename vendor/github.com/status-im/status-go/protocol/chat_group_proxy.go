package protocol

import (
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

func newProtocolGroupFromChat(chat *Chat) (*v1protocol.Group, error) {
	return v1protocol.NewGroup(chat.ID, chatToFlattenMembershipUpdate(chat))
}

func chatToFlattenMembershipUpdate(chat *Chat) []v1protocol.MembershipUpdateFlat {
	result := make([]v1protocol.MembershipUpdateFlat, len(chat.MembershipUpdates))
	for idx, update := range chat.MembershipUpdates {
		result[idx] = v1protocol.MembershipUpdateFlat{
			ChatID:    chat.ID,
			From:      update.From,
			Signature: update.Signature,
			MembershipUpdateEvent: v1protocol.MembershipUpdateEvent{
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
