package statusproto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"reflect"
	"sort"

	"github.com/status-im/status-protocol-go/crypto"
)

const (
	MembershipUpdateChatCreated   = "chat-created"
	MembershipUpdateNameChanged   = "name-changed"
	MembershipUpdateMembersAdded  = "members-added"
	MembershipUpdateMemberJoined  = "member-joined"
	MembershipUpdateMemberRemoved = "member-removed"
	MembershipUpdateAdminsAdded   = "admins-added"
	MembershipUpdateAdminRemoved  = "admin-removed"
)

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

// Sign creates a signature from MembershipUpdateEvents
// and updates MembershipUpdate's signature.
// It follows the algorithm describe in the spec:
// https://github.com/status-im/specs/blob/master/status-group-chats-spec.md#signature.
func (u *MembershipUpdate) Sign(identity *ecdsa.PrivateKey) error {
	sort.Slice(u.Events, func(i, j int) bool {
		return u.Events[i].ClockValue < u.Events[j].ClockValue
	})
	tuples := make([]interface{}, len(u.Events))
	for idx, event := range u.Events {
		tuples[idx] = tupleMembershipUpdateEvent(event)
	}
	structureToSign := []interface{}{
		tuples,
		u.ChatID,
	}
	data, err := json.Marshal(structureToSign)
	if err != nil {
		return err
	}
	signature, err := crypto.SignBytesAsHex(data, identity)
	if err != nil {
		return err
	}
	u.Signature = signature
	return nil
}

type MembershipUpdateEvent struct {
	Type       string   `json:"type"`
	ClockValue int64    `json:"clockValue"`
	Member     string   `json:"member,omitempty"`  // in "member-joined", "member-removed" and "admin-removed" events
	Members    []string `json:"members,omitempty"` // in "members-added" and "admins-added" events
	Name       string   `json:"name,omitempty"`    // name of the group chat
}

func NewChatCreatedEvent(name string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateChatCreated,
		Name:       name,
		ClockValue: clock,
	}
}

func NewNameChangedEvent(name string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateNameChanged,
		Name:       name,
		ClockValue: clock,
	}
}

func NewMembersAddedEvent(members []string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateMembersAdded,
		Members:    members,
		ClockValue: clock,
	}
}

func NewMemberJoinedEvent(member string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateMemberJoined,
		Member:     member,
		ClockValue: clock,
	}
}

func NewAdminsAddedEvent(admins []string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateAdminsAdded,
		Members:    admins,
		ClockValue: clock,
	}
}

func NewMemberRemovedEvent(member string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateMemberRemoved,
		Member:     member,
		ClockValue: clock,
	}
}

func NewAdminRemovedEvent(admin string, clock int64) MembershipUpdateEvent {
	return MembershipUpdateEvent{
		Type:       MembershipUpdateAdminRemoved,
		Member:     admin,
		ClockValue: clock,
	}
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

var membershipUpdateEventFieldNamesCompat = map[string]string{
	"ClockValue": "clock-value",
	"Name":       "name",
	"Type":       "type",
	"Member":     "member",
	"Members":    "members",
}

func tupleMembershipUpdateEvent(update MembershipUpdateEvent) [][]interface{} {
	// Sort all slices first.
	sort.Slice(update.Members, func(i, j int) bool {
		return update.Members[i] < update.Members[j]
	})
	v := reflect.ValueOf(update)
	result := make([][]interface{}, 0, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		if name, exists := membershipUpdateEventFieldNamesCompat[fieldName]; exists {
			fieldName = name
		}
		field := v.Field(i)
		if !isZeroValue(field) {
			result = append(result, []interface{}{fieldName, field.Interface()})
		}
	}
	// Sort the result lexicographically.
	// We know that the first item of a tuple is a string
	// because it's a field name.
	sort.Slice(result, func(i, j int) bool {
		return result[i][0].(string) < result[j][0].(string)
	})
	return result
}

type Group struct {
	ChatID   string
	Admins   []string
	Contacts []string
}

// ValidateEvent returns true if a given event is valid.
func (g *Group) ValidateEvent(from string, event MembershipUpdateEvent) bool {
	switch event.Type {
	case MembershipUpdateChatCreated:
		return len(g.Admins) == 0 && len(g.Contacts) == 0
	case MembershipUpdateNameChanged:
		return stringSliceContains(g.Admins, from) && len(event.Name) > 0
	case MembershipUpdateMembersAdded:
		return stringSliceContains(g.Admins, from)
	case MembershipUpdateMemberJoined:
		return stringSliceContains(g.Contacts, from) && from == event.Member
	case MembershipUpdateMemberRemoved:
		// Member can remove themselves or admin can remove a member.
		return from == event.Member || (stringSliceContains(g.Admins, from) && !stringSliceContains(g.Admins, event.Member))
	case MembershipUpdateAdminsAdded:
		return stringSliceContains(g.Admins, from) && stringSliceSubset(event.Members, g.Contacts)
	case MembershipUpdateAdminRemoved:
		return stringSliceContains(g.Admins, from) && from == event.Member
	default:
		return false
	}
}

func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func stringSliceSubset(subset []string, set []string) bool {
	for _, item1 := range set {
		var found bool
		for _, item2 := range subset {
			if item1 == item2 {
				found = true
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}
