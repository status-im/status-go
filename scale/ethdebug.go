package scale

import "github.com/ethereum/go-ethereum/rpc"

type Metric struct {
	Overall float64 `json:"Overall"`
}

type P2P struct {
	InboundTraffic  Metric `json:"InboundTraffic"`
	OutboundTraffic Metric `json:"OutboundTraffic"`
}

type Metrics struct {
	Peer2Peer P2P `json:"p2p"`
}

func ethMetrics(url string) (rst Metrics, err error) {
	client, err := rpc.Dial(url)
	if err != nil {
		return rst, err
	}
	return rst, client.Call(&rst, "debug_metrics", true)
}
