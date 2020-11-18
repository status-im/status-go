package protocol

import (
	"crypto/ecdsa"
	"errors"
	"math/rand"

	"github.com/status-im/status-go/eth-node/crypto"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/communities"
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
	ChatTypeProfile
	ChatTypeTimeline
	ChatTypeCommunityChat
)

const pkStringLength = 68

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
	UnviewedMessagesCount uint            `json:"unviewedMessagesCount"`
	LastMessage           *common.Message `json:"lastMessage"`

	// Group chat fields
	// Members are the members who have been invited to the group chat
	Members []ChatMember `json:"members"`
	// MembershipUpdates is all the membership events in the chat
	MembershipUpdates []v1protocol.MembershipUpdateEvent `json:"membershipUpdateEvents"`

	// Generated username name of the chat for one-to-ones
	Alias string `json:"alias,omitempty"`
	// Identicon generated from public key
	Identicon string `json:"identicon"`

	// Muted is used to check whether we want to receive
	// push notifications for this chat
	Muted bool `json:"muted,omitempty"`

	// Public key of administrator who created invitation link
	InvitationAdmin string `json:"invitationAdmin,omitempty"`

	// Public key of user profile
	Profile string `json:"profile,omitempty"`

	// CommunityID is the id of the community it belongs to
	CommunityID string `json:"communityId,omitempty"`
}

func (c *Chat) PublicKey() (*ecdsa.PublicKey, error) {
	// For one to one chatID is an encoded public key
	if c.ChatType != ChatTypeOneToOne {
		return nil, nil
	}
	return common.HexToPubkey(c.ID)
}

func (c *Chat) Public() bool {
	return c.ChatType == ChatTypePublic
}

func (c *Chat) ProfileUpdates() bool {
	return c.ChatType == ChatTypeProfile
}

func (c *Chat) Timeline() bool {
	return c.ChatType == ChatTypeTimeline
}

func (c *Chat) OneToOne() bool {
	return c.ChatType == ChatTypeOneToOne
}

func (c *Chat) CommunityChatID() string {
	if c.ChatType != ChatTypeCommunityChat {
		return c.ID
	}

	// Strips out the local prefix of the community-id
	return c.ID[pkStringLength:]
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

func (c *Chat) MembersAsPublicKeys() ([]*ecdsa.PublicKey, error) {
	publicKeys := make([]string, len(c.Members))
	for idx, item := range c.Members {
		publicKeys[idx] = item.ID
	}
	return stringSliceToPublicKeys(publicKeys)
}

func (c *Chat) JoinedMembersAsPublicKeys() ([]*ecdsa.PublicKey, error) {
	var publicKeys []string
	for _, member := range c.Members {
		if member.Joined {
			publicKeys = append(publicKeys, member.ID)
		}
	}
	return stringSliceToPublicKeys(publicKeys)
}

func (c *Chat) HasMember(memberID string) bool {
	for _, member := range c.Members {
		if memberID == member.ID {
			return true
		}
	}

	return false
}

func (c *Chat) updateChatFromGroupMembershipChanges(myID string, g *v1protocol.Group) {

	// Check if we were already in the chat
	hadJoined := c.HasMember(myID)

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
		// Check if we joined thanks to these changes, if so, make chat active
		if m == myID && !hadJoined {
			c.Active = true
		}

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
func (c *Chat) NextClockAndTimestamp(timesource common.TimeSource) (uint64, uint64) {
	clock := c.LastClockValue
	timestamp := timesource.GetCurrentTime()
	if clock == 0 || clock < timestamp {
		clock = timestamp
	} else {
		clock = clock + 1
	}
	return clock, timestamp
}

func (c *Chat) UpdateFromMessage(message *common.Message, timesource common.TimeSource) error {
	c.Timestamp = int64(timesource.GetCurrentTime())

	// If the clock of the last message is lower, we set the message
	if c.LastMessage == nil || c.LastMessage.Clock <= message.Clock {
		c.LastMessage = message
	}
	// If the clock is higher we set the clock
	if c.LastClockValue < message.Clock {
		c.LastClockValue = message.Clock
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
	return common.HexToPubkey(c.ID)
}

func oneToOneChatID(publicKey *ecdsa.PublicKey) string {
	return types.EncodeHex(crypto.FromECDSAPub(publicKey))
}

func OneToOneFromPublicKey(pk *ecdsa.PublicKey, timesource common.TimeSource) *Chat {
	chatID := types.EncodeHex(crypto.FromECDSAPub(pk))
	newChat := CreateOneToOneChat(chatID[:8], pk, timesource)

	return &newChat
}

func CreateOneToOneChat(name string, publicKey *ecdsa.PublicKey, timesource common.TimeSource) Chat {
	return Chat{
		ID:        oneToOneChatID(publicKey),
		Name:      name,
		Timestamp: int64(timesource.GetCurrentTime()),
		Active:    true,
		ChatType:  ChatTypeOneToOne,
	}
}

func CreateCommunityChat(orgID, chatID string, orgChat *protobuf.CommunityChat, timesource common.TimeSource) Chat {
	color := orgChat.Identity.Color
	if color == "" {
		color = chatColors[rand.Intn(len(chatColors))]
	}

	return Chat{
		CommunityID: orgID,
		Name:        orgChat.Identity.DisplayName,
		Active:      true,
		Color:       color,
		ID:          orgID + chatID,
		Timestamp:   int64(timesource.GetCurrentTime()),
		ChatType:    ChatTypeCommunityChat,
	}
}

func CreateCommunityChats(org *communities.Community, timesource common.TimeSource) []Chat {
	var chats []Chat
	orgID := org.IDString()

	for chatID, chat := range org.Chats() {
		chats = append(chats, CreateCommunityChat(orgID, chatID, chat, timesource))
	}
	return chats
}

func CreatePublicChat(name string, timesource common.TimeSource) Chat {
	return Chat{
		ID:        name,
		Name:      name,
		Active:    true,
		Timestamp: int64(timesource.GetCurrentTime()),
		Color:     chatColors[rand.Intn(len(chatColors))],
		ChatType:  ChatTypePublic,
	}
}

func CreateProfileChat(name string, profile string, timesource common.TimeSource) Chat {
	return Chat{
		ID:        name,
		Name:      name,
		Active:    true,
		Timestamp: int64(timesource.GetCurrentTime()),
		Color:     chatColors[rand.Intn(len(chatColors))],
		ChatType:  ChatTypeProfile,
		Profile:   profile,
	}
}

func CreateGroupChat(timesource common.TimeSource) Chat {
	return Chat{
		Active:    true,
		Color:     chatColors[rand.Intn(len(chatColors))],
		Timestamp: int64(timesource.GetCurrentTime()),
		ChatType:  ChatTypePrivateGroupChat,
	}
}

func stringSliceToPublicKeys(slice []string) ([]*ecdsa.PublicKey, error) {
	result := make([]*ecdsa.PublicKey, len(slice))
	for idx, item := range slice {
		var err error
		result[idx], err = common.HexToPubkey(item)
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
