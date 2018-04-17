package rpc

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestRPCTestSuite(t *testing.T) {
	suite.Run(t, new(RPCTestSuite))
}

type RPCTestSuite struct {
	e2e.StatusNodeTestSuite
}

func (s *RPCTestSuite) SetupTest() {
	s.StatusNode = node.New()
	s.NotNil(s.StatusNode)
}

func (s *RPCTestSuite) TestCallRPC() {
	if GetNetworkID() == params.StatusChainNetworkID {
		s.T().Skip()
	}

	for _, upstreamEnabled := range []bool{false, true} {
		nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
		s.NoError(err)

		nodeConfig.IPCEnabled = false
		nodeConfig.HTTPHost = "" // to make sure that no HTTP interface is started

		if upstreamEnabled {
			networkURL, err := GetRemoteURL()
			s.NoError(err)

			nodeConfig.UpstreamConfig.Enabled = true
			nodeConfig.UpstreamConfig.URL = networkURL
		}

		s.NoError(s.StatusNode.Start(nodeConfig))

		rpcClient := s.StatusNode.RPCClient()
		s.NotNil(rpcClient)

		type rpcCall struct {
			inputJSON string
			validator func(resultJSON string)
		}
		var rpcCalls = []rpcCall{
			{
				`{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}`,
				func(resultJSON string) {
					expected := `{"jsonrpc":"2.0","id":67,"result":"6.0"}`
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

		s.NoError(s.StatusNode.Stop())
	}
}

// TestCallRawResult checks if returned response is a valid JSON-RPC response.
func (s *RPCTestSuite) TestCallRawResult() {
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	s.NoError(s.StatusNode.Start(nodeConfig))

	client := s.StatusNode.RPCClient()
	s.NotNil(client)

	jsonResult := client.CallRaw(`{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}`)
	s.Equal(`{"jsonrpc":"2.0","id":67,"result":"6.0"}`, jsonResult)

	s.NoError(s.StatusNode.Stop())
}

// TestCallRawResultGetTransactionReceipt checks if returned response
// for a not yet mained transaction is "error":{"code":-32000,"message":"unknown transaction"}.
// Issue: https://github.com/status-im/status-go/issues/547
func (s *RPCTestSuite) TestCallRawResultGetTransactionReceipt() {
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	s.NoError(s.StatusNode.Start(nodeConfig))

	client := s.StatusNode.RPCClient()
	s.NotNil(client)

	jsonResult := client.CallRaw(`{"jsonrpc":"2.0","method":"eth_getTransactionReceipt","params":["0x0ca0d8f2422f62bea77e24ed17db5711a77fa72064cccbb8e53c53b699cd3b34"],"id":5}`)
	s.Equal(`{"jsonrpc":"2.0","id":5,"error":{"code":-32000,"message":"unknown transaction"}}`, jsonResult)

	s.NoError(s.StatusNode.Stop())
}

// TestCallContextResult checks if result passed to CallContext
// is set accordingly to its underlying memory layout.
func (s *RPCTestSuite) TestCallContextResult() {
	s.StartTestNode()
	defer s.StopTestNode()

	EnsureNodeSync(s.StatusNode.EnsureSync)

	client := s.StatusNode.RPCClient()
	s.NotNil(client)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	var balance hexutil.Big
	err := client.CallContext(ctx, &balance, "eth_getBalance", TestConfig.Account1.Address, "latest")
	s.NoError(err)
	s.True(balance.ToInt().Cmp(big.NewInt(0)) > 0, "balance should be higher than 0")
}
