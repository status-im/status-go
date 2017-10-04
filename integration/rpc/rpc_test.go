package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/integration"
	"github.com/stretchr/testify/suite"
)

type txRequest struct {
	Method  string          `json:"method"`
	Version string          `json:"jsonrpc"`
	ID      int             `json:"id,omitempty"`
	Payload json.RawMessage `json:"params,omitempty"`
}

type service struct {
	Handler http.HandlerFunc
}

func (s service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Handler(w, r)
}

func TestRPCTestSuite(t *testing.T) {
	suite.Run(t, new(RPCTestSuite))
}

type RPCTestSuite struct {
	integration.NodeManagerTestSuite
}

func (s *RPCTestSuite) SetupTest() {
	s.NodeManager = node.NewNodeManager()
	s.NotNil(s.NodeManager)
}

func (s *RPCTestSuite) TestCallRPC() {
	for _, upstreamEnabled := range []bool{false, true} {
		nodeConfig, err := integration.MakeTestNodeConfig(params.RinkebyNetworkID)
		s.NoError(err)

		nodeConfig.IPCEnabled = false
		nodeConfig.WSEnabled = false
		nodeConfig.HTTPHost = "" // to make sure that no HTTP interface is started

		if upstreamEnabled {
			nodeConfig.UpstreamConfig.Enabled = true
			nodeConfig.UpstreamConfig.URL = "https://rinkeby.infura.io/nKmXgiFgc2KqtoQ8BCGJ"
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
					expected := `{"jsonrpc":"2.0","id":67,"result":"4"}`
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
					expected := `[{"jsonrpc":"2.0","id":67,"result":"4"},{"jsonrpc":"2.0","id":68,"result":"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"}]`
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
	nodeConfig, err := integration.MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)

	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	<-nodeStarted

	client := s.NodeManager.RPCClient()
	s.NotNil(client)

	jsonResult := client.CallRaw(`{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}`)
	s.Equal(`{"jsonrpc":"2.0","id":67,"result":"5.0"}`, jsonResult)

	s.NodeManager.StopNode()
}

// TestCallContextResult checks if result passed to CallContext
// is set accordingly to its underlying memory layout.
func (s *RPCTestSuite) TestCallContextResult() {
	s.StartTestNode(
		params.RopstenNetworkID,
		integration.WithUpstream("https://ropsten.infura.io/nKmXgiFgc2KqtoQ8BCGJ"),
	)
	defer s.StopTestNode()

	client := s.NodeManager.RPCClient()
	s.NotNil(client)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var blockNumber hexutil.Uint
	err := client.CallContext(ctx, &blockNumber, "eth_blockNumber")
	s.NoError(err)
	s.True(blockNumber > 0, "blockNumber should be higher than 0")
}
