package protocol

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/rand"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/protobuf"
	v1protocol "github.com/status-im/status-go/protocol/v1"
)

var chatColors = []string{
	"#fa6565", // red
	"#887af9", // blue
	"#FE8F59", // orange
	"#7cda00", // green
	"#51d0f0", // light-blue
	"#d37ef4", // purple
}

type ChatType int

const (
	ChatTypeOneToOne ChatType = iota + 1
	ChatTypePublic
	ChatTypePrivateGroupChat
)

type Chat struct {
	// ID is the id of the chat, for public chats it is the name e.g. status, for one-to-one
	// is the hex encoded public key and for group chats is a random uuid appended with
	// the hex encoded pk of the creator of the chat
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	// Active indicates whether the chat has been soft deleted
	Active bool `json:"active"`

	ChatType ChatType `json:"chatType"`

	// Timestamp indicates the last time this chat has received/sent a message
	Timestamp int64 `json:"timestamp"`
	// LastClockValue indicates the last clock value to be used when sending messages
	LastClockValue uint64 `json:"lastClockValue"`
	// DeletedAtClockValue indicates the clock value at time of deletion, messages
	// with lower clock value of this should be discarded
	DeletedAtClockValue uint64 `json:"deletedAtClockValue"`

	// Denormalized fields
	UnviewedMessagesCount uint   `json:"unviewedMessagesCount"`
	LastMessage           []byte `json:"lastMessage"`

	// Group chat fields
	// Members are the members who have been invited to the group chat
	Members []ChatMember `json:"members"`
	// MembershipUpdates is all the membership events in the chat
	MembershipUpdates []v1protocol.MembershipUpdateEvent `json:"membershipUpdateEvents"`
}

func (c *Chat) PublicKey() (*ecdsa.PublicKey, error) {
	// For one to one chatID is an encoded public key
	if c.ChatType != ChatTypeOneToOne {
		return nil, nil
	}
	pkey, err := hex.DecodeString(c.ID[2:])
	if err != nil {
		return nil, err
	}
	// Safety check, make sure is well formed
	return crypto.UnmarshalPubkey(pkey)

}

func (c *Chat) Public() bool {
	return c.ChatType == ChatTypePublic
}

func (c *Chat) OneToOne() bool {
	return c.ChatType == ChatTypeOneToOne
}

func (c *Chat) Validate() error {
	if c.ID == "" {
		return errors.New("chatID can't be blank")
	}

	if c.OneToOne() {
		_, err := c.PublicKey()
		return err
	}
	return nil
}

func (c *Chat) MarshalJSON() ([]byte, error) {
	type ChatAlias Chat
	item := struct {
		*ChatAlias
		LastMessage json.RawMessage `json:"lastMessage"`
	}{
		ChatAlias:   (*ChatAlias)(c),
		LastMessage: c.LastMessage,
	}

	return json.Marshal(item)
}

func (c *Chat) UnmarshalJSON(data []byte) error {
	type ChatAlias Chat
	aux := struct {
		*ChatAlias
		LastMessage *Message `json:"lastMessage"`
	}{
		ChatAlias: (*ChatAlias)(c),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.ID = aux.ID
	c.Name = aux.Name
	c.Color = aux.Color
	c.Active = aux.Active
	c.ChatType = aux.ChatType
	c.Timestamp = aux.Timestamp
	c.LastClockValue = aux.LastClockValue
	c.DeletedAtClockValue = aux.DeletedAtClockValue
	c.UnviewedMessagesCount = aux.UnviewedMessagesCount
	c.Members = aux.Members
	c.MembershipUpdates = aux.MembershipUpdates

	if aux.LastMessage != nil {
		data, err := json.Marshal(aux.LastMessage)
		if err != nil {
			return err
		}
		c.LastMessage = data
	}
	return nil
}

func (c *Chat) MembersAsPublicKeys() ([]*ecdsa.PublicKey, error) {
	publicKeys := make([]string, len(c.Members))
	for idx, item := range c.Members {
		publicKeys[idx] = item.ID
	}
	return stringSliceToPublicKeys(publicKeys, true)
}

func (c *Chat) updateChatFromProtocolGroup(g *v1protocol.Group) {
	// ID
	c.ID = g.ChatID()

	// Name
	c.Name = g.Name()

	// Members
	members := g.Members()
	admins := g.Admins()
	joined := g.Joined()
	chatMembers := make([]ChatMember, 0, len(members))
	for _, m := range members {
		chatMember := ChatMember{
			ID: m,
		}
		chatMember.Admin = stringSliceContains(admins, m)
		chatMember.Joined = stringSliceContains(joined, m)
		chatMembers = append(chatMembers, chatMember)
	}
	c.Members = chatMembers

	// MembershipUpdates
	c.MembershipUpdates = g.Events()
}

// NextClockAndTimestamp returns the next clock value
// and the current timestamp
func (c *Chat) NextClockAndTimestamp(timesource TimeSource) (uint64, uint64) {
	clock := c.LastClockValue
	timestamp := timesource.GetCurrentTime()
	if clock == 0 || clock < timestamp {
		clock = timestamp
	} else {
		clock = clock + 1
	}
	return clock, timestamp
}

func (c *Chat) UpdateFromMessage(message *Message, timesource TimeSource) error {
	c.Timestamp = int64(timesource.GetCurrentTime())
	if c.LastClockValue <= message.Clock {
		jsonMessage, err := json.Marshal(message)
		if err != nil {
			return err
		}

		c.LastClockValue = message.Clock
		c.LastMessage = jsonMessage
	}
	return nil
}

// ChatMembershipUpdate represent an event on membership of the chat
type ChatMembershipUpdate struct {
	// Unique identifier for the event
	ID string `json:"id"`
	// Type indicates the kind of event
	Type protobuf.MembershipUpdateEvent_EventType `json:"type"`
	// Name represents the name in the event of changing name events
	Name string `json:"name,omitempty"`
	// Clock value of the event
	ClockValue uint64 `json:"clockValue"`
	// Signature of the event
	Signature string `json:"signature"`
	// Hex encoded public key of the creator of the event
	From string `json:"from"`
	// Target of the event for single-target events
	Member string `json:"member,omitempty"`
	// Target of the event for multi-target events
	Members []string `json:"members,omitempty"`
}

// ChatMember represents a member who participates in a group chat
type ChatMember struct {
	// ID is the hex encoded public key of the member
	ID string `json:"id"`
	// Admin indicates if the member is an admin of the group chat
	Admin bool `json:"admin"`
	// Joined indicates if the member has joined the group chat
	Joined bool `json:"joined"`
}

func (c ChatMember) PublicKey() (*ecdsa.PublicKey, error) {
	b, err := types.DecodeHex(c.ID)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPubkey(b)
}

func oneToOneChatID(publicKey *ecdsa.PublicKey) string {
	return types.EncodeHex(crypto.FromECDSAPub(publicKey))
}

func OneToOneFromPublicKey(pk *ecdsa.PublicKey, timesource TimeSource) *Chat {
	chatID := types.EncodeHex(crypto.FromECDSAPub(pk))
	newChat := CreateOneToOneChat(chatID[:8], pk, timesource)

	return &newChat
}

func CreateOneToOneChat(name string, publicKey *ecdsa.PublicKey, timesource TimeSource) Chat {
	return Chat{
		ID:        oneToOneChatID(publicKey),
		Name:      name,
		Timestamp: int64(timesource.GetCurrentTime()),
		Active:    true,
		ChatType:  ChatTypeOneToOne,
	}
}

func CreatePublicChat(name string, timesource TimeSource) Chat {
	return Chat{
		ID:        name,
		Name:      name,
		Active:    true,
		Timestamp: int64(timesource.GetCurrentTime()),
		Color:     chatColors[rand.Intn(len(chatColors))],
		ChatType:  ChatTypePublic,
	}
}

func CreateGroupChat(timesource TimeSource) Chat {
	return Chat{
		Active:    true,
		Color:     chatColors[rand.Intn(len(chatColors))],
		Timestamp: int64(timesource.GetCurrentTime()),
		ChatType:  ChatTypePrivateGroupChat,
	}
}

func stringSliceToPublicKeys(slice []string, prefixed bool) ([]*ecdsa.PublicKey, error) {
	result := make([]*ecdsa.PublicKey, len(slice))
	for idx, item := range slice {
		var (
			b   []byte
			err error
		)
		if prefixed {
			b, err = types.DecodeHex(item)
		} else {
			b, err = hex.DecodeString(item)
		}
		if err != nil {
			return nil, err
		}
		result[idx], err = crypto.UnmarshalPubkey(b)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
