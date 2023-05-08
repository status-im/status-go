package protocol

import (
	"crypto/ecdsa"

	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/protobuf"
)

// DeleteForMeMessage represents a delete of a message from a user in the
// application layer, used for persistence, querying and signaling
type DeleteForMeMessage struct {
	protobuf.DeleteForMeMessage

	// ID is the ID of the message that has been edited
	ID string `json:"id,omitempty"`

	// From is a public key of the author of the edit reaction.
	From string `json:"from,omitempty"`

	// SigPubKey is the ecdsa encoded public key of the edit author
	SigPubKey *ecdsa.PublicKey `json:"-"`
}

// GetSigPubKey returns an ecdsa encoded public key
// this function is required to implement the ChatEntity interface
func (e DeleteForMeMessage) GetSigPubKey() *ecdsa.PublicKey {
	return e.SigPubKey
}

// GetProtoBuf returns the struct's embedded protobuf struct
// this function is required to implement the ChatEntity interface
func (e DeleteForMeMessage) GetProtobuf() proto.Message {
	return &e.DeleteForMeMessage
}

// WrapGroupMessage indicates whether we should wrap this in membership information
func (e DeleteForMeMessage) WrapGroupMessage() bool {
	return false
}
