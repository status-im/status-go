package main

import (
	"flag"
	"log"
	"time"

	sdk "github.com/status-im/status-go-sdk"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/notifier"
)

var (
	clusterConfigFile   = flag.String("clusterconfig", "", "Cluster configuration file")
	dataDir             = flag.String("datadir", params.DataDir, "Data directory for the databases and keystore")
	networkID           = flag.Int("networkid", params.RopstenNetworkID, "Network identifier (integer, 1=Homestead, 3=Ropsten, 4=Rinkeby, 777=StatusChain)")
	pushNotificationURI = flag.String("uri", "localhost:3000", "Push notification uri")
	discoveryTopic      = flag.String("topic", "notifier", "Discovery topic")
)

func main() {
	var n *notifier.Notifier
	var backend *api.StatusBackend

	flag.Parse()

	address := *pushNotificationURI
	if n = notifier.New(address); n == nil {
		panic("Couldn't connect to push notification server on " + address)
	}
	defer func() {
		if err := n.Close(); err != nil {
			log.Println("Error closing connection : " + err.Error())
		}
	}()

	if backend = notifier.NewStatusBackend(*dataDir, *clusterConfigFile, uint64(*networkID)); backend == nil {
		panic("Couldn't setup the node")
	}

	t := *discoveryTopic
	m, err := notifier.NewMessenger(newRPCClient(backend), n, t, 5*time.Second)
	if m == nil {
		panic(err)
	}
	/*
		if err = m.BroadcastAvailability(); err != nil {
			panic(err)
		}
	*/

	// go func() {
	_ = m.ManageRegistrations()
	//}()

	if backend.StatusNode().GethNode() != nil {
		backend.StatusNode().GethNode().Wait()
	}
}

type rpcClient struct {
	b *api.StatusBackend
}

func newRPCClient(b *api.StatusBackend) sdk.RPCClient {
	return &rpcClient{b: b}
}

func (c *rpcClient) Call(request interface{}) (response interface{}, err error) {
	response = c.b.CallPrivateRPC(request.(string))
	return
}
