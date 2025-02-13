package waku

import (
	"time"
)

var DefaultWakuConfig WakuConfig

func init() {

	udpPort, _, err1 := GetFreePortIfNeeded(0, 0)
	tcpPort, _, err2 := GetFreePortIfNeeded(0, 0)

	if err1 != nil || err2 != nil {
		Error("Failed to get free ports %v %v", err1, err2)
	}

	DefaultWakuConfig = WakuConfig{
		Relay:           false,
		LogLevel:        "DEBUG",
		Discv5Discovery: true,
		ClusterID:       16,
		Shards:          []uint16{64},
		PeerExchange:    false,
		Store:           false,
		Filter:          false,
		Lightpush:       false,
		Discv5UdpPort:   udpPort,
		TcpPort:         tcpPort,
	}
}

const ConnectPeerTimeout = 10 * time.Second //default timeout for node to connect to another node

var DefaultPubsubTopic = "/waku/2/rs/16/64"
var (
	MinPort = 1024  // Minimum allowable port (exported)
	MaxPort = 65535 // Maximum allowable port (exported)
)
