package datasync

import (
	"github.com/status-im/mvds/protobuf"
	"github.com/status-im/mvds/state"

	"github.com/status-im/status-go/pkg/messaging/diagnostics"
)

// PayloadTraceMetadata copies identifiers only; payload bodies never enter signals.
func PayloadTraceMetadata(peer state.PeerID, payload *protobuf.Payload) diagnostics.Metadata {
	metadata := diagnostics.Metadata{Reliability: "mvds"}
	if !diagnostics.Enabled() {
		return metadata
	}
	metadata.Peer = diagnostics.Hexes([][]byte{peer[:]})[0]
	if payload == nil {
		return metadata
	}
	metadata.AckIDs = diagnostics.Hexes(payload.Acks)
	for _, message := range payload.Messages {
		if message != nil {
			id := message.ID()
			metadata.MVDSMessageIDs = append(metadata.MVDSMessageIDs, diagnostics.Hexes([][]byte{id[:]})[0])
		}
	}
	return metadata
}
