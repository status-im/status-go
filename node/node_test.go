package node

import (
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"testing"
)

var enode1 = "enode://f32efef2739e5135a0f9a80600b321ba4d13393a5f1d3f5f593df85919262f06c70bfa66d38507b9d79a91021f5e200ec20150592e72934c66248e87014c4317@1.1.1.1:30404"
var enode2 = "enode://f32efef2739e5135a0f9a80600b321ba4d13393a5f1d3f5f593df85919262f06c70bfa66d38507b9d79a91021f5e200ec20150592e72934c66248e87014c4317@1.1.1.1:30404"

func TestMakeNodeDefaultConfig(t *testing.T) {
	config, err := MakeTestNodeConfig(3)
	require.NoError(t, err)

	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	require.NoError(t, err)

	_, err = MakeNode(config, db)
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

	_, err = MakeNode(config, db)
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

	_, err = MakeNode(config, db)
	require.NoError(t, err)
}
