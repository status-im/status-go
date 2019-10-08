package statusproto

import (
	"crypto/ecdsa"
	"crypto/sha1"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	statusproto "github.com/status-im/status-protocol-go/types"
	protocol "github.com/status-im/status-protocol-go/v1"
)

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

	// Only filled for one to one chats
	PublicKey *ecdsa.PublicKey `json:"-"`

	// Timestamp indicates the last time this chat has received/sent a message
	Timestamp int64 `json:"timestamp"`
	// LastClockValue indicates the last clock value to be used when sending messages
	LastClockValue uint64 `json:"lastClockValue"`
	// DeletedAtClockValue indicates the clock value at time of deletion, messages
	// with lower clock value of this should be discarded
	DeletedAtClockValue uint64 `json:"deletedAtClockValue"`

	// Denormalized fields
	UnviewedMessagesCount  uint   `json:"unviewedMessagesCount"`
	LastMessageContentType string `json:"lastMessageContentType"`
	LastMessageContent     string `json:"lastMessageContent"`
	LastMessageTimestamp   int64  `json:"lastMessageTimestamp"`
	LastMessageClockValue  int64  `json:"lastMessageClockValue"`

	// Group chat fields
	// Members are the members who have been invited to the group chat
	Members []ChatMember `json:"members"`
	// MembershipUpdates is all the membership events in the chat
	MembershipUpdates []ChatMembershipUpdate `json:"membershipUpdates"`
}

func (c *Chat) MembersAsPublicKeys() ([]*ecdsa.PublicKey, error) {
	publicKeys := make([]string, len(c.Members))
	for idx, item := range c.Members {
		publicKeys[idx] = item.ID
	}
	return stringSliceToPublicKeys(publicKeys, true)
}

func (c *Chat) updateChatFromProtocolGroup(g *protocol.Group) {
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
	updates := g.Updates()
	membershipUpdates := make([]ChatMembershipUpdate, 0, len(updates))
	for _, update := range updates {
		membershipUpdate := ChatMembershipUpdate{
			Type:       update.Type,
			Name:       update.Name,
			ClockValue: uint64(update.ClockValue), // TODO: get rid of type casting
			Signature:  update.Signature,
			From:       update.From,
			Member:     update.Member,
			Members:    update.Members,
		}
		membershipUpdate.setID()
		membershipUpdates = append(membershipUpdates, membershipUpdate)
	}
	c.MembershipUpdates = membershipUpdates
}

// ChatMembershipUpdate represent an event on membership of the chat
type ChatMembershipUpdate struct {
	// Unique identifier for the event
	ID string `json:"id"`
	// Type indicates the kind of event (i.e changed-name, added-member, etc)
	Type string `json:"type"`
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

func (u *ChatMembershipUpdate) setID() {
	sum := sha1.Sum([]byte(u.Signature))
	u.ID = hex.EncodeToString(sum[:])
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
	b, err := statusproto.DecodeHex(c.ID)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPubkey(b)
}

func oneToOneChatID(publicKey *ecdsa.PublicKey) string {
	return statusproto.EncodeHex(crypto.FromECDSAPub(publicKey))
}

func CreateOneToOneChat(name string, publicKey *ecdsa.PublicKey) Chat {
	return Chat{
		ID:        oneToOneChatID(publicKey),
		Name:      name,
		Active:    true,
		ChatType:  ChatTypeOneToOne,
		PublicKey: publicKey,
	}
}

func CreatePublicChat(name string) Chat {
	return Chat{
		ID:       name,
		Name:     name,
		Active:   true,
		ChatType: ChatTypePublic,
	}
}

func createGroupChat() Chat {
	return Chat{
		Active:   true,
		ChatType: ChatTypePrivateGroupChat,
	}
}

func findChatByID(chatID string, chats []*Chat) *Chat {
	for _, c := range chats {
		if c.ID == chatID {
			return c
		}
	}
	return nil
}

func stringSliceToPublicKeys(slice []string, prefixed bool) ([]*ecdsa.PublicKey, error) {
	result := make([]*ecdsa.PublicKey, len(slice))
	for idx, item := range slice {
		var (
			b   []byte
			err error
		)
		if prefixed {
			b, err = hexutil.Decode(item)
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
