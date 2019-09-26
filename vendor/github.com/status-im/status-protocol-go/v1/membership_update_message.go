package statusproto

import "bytes"

// MembershipUpdateMessage is a message used to propagate information
// about group membership changes.
// For more information, see https://github.com/status-im/specs/blob/master/status-group-chats-spec.md.
type MembershipUpdateMessage struct {
	ChatID  string             `json:"chatId"` // UUID concatenated with hex-encoded public key of the creator for the chat
	Updates []MembershipUpdate `json:"updates"`
	Message *Message           `json:"message"` // optional message
}

type MembershipUpdate struct {
	ChatID    string                  `json:"chatId"`
	From      string                  `json:"from"`
	Signature string                  `json:"signature"`
	Events    []MembershipUpdateEvent `json:"events"`
}

type MembershipUpdateEvent struct {
	Type       string   `json:"type"`
	ClockValue int64    `json:"clockValue"`
	Member     string   `json:"member,omitempty"`  // in "member-joined", "member-removed" and "admin-removed" events
	Members    []string `json:"members,omitempty"` // in "members-added" and "admins-added" events
	Name       string   `json:"name,omitempty"`    // name of the group chat
}

// EncodeMembershipUpdateMessage encodes a MembershipUpdateMessage using Transit serialization.
func EncodeMembershipUpdateMessage(value MembershipUpdateMessage) ([]byte, error) {
	var buf bytes.Buffer
	encoder := NewMessageEncoder(&buf)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
