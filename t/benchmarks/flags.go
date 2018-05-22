package benchmarks

import (
	"flag"

	"github.com/ethereum/go-ethereum/p2p/discover"
)

var (
	// general
	peerURL = flag.String("peerurl", "", "Peer raw URL to which send messages")
	// mailserver tests
	ccyPeers = flag.Int("ccypeers", 1, "Number of concurrent peers requesting messages")
	// messages tests
	msgPass      = flag.String("msgpass", "message-pass", "Password to create sym key from")
	msgCount     = flag.Int64("msgcount", 100, "Number of messages to send")
	msgSize      = flag.Int64("msgsize", int64(1024), "Message size in bytes")
	msgBatchSize = flag.Int64("msgbatchsize", int64(20), "Number of messages to send in a batch")
)

var peerEnode *discover.Node

func init() {
	flag.Parse()

	peerEnode = discover.MustParseNode(*peerURL)
}
