package types

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type Traffic struct {
	BytesSent     int64 `json:"bytesSent"`
	BytesReceived int64 `json:"bytesReceived"`
}

type ProtocolTraffic struct {
	// Protocol is empty for streams that have not negotiated one yet.
	Protocol protocol.ID `json:"protocol"`
	Traffic
}

type PeerTraffic struct {
	Peer peer.ID `json:"peer"`
	Traffic
}

// TrafficReport is what libp2p streams moved between Since and Until.
// Protocols and Peers are two cuts of the same bytes, heaviest first.
type TrafficReport struct {
	Since     time.Time         `json:"since"`
	Until     time.Time         `json:"until"`
	Totals    Traffic           `json:"totals"`
	Protocols []ProtocolTraffic `json:"protocols"`
	Peers     []PeerTraffic     `json:"peers"`
}
