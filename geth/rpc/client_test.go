package rpc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	. "github.com/status-im/status-go/geth/testing"
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
	BaseTestSuite
}

func (s *RPCTestSuite) SetupTest() {
	require := s.Require()

	nodeManager := node.NewNodeManager()
	require.NotNil(nodeManager)
	s.NodeManager = nodeManager
}

func (s *RPCTestSuite) TestNewClient() {
	require := s.Require()

	config, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	nodeStarted, err := s.NodeManager.StartNode(config)
	require.NoError(err)
	require.NotNil(config)

	<-nodeStarted

	node, err := s.NodeManager.Node()
	require.NoError(err)

	// upstream disabled, local node ok
	_, err = rpc.NewClient(node, config.UpstreamConfig)
	require.NoError(err)

	// upstream enabled with incorrect URL, local node ok
	upstreamBad := config.UpstreamConfig
	upstreamBad.Enabled = true
	upstreamBad.URL = "///__httphh://///incorrect_urlxxx"
	_, err = rpc.NewClient(node, upstreamBad)
	require.NotNil(err)

	// upstream enabled with correct URL, local node ok
	upstreamGood := config.UpstreamConfig
	upstreamGood.Enabled = true
	upstreamGood.URL = "http://example.com/rpc"
	_, err = rpc.NewClient(node, upstreamGood)
	require.Nil(err)

	// upstream disabled, local node failed (stopped)
	nodeStopped, err := s.NodeManager.StopNode()
	require.NoError(err)

	<-nodeStopped

	_, err = rpc.NewClient(node, config.UpstreamConfig)
	require.NotNil(err)
}

func (s *RPCTestSuite) TestRPCSendTransaction() {
	require := s.Require()
	expectedResponse := []byte(`{"jsonrpc":"2.0","id":10,"result":"3434=done"}`)

	// httpRPCServer will serve as an upstream server accepting transactions.
	httpRPCServer := httptest.NewServer(service{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close() //nolint: errcheck

			var txReq txRequest
			err := json.NewDecoder(r.Body).Decode(&txReq)
			require.NoError(err)

			if txReq.Method == "eth_getTransactionCount" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc": "2.0", "result": "0x434"}`)) //nolint: errcheck
				return
			}

			payload := ([]byte)(txReq.Payload)

			var bu []interface{}
			jserr := json.Unmarshal(payload, &bu)
			require.NoError(jserr)
			require.Len(bu, 1)
			require.IsType(bu[0], (map[string]interface{})(nil))

			w.WriteHeader(http.StatusOK)
			w.Write(expectedResponse) //nolint: errcheck
		},
	})

	s.StartTestNode(params.RopstenNetworkID, WithUpstream(httpRPCServer.URL))
	defer s.StopTestNode()

	rpcClient := s.NodeManager.RPCClient()
	require.NotNil(rpcClient)

	response := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"id":10,
		"method": "eth_sendTransaction",
		"params": [{
			"from":     "` + TestConfig.Account1.Address + `",
			"to":       "` + TestConfig.Account2.Address + `",
			"value":    "0x200",
			"nonce":    "0x100",
			"data":     "Will-power",
			"gasPrice": "0x4a817c800",
			"gasLimit": "0x5208",
			"chainId":  3391
		}]
	}`)
	require.Equal(response, string(expectedResponse))
}

func (s *RPCTestSuite) TestCallRPC() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	for _, upstreamEnabled := range []bool{false, true} {
		s.T().Logf("TestCallRPC with upstream: %t", upstreamEnabled)

		nodeConfig, err := MakeTestNodeConfig(params.RinkebyNetworkID)
		require.NoError(err)

		nodeConfig.IPCEnabled = false
		nodeConfig.WSEnabled = false
		nodeConfig.HTTPHost = "" // to make sure that no HTTP interface is started

		if upstreamEnabled {
			nodeConfig.UpstreamConfig.Enabled = true
			nodeConfig.UpstreamConfig.URL = "https://rinkeby.infura.io/nKmXgiFgc2KqtoQ8BCGJ"
		}

		nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
		require.NoError(err)

		<-nodeStarted

		rpcClient := s.NodeManager.RPCClient()
		require.NotNil(rpcClient)

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
			s.NodeManager.StopNode() //nolint: errcheck
			s.FailNow("test timed out")
		case <-done:
			s.NodeManager.StopNode() //nolint: errcheck
		}
	}
}

// TestCallRawResult checks if returned response is a valid JSON-RPC response.
func (s *RPCTestSuite) TestCallRawResult() {
	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)

	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	defer s.NodeManager.StopNode() //nolint: errcheck

	<-nodeStarted

	client := s.NodeManager.RPCClient()

	jsonResult := client.CallRaw(`{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}`)
	s.Equal(`{"jsonrpc":"2.0","id":67,"result":"5.0"}`, jsonResult)
}

// TestCallContextResult checks if result passed to CallContext
// is set accordingly to its underlying memory layout.
func (s *RPCTestSuite) TestCallContextResult() {
	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	s.NoError(err)

	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	s.NoError(err)
	defer s.NodeManager.StopNode() //nolint: errcheck

	<-nodeStarted

	client := s.NodeManager.RPCClient()

	var blockNumber hexutil.Uint
	err = client.CallContext(context.Background(), &blockNumber, "eth_blockNumber")
	s.NoError(err)
	s.True(blockNumber > 0, "blockNumber should be higher than 0")
}
