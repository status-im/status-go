package wakuv2

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/protocol"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"

	"github.com/waku-org/go-waku/waku/v2/protocol/filter"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

type BandwidthTelemetryClient struct {
	serverURL  string
	httpClient *http.Client
	hostID     string
	logger     *zap.Logger
}

func NewBandwidthTelemetryClient(logger *zap.Logger, serverURL string) *BandwidthTelemetryClient {
	return &BandwidthTelemetryClient{
		serverURL:  serverURL,
		httpClient: &http.Client{Timeout: time.Minute},
		hostID:     uuid.NewString(),
		logger:     logger.Named("bandwidth-telemetry"),
	}
}

func getStatsPerProtocol(protocolID protocol.ID, stats map[protocol.ID]metrics.Stats) map[string]interface{} {
	return map[string]interface{}{
		"rateIn":   stats[protocolID].RateIn,
		"rateOut":  stats[protocolID].RateOut,
		"totalIn":  stats[protocolID].TotalIn,
		"totalOut": stats[protocolID].TotalOut,
	}
}

func (c *BandwidthTelemetryClient) getTelemetryRequestBody(stats map[protocol.ID]metrics.Stats) map[string]interface{} {
	return map[string]interface{}{
		"hostID":           c.hostID,
		"relay":            getStatsPerProtocol(relay.WakuRelayID_v200, stats),
		"store":            getStatsPerProtocol(legacy_store.StoreID_v20beta4, stats),
		"filter-push":      getStatsPerProtocol(filter.FilterPushID_v20beta1, stats),
		"filter-subscribe": getStatsPerProtocol(filter.FilterSubscribeID_v20beta1, stats),
		"lightpush":        getStatsPerProtocol(lightpush.LightPushID_v20beta1, stats),
	}
}

func (c *BandwidthTelemetryClient) PushProtocolStats(stats map[protocol.ID]metrics.Stats) {
	defer gocommon.LogOnPanic()
	url := fmt.Sprintf("%s/protocol-stats", c.serverURL)
	body, _ := json.Marshal(c.getTelemetryRequestBody(stats))
	_, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.logger.Error("Error sending message to telemetry server", zap.Error(err))
	}
}
