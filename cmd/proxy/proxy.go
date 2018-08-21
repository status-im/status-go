package main

import (
	"flag"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/params"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/status-go/discovery"
	sparams "github.com/status-im/status-go/params"
)

var (
	laddr           = flag.String("laddr", "0.0.0.0:31143", "Listening address for discovery v5.")
	verbosity       = flag.String("v", "info", "Logger verbosity")
	rendezvousNodes = StringSlice{}
	bootnodes       = StringSlice{}
	topics          = StringSlice{}
	les             = IntSlice{}
	useEthereum     = flag.Bool("use-ethereum-boot", false, "If true ethereum bootnodes will be used")
)

func main() {
	flag.Var(&rendezvousNodes, "rendezvous-node", "Rendezvous server.")
	flag.Var(&bootnodes, "bootnode", "Discovery v5 node.")
	flag.Var(&les, "les", "Proxy les topic for a given network.")
	flag.Var(&topics, "topic", "Topic that will be proxied")
	flag.Parse()

	level, err := log.LvlFromString(strings.ToLower(*verbosity))
	if err != nil {
		panic(fmt.Errorf("unable to get logger level from string %s: %v", *verbosity, err))
	}
	filteredHandler := log.LvlFilterHandler(level, log.StderrHandler)
	log.Root().SetHandler(filteredHandler)

	for _, net := range les {
		if t := sparams.LesTopic(net); len(t) != 0 {
			topics = append(topics, t)
		}
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
	rendezvousServers := parseMultiaddrs(rendezvousNodes)
	var wg sync.WaitGroup
	stop := make(chan struct{})
	defer close(stop)
	for _, t := range topics {
		log.Info("proxying records for", "topic", t, "bootnodes", rst, "rendezvous servers", rendezvousNodes)
		t := t
		wg.Add(1)
		go func() {
			if err := discovery.ProxyToRendezvous(v5, rendezvousServers, t, stop); err != nil {
				log.Error("proxying to rendezvous servers failed", "servers", rendezvousNodes, "topic", t, "error", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func parseMultiaddrs(nodes []string) []ma.Multiaddr {
	var (
		rst = make([]ma.Multiaddr, len(nodes))
		err error
	)
	for i := range nodes {
		rst[i], err = ma.NewMultiaddr(nodes[i])
		if err != nil {
			log.Crit("unable to parse mutliaddr", "source", nodes[i], "error", err)
		}
	}
	return rst
}

func parseNodesV5(nodes []string) []*discv5.Node {
	var (
		rst = make([]*discv5.Node, len(nodes))
		err error
	)
	for i := range nodes {
		rst[i], err = discv5.ParseNode(nodes[i])
		if err != nil {
			log.Crit("Failed to parse enode", "source", nodes[i], "err", err)
		}
	}
	return rst
}
