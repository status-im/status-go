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
	"github.com/ethereum/go-ethereum/common"
)

// EventType used to define known waku events.
type EventType string

const (
	// EventEnvelopeSent fires when an envelope is confirmed as sent into the network.
	EventEnvelopeSent EventType = "envelope.sent"

	// EventEnvelopeExpired fires when an envelope failed to be sent and won't be retried.
	EventEnvelopeExpired EventType = "envelope.expired"

	// EventEnvelopeAvailable fires when a received envelope is available for the
	// transport to decode and route.
	EventEnvelopeAvailable EventType = "envelope.available"
)

// EnvelopeEvent represents an envelope event.
type EnvelopeEvent struct {
	Event EventType
	Hash  common.Hash
	Data  interface{}
}
