package common

import (
	"context"

	"github.com/status-im/status-go/wakuv2/common"
	wakuv2protocol "github.com/waku-org/go-waku/waku/v2/protocol"

	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/libp2p/go-libp2p/core/metrics"
)

type ProtocolStatsMap map[protocol.ID]metrics.Stats

type SentEnvelope struct {
	Envelope      *wakuv2protocol.Envelope
	PublishMethod common.PublishMethod
}

type ErrorSendingEnvelope struct {
	Error        error
	SentEnvelope SentEnvelope
}

type ITelemetryClient interface {
	PushReceivedEnvelope(ctx context.Context, receivedEnvelope *wakuv2protocol.Envelope)
	PushSentEnvelope(ctx context.Context, sentEnvelope SentEnvelope)
	PushErrorSendingEnvelope(ctx context.Context, errorSendingEnvelope ErrorSendingEnvelope)
	PushPeerCount(ctx context.Context, peerCount int)
	PushPeerConnFailures(ctx context.Context, peerConnFailures map[string]int)
	PushProtocolStats(ctx context.Context, stats ProtocolStatsMap)
}
