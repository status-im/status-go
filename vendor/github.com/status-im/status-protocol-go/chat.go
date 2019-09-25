package statusproto

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type ChatPagination struct {
	From uint
	To   uint
}

type ChatType int

const (
	ChatTypeOneToOne = iota + 1
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

	// Group chat fields
	// Members are the members who have been invited to the group chat
	Members []ChatMember `json:"members"`
	// MembershipUpdates is all the membership events in the chat
	MembershipUpdates []ChatMembershipUpdate `json:"membershipUpdates"`
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
	b, err := hexutil.Decode(c.ID)
	if err != nil {
		return nil, err
	}
	return crypto.UnmarshalPubkey(b)
}
