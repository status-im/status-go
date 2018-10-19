package whisperv6

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
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
	payload := append(requestID[:], lastEnvelopeHash[:]...)
	payload = append(payload, cursor...)
	return payload
}

// CreateMailServerRequestFailedPayload creates a payload representing
// a failed request to a mailserver
func CreateMailServerRequestFailedPayload(requestID common.Hash, err error) []byte {
	payloadPrefix := []byte(mailServerFailedPayloadPrefix)
	errorString := []byte(err.Error())
	payload := append(payloadPrefix, requestID[:]...)
	payload = append(payload, errorString[:]...)
	return payload
}

// CreateMailServerEvent returns EnvelopeEvent with correct data
// if payload corresponds to any of the know mailserver events:
// * request completed successfully
// * request failed
// If the payload is unknown/unparseable, it returns `nil`
func CreateMailServerEvent(payload []byte) (*EnvelopeEvent, error) {

	if len(payload) < common.HashLength {
		return nil, invalidResponseSizeError(len(payload))
	}

	event, err := tryCreateMailServerRequestFailedEvent(payload)

	if err != nil || event != nil {
		return event, err
	}

	return tryCreateMailServerRequestCompletedEvent(payload)
}

func tryCreateMailServerRequestFailedEvent(payload []byte) (*EnvelopeEvent, error) {
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
		Hash:  requestID,
		Event: EventMailServerRequestCompleted,
		Data: &MailServerResponse{
			Error: errors.New(errorMsg),
		},
	}

	return &event, nil

}

func tryCreateMailServerRequestCompletedEvent(payload []byte) (*EnvelopeEvent, error) {
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
