package types

import (
	"crypto/ecdsa"

	"github.com/status-im/status-go/protocol/protobuf"
)

type SendPublicParams struct {
	Sender              *ecdsa.PublicKey
	Payload             []byte
	PubsubTopic         string
	ContentTopic        string
	SkipEncryptionLayer bool
	Ephemeral           bool
	Priority            *MessagePriority
	HashRatchet         *SendPublicHashRatchetParams
	CommunityPublicKey  *ecdsa.PublicKey
	CommunityID         []byte
	MessageType         protobuf.ApplicationMetadataMessage_Type
}

type SendPublicHashRatchetParams struct {
	Encrypt   bool
	GroupID   []byte
	KeyExType CommKeyExMsgType
	Members   []*ecdsa.PublicKey
}

type SendPrivateParams struct {
	Sender              *ecdsa.PrivateKey
	Recipient           *ecdsa.PublicKey
	Payload             []byte
	PubsubTopic         string
	WithReliability     bool
	SendWithDH          bool
	SkipEncryptionLayer bool
	SendOnPersonalTopic bool
	HashRatchetGroupID  []byte
}

type MessagePriority = int

var (
	LowPriority    MessagePriority = 0
	NormalPriority MessagePriority = 1
	HighPriority   MessagePriority = 2
)

type CommKeyExMsgType uint8

const (
	KeyExMsgNone  CommKeyExMsgType = 0
	KeyExMsgReuse CommKeyExMsgType = 1
	KeyExMsgRekey CommKeyExMsgType = 2
)
