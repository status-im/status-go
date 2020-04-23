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

// Originally inherited from https://github.com/ethereum/go-ethereum/blob/master/whisper/whisperv6/doc.go,
// refactored due to https://github.com/status-im/status-go/pull/1950

package whisper

import (
	"time"

	"github.com/ethereum/go-ethereum/crypto"
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
	rateLimitingCode       = 20  // includes peer's rate limiting settings
	p2pSyncRequestCode     = 123 // used to sync envelopes between two mail servers
	p2pSyncResponseCode    = 124 // used to sync envelopes between two mail servers
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
