// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

/*
Package whisper implements the Whisper protocol (version 6).

Whisper combines aspects of both DHTs and datagram messaging systems (e.g. UDP).
As such it may be likened and compared to both, not dissimilar to the
matter/energy duality (apologies to physicists for the blatant abuse of a
fundamental and beautiful natural principle).

Whisper is a pure identity-based messaging system. Whisper provides a low-level
(non-application-specific) but easily-accessible API without being based upon
or prejudiced by the low-level hardware attributes and characteristics,
particularly the notion of singular endpoints.
*/

// Contains the Whisper protocol constant definitions

package whisperv6

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// Whisper protocol parameters
const (
	ProtocolVersion    = uint64(6) // Protocol version number
	ProtocolVersionStr = "6.0"     // The same, as a string
	ProtocolName       = "shh"     // Nickname of the protocol in geth

	// whisper protocol message codes, according to EIP-627
	statusCode             = 0   // used by whisper protocol
	messagesCode           = 1   // normal whisper message
	powRequirementCode     = 2   // PoW requirement
	bloomFilterExCode      = 3   // bloom filter exchange
	batchAcknowledgedCode  = 11  // confirmation that batch of envelopes was received
	messageResponseCode    = 12  // includes confirmation for delivery and information about errors
	p2pSyncRequestCode     = 123 // used to sync envelopes between two mail servers
	p2pSyncResponseCode    = 124 // used to sync envelopes between two mail servers
	p2pRequestCompleteCode = 125 // peer-to-peer message, used by Dapp protocol
	p2pRequestCode         = 126 // peer-to-peer message, used by Dapp protocol
	p2pMessageCode         = 127 // peer-to-peer message (to be consumed by the peer, but not forwarded any further)
	NumberOfMessageCodes   = 128

	SizeMask      = byte(3) // mask used to extract the size of payload size field from the flags
	signatureFlag = byte(4)

	TopicLength     = 4  // in bytes
	signatureLength = crypto.SignatureLength // in bytes
	aesKeyLength    = 32 // in bytes
	aesNonceLength  = 12 // in bytes; for more info please see cipher.gcmStandardNonceSize & aesgcm.NonceSize()
	keyIDSize       = 32 // in bytes
	BloomFilterSize = 64 // in bytes
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
)

// MailServer represents a mail server, capable of
// archiving the old messages for subsequent delivery
// to the peers. Any implementation must ensure that both
// functions are thread-safe. Also, they must return ASAP.
// DeliverMail should use directMessagesCode for delivery,
// in order to bypass the expiry checks.
type MailServer interface {
	Archive(env *Envelope)
	DeliverMail(whisperPeer *Peer, request *Envelope)
	SyncMail(*Peer, SyncMailRequest) error
}

// SyncMailRequest contains details which envelopes should be synced
// between Mail Servers.
type SyncMailRequest struct {
	// Lower is a lower bound of time range for which messages are requested.
	Lower uint32
	// Upper is a lower bound of time range for which messages are requested.
	Upper uint32
	// Bloom is a bloom filter to filter envelopes.
	Bloom []byte
	// Limit is the max number of envelopes to return.
	Limit uint32
	// Cursor is used for pagination of the results.
	Cursor []byte
}

// Validate checks request's fields if they are valid.
func (r SyncMailRequest) Validate() error {
	if r.Limit == 0 {
		return errors.New("invalid 'Limit' value, expected value greater than 0")
	}

	if r.Limit > MaxLimitInSyncMailRequest {
		return fmt.Errorf("invalid 'Limit' value, expected value lower than %d", MaxLimitInSyncMailRequest)
	}

	if r.Lower > r.Upper {
		return errors.New("invalid 'Lower' value, can't be greater than 'Upper'")
	}

	return nil
}

// SyncResponse is a struct representing a response sent to the peer
// asking for syncing archived envelopes.
type SyncResponse struct {
	Envelopes []*Envelope
	Cursor    []byte
	Final     bool // if true it means all envelopes were processed
	Error     string
}

// RawSyncResponse is a struct representing a response sent to the peer
// asking for syncing archived envelopes.
type RawSyncResponse struct {
	Envelopes []rlp.RawValue
	Cursor    []byte
	Final     bool // if true it means all envelopes were processed
	Error     string
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

// NewMessagesResponse returns instane of the version messages response.
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
