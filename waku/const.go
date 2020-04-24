// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package waku

import (
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// Waku protocol parameters
const (
	ProtocolVersion    = uint64(0) // Protocol version number
	ProtocolVersionStr = "0"       // The same, as a string
	ProtocolName       = "waku"    // Nickname of the protocol

	// Waku protocol message codes, according to https://github.com/vacp2p/specs/blob/master/waku.md
	statusCode             = 0   // used in the handshake
	messagesCode           = 1   // regular message
	statusUpdateCode       = 22  // update of settings
	batchAcknowledgedCode  = 11  // confirmation that batch of envelopes was received
	messageResponseCode    = 12  // includes confirmation for delivery and information about errors
	p2pRequestCompleteCode = 125 // peer-to-peer message, used by Dapp protocol
	p2pRequestCode         = 126 // peer-to-peer message, used by Dapp protocol
	p2pMessageCode         = 127 // peer-to-peer message (to be consumed by the peer, but not forwarded any further)
	NumberOfMessageCodes   = 128

	SizeMask      = byte(3) // mask used to extract the size of payload size field from the flags
	signatureFlag = byte(4)

	TopicLength      = 4                      // in bytes
	signatureLength  = crypto.SignatureLength // in bytes
	aesKeyLength     = 32                     // in bytes
	aesNonceLength   = 12                     // in bytes; for more info please see cipher.gcmStandardNonceSize & aesgcm.NonceSize()
	keyIDSize        = 32                     // in bytes
	BloomFilterSize  = 64                     // in bytes
	MaxTopicInterest = 10000
	flagsLength      = 1

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
