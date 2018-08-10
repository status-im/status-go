package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/params"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/status-go/discovery"
)

var (
	laddr           = flag.String("laddr", "0.0.0.0:31143", "Listening address for discovery v5.")
	verbosity       = flag.String("v", "info", "Logger verbosity")
	rendezvousNodes = StringSlice{}
	bootnodes       = StringSlice{}
	topics          = StringSlice{}
	les             = flag.Int("les", 0, "Proxy les topic for a given network.")
	useEthereum     = flag.Bool("use-ethereum-boot", false, "If true ethereum bootnodes will be used")
)

func main() {
	flag.Var(&rendezvousNodes, "rendezvous-node", "Rendezvous server.")
	flag.Var(&bootnodes, "bootnode", "Discovery v5 node.")
	flag.Var(&topics, "topic", "Topic that will be proxied")
	flag.Parse()

	level, err := log.LvlFromString(strings.ToLower(*verbosity))
	if err != nil {
		panic(fmt.Errorf("unable to get logger level from string %s: %v", *verbosity, err))
	}
	filteredHandler := log.LvlFilterHandler(level, log.StderrHandler)
	log.Root().SetHandler(filteredHandler)

	if t := lesTopic(*les); len(t) != 0 {
		topics = append(topics, t)
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		log.Crit("unable to generate a key", "error", err)
	}
	rst := []string(bootnodes)
	if *useEthereum {
		rst = append(rst, params.DiscoveryV5Bootnodes...)
	}
	v5 := discovery.NewDiscV5(key, *laddr, parseNodesV5(rst))
	if err := v5.Start(); err != nil {
		log.Crit("unable to start discovery v5", "address", *laddr, "error", err)
	}
	proxy := discovery.NewProxy(v5, parseMultiaddrs(rendezvousNodes))
	stop := make(chan struct{})
	defer close(stop)
	for _, t := range topics {
		log.Info("proxying records for", "topic", t, "bootnodes", rst, "rendezvous servers", rendezvousNodes)
		t := t
		go proxy.Run(t, stop)
	}
	select {}
}

func lesTopic(netid int) string {
	switch netid {
	case 1:
		return "LES2@" + common.Bytes2Hex(params.MainnetGenesisHash.Bytes()[:8])
	case 4:
		return "LES2@" + common.Bytes2Hex(params.TestnetGenesisHash.Bytes()[:8])
	default:
		return ""
	}
}

func parseMultiaddrs(nodes []string) []ma.Multiaddr {
	rst := make([]ma.Multiaddr, len(nodes))
	for i := range nodes {
		var err error
		rst[i], err = ma.NewMultiaddr(nodes[i])
		if err != nil {
			log.Crit("unable to parse mutliaddr", "source", nodes[i], "error", err)
		}
	}
	return rst
}

func parseNodesV5(enodes []string) []*discv5.Node {
	var nodes []*discv5.Node
	for _, enode := range enodes {
		parsedPeer, err := discv5.ParseNode(enode)
		if err == nil {
			nodes = append(nodes, parsedPeer)
		} else {
			log.Crit("Failed to parse enode", "enode", enode, "err", err)
		}
	}
	return nodes
}
