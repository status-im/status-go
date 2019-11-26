package waku

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// Waku protocol parameters
const (
	ProtocolVersion    = uint64(0) // Protocol version number
	ProtocolVersionStr = "0.1"     // The same, as a string
	ProtocolName       = "waku"    // Nickname of the protocol

	// Waku protocol message codes, according to https://github.com/vacp2p/specs/blob/master/waku.md
	statusCode         = 0 // used in the handshake
	messagesCode       = 1 // regular message
	powRequirementCode = 2 // node's PoW requirement
	bloomFilterExCode  = 3 // bloom filter exchange
	// TODO: add rate limiting code
	// TODO: message confirm
	p2pRequestCompleteCode = 125 // peer-to-peer message, used by Dapp protocol
	p2pRequestCode         = 126 // peer-to-peer message, used by Dapp protocol
	p2pMessageCode         = 127 // peer-to-peer message (to be consumed by the peer, but not forwarded any further)
	NumberOfMessageCodes   = 128

	SizeMask      = byte(3) // mask used to extract the size of payload size field from the flags
	signatureFlag = byte(4)

	TopicLength     = 4                      // in bytes
	signatureLength = crypto.SignatureLength // in bytes
	aesKeyLength    = 32                     // in bytes
	aesNonceLength  = 12                     // in bytes; for more info please see cipher.gcmStandardNonceSize & aesgcm.NonceSize()
	keyIDSize       = 32                     // in bytes
	BloomFilterSize = 64                     // in bytes
	flagsLength     = 1

	EnvelopeHeaderLength = 20

	MaxMessageSize        = uint32(10 * 1024 * 1024) // maximum accepted size of a message.
	DefaultMaxMessageSize = uint32(1024 * 1024)
	DefaultMinimumPoW     = 0.2

	padSizeLimit      = 256 // just an arbitrary number, could be changed without breaking the protocol
	messageQueueLimit = 1024

	expirationCycle   = time.Second
	transmissionCycle = 300 * time.Millisecond

	DefaultTTL           = 50 // seconds
	DefaultSyncAllowance = 10 // seconds

	MaxLimitInSyncMailRequest = 1000

	EnvelopeTimeNotSynced uint = iota + 1
	EnvelopeOtherError

	MaxLimitInMessagesRequest = 1000
)

// MailServer represents a mail server, capable of
// archiving the old messages for subsequent delivery
// to the peers. Any implementation must ensure that both
// functions are thread-safe. Also, they must return ASAP.
// DeliverMail should use p2pMessageCode for delivery,
// in order to bypass the expiry checks.
type MailServer interface {
	Archive(env *Envelope)
	DeliverMail(whisperPeer *Peer, request *Envelope) // DEPRECATED
	Deliver(whisperPeer *Peer, request MessagesRequest)
	SyncMail(*Peer, SyncMailRequest) error
}

// MessagesRequest contains details of a request of historic messages.
type MessagesRequest struct {
	// ID of the request. The current implementation requires ID to be 32-byte array,
	// however, it's not enforced for future implementation.
	ID []byte `json:"id"`

	// From is a lower bound of time range.
	From uint32 `json:"from"`

	// To is a upper bound of time range.
	To uint32 `json:"to"`

	// Limit determines the number of messages sent by the mail server
	// for the current paginated request.
	Limit uint32 `json:"limit"`

	// Cursor is used as starting point for paginated requests.
	Cursor []byte `json:"cursor"`

	// Bloom is a filter to match requested messages.
	Bloom []byte `json:"bloom"`
}

func (r MessagesRequest) Validate() error {
	if len(r.ID) != common.HashLength {
		return errors.New("invalid 'ID', expected a 32-byte slice")
	}

	if r.From > r.To {
		return errors.New("invalid 'From' value which is greater than To")
	}

	if r.Limit > MaxLimitInMessagesRequest {
		return fmt.Errorf("invalid 'Limit' value, expected value lower than %d", MaxLimitInMessagesRequest)
	}

	if len(r.Bloom) == 0 {
		return errors.New("invalid 'Bloom' provided")
	}

	return nil
}

// MessagesResponse sent as a response after processing batch of envelopes.
type MessagesResponse struct {
	// Hash is a hash of all envelopes sent in the single batch.
	Hash common.Hash
	// Per envelope error.
	Errors []EnvelopeError
}

// EnvelopeError code and optional description of the error.
type EnvelopeError struct {
	Hash        common.Hash
	Code        uint
	Description string
}

// MultiVersionResponse allows to decode response into chosen version.
type MultiVersionResponse struct {
	Version  uint
	Response rlp.RawValue
}

// DecodeResponse1 decodes response into first version of the messages response.
func (m MultiVersionResponse) DecodeResponse1() (resp MessagesResponse, err error) {
	return resp, rlp.DecodeBytes(m.Response, &resp)
}

// Version1MessageResponse first version of the message response.
type Version1MessageResponse struct {
	Version  uint
	Response MessagesResponse
}

// NewMessagesResponse returns instance of the version messages response.
func NewMessagesResponse(batch common.Hash, errors []EnvelopeError) Version1MessageResponse {
	return Version1MessageResponse{
		Version: 1,
		Response: MessagesResponse{
			Hash:   batch,
			Errors: errors,
		},
	}
}

// ErrorToEnvelopeError converts common golang error into EnvelopeError with a code.
func ErrorToEnvelopeError(hash common.Hash, err error) EnvelopeError {
	code := EnvelopeOtherError
	switch err.(type) {
	case TimeSyncError:
		code = EnvelopeTimeNotSynced
	}
	return EnvelopeError{
		Hash:        hash,
		Code:        code,
		Description: err.Error(),
	}
}
