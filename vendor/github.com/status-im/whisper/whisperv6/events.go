package whisperv6

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

// EventType used to define known envelope events.
type EventType string

const (
	// EventEnvelopeSent fires when envelope was sent to a peer.
	EventEnvelopeSent EventType = "envelope.sent"
	// EventEnvelopeExpired fires when envelop expired
	EventEnvelopeExpired EventType = "envelope.expired"
	// EventEnvelopeAvailable fires when envelop is available for filters
	EventEnvelopeAvailable EventType = "envelope.available"
	// EventMailServerRequestCompleted fires after mailserver sends all the requested messages
	EventMailServerRequestCompleted EventType = "mailserver.request.completed"
	// EventMailServerRequestExpired fires after mailserver the request TTL ends
	EventMailServerRequestExpired EventType = "mailserver.request.expired"
	// EventMailServerEnvelopeArchived fires after an envelope has been archived
	EventMailServerEnvelopeArchived EventType = "mailserver.envelope.archived"
)

// EnvelopeEvent used for envelopes events.
type EnvelopeEvent struct {
	Event EventType
	Hash  common.Hash
	Peer  discover.NodeID
	Data  interface{}
}
