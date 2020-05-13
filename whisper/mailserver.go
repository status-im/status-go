package whisper

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	mailServerFailedPayloadPrefix = "ERROR="
	cursorSize                    = 36
)

// MailServer represents a mail server, capable of
// archiving the old messages for subsequent delivery
// to the peers. Any implementation must ensure that both
// functions are thread-safe. Also, they must return ASAP.
// DeliverMail should use directMessagesCode for delivery,
// in order to bypass the expiry checks.
type MailServer interface {
	Archive(env *Envelope)
	DeliverMail(peerID []byte, req *Envelope) // DEPRECATED; user Deliver instead
	Deliver(peerID []byte, req MessagesRequest)
	SyncMail(peerID []byte, req SyncMailRequest) error
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

	if err != nil || event != nil {
		return event, err
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
