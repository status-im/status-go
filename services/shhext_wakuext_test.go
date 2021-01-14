package services

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/services/shhext"
	"github.com/status-im/status-go/services/wakuext"
	"github.com/status-im/status-go/waku"
	"github.com/status-im/status-go/whisper"
)

func TestShhextAndWakuextInSingleNode(t *testing.T) {
	aNode, err := node.New(&node.Config{
		P2P: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
		NoUSB: true,
	}) // in-memory node as no data dir
	require.NoError(t, err)

	// register waku and whisper services
	wakuWrapper := gethbridge.NewGethWakuWrapper(waku.New(nil, nil))
	err = aNode.Register(func(*node.ServiceContext) (node.Service, error) {
		return gethbridge.GetGethWakuFrom(wakuWrapper), nil
	})
	require.NoError(t, err)
	whisperWrapper := gethbridge.NewGethWhisperWrapper(whisper.New(nil))
	err = aNode.Register(func(*node.ServiceContext) (node.Service, error) {
		return gethbridge.GetGethWhisperFrom(whisperWrapper), nil
	})
	require.NoError(t, err)

	nodeWrapper := ext.NewTestNodeWrapper(whisperWrapper, wakuWrapper)

	// register ext services
	err = aNode.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return wakuext.New(params.ShhextConfig{}, nodeWrapper, ctx, ext.EnvelopeSignalHandler{}, nil), nil
	})
	require.NoError(t, err)
	err = aNode.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return shhext.New(params.ShhextConfig{}, nodeWrapper, ctx, ext.EnvelopeSignalHandler{}, nil), nil
	})
	require.NoError(t, err)

	// start node
	err = aNode.Start()
	require.NoError(t, err)
	defer func() { require.NoError(t, aNode.Stop()) }()

	// verify the services are available
	rpc, err := aNode.Attach()
	require.NoError(t, err)
	var result string
	err = rpc.Call(&result, "shhext_echo", "shhext test")
	require.NoError(t, err)
	require.Equal(t, "shhext test", result)
	err = rpc.Call(&result, "wakuext_echo", "wakuext test")
	require.NoError(t, err)
	require.Equal(t, "wakuext test", result)
}
