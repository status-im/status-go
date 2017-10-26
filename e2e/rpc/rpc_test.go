package rpc

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/node"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/suite"
)

func TestRPCTestSuite(t *testing.T) {
	suite.Run(t, new(RPCTestSuite))
}

type RPCTestSuite struct {
	e2e.NodeManagerTestSuite
}

func (s *RPCTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
	s.NotNil(s.NodeManager)
}

func (s *RPCTestSuite) TestCallRPC() {
	if GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
		return
	}

	for _, upstreamEnabled := range []bool{false, true} {
		nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
		s.NoError(err)

		nodeConfig.IPCEnabled = false
		nodeConfig.WSEnabled = false
		nodeConfig.HTTPHost = "" // to make sure that no HTTP interface is started

		if upstreamEnabled {
			networkURL, err := GetRemoteURLForNetworkID()
			s.NoError(err)

			nodeConfig.UpstreamConfig.Enabled = true
			nodeConfig.UpstreamConfig.URL = networkURL
		}

		nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
		s.NoError(err)
		<-nodeStarted

		rpcClient := s.NodeManager.RPCClient()
		s.NotNil(rpcClient)

		type rpcCall struct {
			inputJSON string
			validator func(resultJSON string)
		}
		var rpcCalls = []rpcCall{
			{
				`{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}`,
				func(resultJSON string) {
					expected := `{"jsonrpc":"2.0","id":67,"result":"5.0"}`
					s.Equal(expected, resultJSON)
				},
			},
			{
				`{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":64}`,
				func(resultJSON string) {
					expected := `{"jsonrpc":"2.0","id":64,"result":"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"}`
					s.Equal(expected, resultJSON)
				},
			},
			{
				`{"jsonrpc":"2.0","method":"net_version","params":[],"id":67}`,
				func(resultJSON string) {
					expected := `{"jsonrpc":"2.0","id":67,"result":"` + fmt.Sprintf("%d", GetNetworkID()) + `"}`
					s.Equal(expected, resultJSON)
				},
			},
			{
				`[{"jsonrpc":"2.0","method":"net_listening","params":[],"id":67}]`,
				func(resultJSON string) {
					expected := `[{"jsonrpc":"2.0","id":67,"result":true}]`
					s.Equal(expected, resultJSON)
				},
			},
			{
				`[{"jsonrpc":"2.0","method":"net_version","params":[],"id":67},{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":68}]`,
				func(resultJSON string) {
					expected := `[{"jsonrpc":"2.0","id":67,"result":"` + fmt.Sprintf("%d", GetNetworkID()) + `"},{"jsonrpc":"2.0","id":68,"result":"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"}]`
					s.Equal(expected, resultJSON)
				},
			},
		}

		var wg sync.WaitGroup
		for _, r := range rpcCalls {
			wg.Add(1)
			go func(r rpcCall) {
				defer wg.Done()
				resultJSON := rpcClient.CallRaw(r.inputJSON)
				r.validator(resultJSON)
			}(r)
		}

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-time.After(time.Second * 30):
			s.Fail("test timed out")
		case <-done:
		}

		stoppedNode, err := s.NodeManager.StopNode()
		s.NoError(err)
		<-stoppedNode
	}
}

// TestCallRawResult checks if returned response is a valid JSON-RPC response.
func (s *RPCTestSuite) TestCallRawResult() {
	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	<-nodeStarted

	client := s.NodeManager.RPCClient()
	s.NotNil(client)

	jsonResult := client.CallRaw(`{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}`)
	s.Equal(`{"jsonrpc":"2.0","id":67,"result":"5.0"}`, jsonResult)

	s.NodeManager.StopNode() //nolint: errcheck
}

// TestCallContextResult checks if result passed to CallContext
// is set accordingly to its underlying memory layout.
func (s *RPCTestSuite) TestCallContextResult() {
	s.StartTestNode()
	defer s.StopTestNode()

	EnsureNodeSync(s.NodeManager)

	client := s.NodeManager.RPCClient()
	s.NotNil(client)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	var balance hexutil.Big
	err := client.CallContext(ctx, &balance, "eth_getBalance", "0xAdAf150b905Cf5E6A778E553E15A139B6618BbB7", "latest")
	s.NoError(err)
	s.True(balance.ToInt().Cmp(big.NewInt(0)) > 0, "balance should be higher than 0")
}
