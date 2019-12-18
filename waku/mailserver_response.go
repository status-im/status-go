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
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const (
	mailServerFailedPayloadPrefix = "ERROR="
	cursorSize                    = 36
)

func invalidResponseSizeError(size int) error {
	return fmt.Errorf("unexpected payload size: %d", size)
}

// CreateMailServerRequestCompletedPayload creates a payload representing
// a successful request to mailserver
func CreateMailServerRequestCompletedPayload(requestID, lastEnvelopeHash common.Hash, cursor []byte) []byte {
	payload := make([]byte, len(requestID))
	copy(payload, requestID[:])
	payload = append(payload, lastEnvelopeHash[:]...)
	payload = append(payload, cursor...)
	return payload
}

// CreateMailServerRequestFailedPayload creates a payload representing
// a failed request to a mailserver
func CreateMailServerRequestFailedPayload(requestID common.Hash, err error) []byte {
	payload := []byte(mailServerFailedPayloadPrefix)
	payload = append(payload, requestID[:]...)
	payload = append(payload, []byte(err.Error())...)
	return payload
}

// CreateMailServerEvent returns EnvelopeEvent with correct data
// if payload corresponds to any of the know mailserver events:
// * request completed successfully
// * request failed
// If the payload is unknown/unparseable, it returns `nil`
func CreateMailServerEvent(nodeID enode.ID, payload []byte) (*EnvelopeEvent, error) {
	if len(payload) < common.HashLength {
		return nil, invalidResponseSizeError(len(payload))
	}

	event, err := tryCreateMailServerRequestFailedEvent(nodeID, payload)
	if err != nil {
		return nil, err
	} else if event != nil {
		return event, nil
	}

	return tryCreateMailServerRequestCompletedEvent(nodeID, payload)
}

func tryCreateMailServerRequestFailedEvent(nodeID enode.ID, payload []byte) (*EnvelopeEvent, error) {
	if len(payload) < common.HashLength+len(mailServerFailedPayloadPrefix) {
		return nil, nil
	}

	prefix, remainder := extractPrefix(payload, len(mailServerFailedPayloadPrefix))

	if !bytes.Equal(prefix, []byte(mailServerFailedPayloadPrefix)) {
		return nil, nil
	}

	var (
		requestID common.Hash
		errorMsg  string
	)

	requestID, remainder = extractHash(remainder)
	errorMsg = string(remainder)

	event := EnvelopeEvent{
		Peer:  nodeID,
		Hash:  requestID,
		Event: EventMailServerRequestCompleted,
		Data: &MailServerResponse{
			Error: errors.New(errorMsg),
		},
	}

	return &event, nil

}

func tryCreateMailServerRequestCompletedEvent(nodeID enode.ID, payload []byte) (*EnvelopeEvent, error) {
	// check if payload is
	// - requestID or
	// - requestID + lastEnvelopeHash or
	// - requestID + lastEnvelopeHash + cursor
	// requestID is the hash of the request envelope.
	// lastEnvelopeHash is the last envelope sent by the mail server
	// cursor is the db key, 36 bytes: 4 for the timestamp + 32 for the envelope hash.
	if len(payload) > common.HashLength*2+cursorSize {
		return nil, invalidResponseSizeError(len(payload))
	}

	var (
		requestID        common.Hash
		lastEnvelopeHash common.Hash
		cursor           []byte
	)

	requestID, remainder := extractHash(payload)

	if len(remainder) >= common.HashLength {
		lastEnvelopeHash, remainder = extractHash(remainder)
	}

	if len(remainder) >= cursorSize {
		cursor = remainder
	}

	event := EnvelopeEvent{
		Peer:  nodeID,
		Hash:  requestID,
		Event: EventMailServerRequestCompleted,
		Data: &MailServerResponse{
			LastEnvelopeHash: lastEnvelopeHash,
			Cursor:           cursor,
		},
	}

	return &event, nil
}

func extractHash(payload []byte) (common.Hash, []byte) {
	prefix, remainder := extractPrefix(payload, common.HashLength)
	return common.BytesToHash(prefix), remainder
}

func extractPrefix(payload []byte, size int) ([]byte, []byte) {
	return payload[:size], payload[size:]
}
