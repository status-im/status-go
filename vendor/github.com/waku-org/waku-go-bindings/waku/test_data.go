package waku

import (
	"time"

	"github.com/waku-org/waku-go-bindings/waku/common"
)

var DefaultWakuConfig common.WakuConfig

func init() {

	DefaultWakuConfig = common.WakuConfig{
		Relay:           false,
		LogLevel:        "DEBUG",
		Discv5Discovery: true,
		ClusterID:       16,
		Shards:          []uint16{64},
		PeerExchange:    false,
		Store:           false,
		Filter:          false,
		Lightpush:       false,
		Discv5UdpPort:   0,
		TcpPort:         0,
	}
}

const ConnectPeerTimeout = 10 * time.Second //default timeout for node to connect to another node
const DefaultTimeOut = 3 * time.Second

var DefaultPubsubTopic = "/waku/2/rs/16/64"
var (
	MinPort = 1024  // Minimum allowable port (exported)
	MaxPort = 65535 // Maximum allowable port (exported)
)
