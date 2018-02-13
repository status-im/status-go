package scale

import "github.com/ethereum/go-ethereum/rpc"

// Metric is single ethereum metric.
type Metric struct {
	Overall float64 `json:"Overall"`
}

// P2P is a collection of metrics from p2p module.
type P2P struct {
	InboundTraffic  Metric `json:"InboundTraffic"`
	OutboundTraffic Metric `json:"OutboundTraffic"`
}

// Metrics is a result of debug_metrics rpc call.
type Metrics struct {
	Peer2Peer P2P `json:"p2p"`
}

func getEthMetrics(url string) (rst Metrics, err error) {
	client, err := rpc.Dial(url)
	if err != nil {
		return rst, err
	}
	return rst, client.Call(&rst, "debug_metrics", true)
}
