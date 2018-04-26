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
)

// EnvelopeEvent used for envelopes events.
type EnvelopeEvent struct {
       Event EventType
       Hash  common.Hash
       Peer  discover.NodeID
}
