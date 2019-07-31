package node

import (
	"net"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/params"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

var enode1 = "enode://f32efef2739e5135a0f9a80600b321ba4d13393a5f1d3f5f593df85919262f06c70bfa66d38507b9d79a91021f5e200ec20150592e72934c66248e87014c4317@1.1.1.1:30404"
var enode2 = "enode://f32efef2739e5135a0f9a80600b321ba4d13393a5f1d3f5f593df85919262f06c70bfa66d38507b9d79a91021f5e200ec20150592e72934c66248e87014c4317@1.1.1.1:30404"

func TestMakeNodeDefaultConfig(t *testing.T) {
	config, err := MakeTestNodeConfig(3)
	require.NoError(t, err)

	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)

	_, err = MakeNode(config, &accounts.Manager{}, db)
	require.NoError(t, err)
}

func TestMakeNodeWellFormedBootnodes(t *testing.T) {
	config, err := MakeTestNodeConfig(3)
	require.NoError(t, err)

	bootnodes := []string{
		enode1,
		enode2,
	}
	config.ClusterConfig.BootNodes = bootnodes

	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)

	_, err = MakeNode(config, &accounts.Manager{}, db)
	require.NoError(t, err)
}

func TestMakeNodeMalformedBootnodes(t *testing.T) {
	config, err := MakeTestNodeConfig(3)
	require.NoError(t, err)

	bootnodes := []string{
		enode1,
		enode2,
		"enode://badkey@3.3.3.3:30303",
	}
	config.ClusterConfig.BootNodes = bootnodes

	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)

	_, err = MakeNode(config, &accounts.Manager{}, db)
	require.NoError(t, err)
}

func TestParseNodesToNodeID(t *testing.T) {
	identity, err := crypto.GenerateKey()
	require.NoError(t, err)
	node := enode.NewV4(&identity.PublicKey, net.IP{10, 10, 10, 10}, 10, 20)
	nodeIDs := parseNodesToNodeID([]string{
		"enode://badkey@127.0.0.1:30303",
		node.String(),
	})
	require.Len(t, nodeIDs, 1)
	require.Equal(t, node.ID(), nodeIDs[0])
}

func TestNewGethNodeConfig(t *testing.T) {
	config, err := params.NewNodeConfig("", params.RopstenNetworkID)
	require.NoError(t, err)
	config.HTTPEnabled = true
	config.HTTPVirtualHosts = []string{"my.domain.com"}
	config.HTTPCors = []string{"http://my.domain.com"}

	nc, err := newGethNodeConfig(config)
	require.NoError(t, err)
	require.Equal(t, []string{"my.domain.com"}, nc.HTTPVirtualHosts)
	require.Equal(t, []string{"http://my.domain.com"}, nc.HTTPCors)
}
