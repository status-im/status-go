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

package common

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// TopicType represents a cryptographically secure, probabilistic partial
// classifications of a message, determined as the first (leftmost) 4 bytes of the
// SHA3 hash of some arbitrary data given by the original author of the message.
type TopicType uint32

// BytesToTopic converts from the byte array representation of a topic
// into the TopicType type.
func BytesToTopic(b []byte) (t TopicType) {
	return TopicType(binary.LittleEndian.Uint32(b))
}

func uint32ToByte(input uint32) []byte {
	result := make([]byte, 4)
	for i := uint32(0); i < 4; i++ {
		result[i] = byte((input >> (8 * i)) & 0xff)
	}
	return result
}

func byteToUint32(input []byte) uint32 {
	var result uint32
	for i := 0; i < 4; i++ {
		result |= uint32(input[i]) << (8 * i)
	}
	return result
}

// String converts a topic byte array to a string representation.
func (t *TopicType) String() string {
	return hexutil.Encode(uint32ToByte(uint32(*t)))
}

// MarshalText returns the hex representation of t.
func (t TopicType) MarshalText() ([]byte, error) {
	return hexutil.Bytes(uint32ToByte(uint32(t))).MarshalText()
}

// UnmarshalText parses a hex representation to a topic.
func (t *TopicType) UnmarshalText(input []byte) error {
	var r []byte
	var result TopicType
	err := hexutil.UnmarshalFixedText("Topic", input, r)
	result = TopicType(byteToUint32(r))
	t = &result
	return err

}
