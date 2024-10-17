package node

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/t/utils"
)

func TestMakeNodeDefaultConfig(t *testing.T) {
	utils.Init()
	config, err := utils.MakeTestNodeConfig(3)
	require.NoError(t, err)

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
	config, err := params.NewNodeConfig("", params.SepoliaNetworkID)
	require.NoError(t, err)
	config.HTTPEnabled = true
	config.HTTPVirtualHosts = []string{"my.domain.com"}
	config.HTTPCors = []string{"http://my.domain.com"}

	nc, err := newGethNodeConfig(config)
	require.NoError(t, err)
	require.Equal(t, []string{"my.domain.com"}, nc.HTTPVirtualHosts)
	require.Equal(t, []string{"http://my.domain.com"}, nc.HTTPCors)
}
