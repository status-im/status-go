package rpc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/status-im/status-go/geth/log"
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
	require.IsType(rpc.UpstreamError{}, err)

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
	require.IsType(rpc.LocalError{}, err)
}

func (s *RPCTestSuite) TestRPCSendTransaction() {
	require := s.Require()
	expectedResponse := []byte(`{"jsonrpc":"2.0","id":10,"result":"3434=done"}`)

	// httpRPCServer will serve as an upstream server accepting transactions.
	httpRPCServer := httptest.NewServer(service{
		Handler: func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			var txReq txRequest
			err := json.NewDecoder(r.Body).Decode(&txReq)
			require.NoError(err)

			if txReq.Method == "eth_getTransactionCount" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc": "2.0", "result": "0x434"}`))
				return
			}

			payload := ([]byte)(txReq.Payload)

			var bu []interface{}
			jserr := json.Unmarshal(payload, &bu)
			require.NoError(jserr)
			require.Len(bu, 1)
			require.IsType(bu[0], (map[string]interface{})(nil))

			w.WriteHeader(http.StatusOK)
			w.Write(expectedResponse)
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

	nodeConfig, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	nodeConfig.IPCEnabled = false
	nodeConfig.WSEnabled = false
	nodeConfig.HTTPHost = "" // to make sure that no HTTP interface is started

	nodeStarted, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)
	require.NotNil(nodeConfig)

	defer s.NodeManager.StopNode()

	<-nodeStarted

	rpcClient := s.NodeManager.RPCClient()
	require.NotNil(rpcClient)

	progress := make(chan struct{}, 25)
	type rpcCall struct {
		inputJSON string
		validator func(resultJSON string)
	}
	var rpcCalls = []rpcCall{
		{
			`{"jsonrpc":"2.0","method":"eth_sendTransaction","params":[{
				"from": "0xb60e8dd61c5d32be8058bb8eb970870f07233155",
				"to": "0xd46e8dd67c5d32be8058bb8eb970870f07244567",
				"gas": "0x76c0",
				"gasPrice": "0x9184e72a000",
				"value": "0x9184e72a",
				"data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"}],
				"id": 1
			}`,
			func(resultJSON string) {
				log.Info("eth_sendTransaction")
				progress <- struct{}{}
			},
		},
		{
			`{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}`,
			func(resultJSON string) {
				expected := `{"jsonrpc":"2.0","id":67,"result":"5.0"}`
				s.Equal(expected, resultJSON)
				progress <- struct{}{}
			},
		},
		{
			`{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":64}`,
			func(resultJSON string) {
				expected := `{"jsonrpc":"2.0","id":64,"result":"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"}`
				s.Equal(expected, resultJSON)
				progress <- struct{}{}
			},
		},
		{
			`{"jsonrpc":"2.0","method":"net_version","params":[],"id":67}`,
			func(resultJSON string) {
				expected := `{"jsonrpc":"2.0","id":67,"result":"4"}`
				s.Equal(expected, resultJSON)
				progress <- struct{}{}
			},
		},
	}

	cnt := len(rpcCalls) - 1 // send transaction blocks up until complete/discarded/times out
	for _, r := range rpcCalls {
		go func(r rpcCall) {
			resultJSON := rpcClient.CallRaw(r.inputJSON)
			r.validator(resultJSON)
		}(r)
	}

	for range progress {
		cnt -= 1
		if cnt <= 0 {
			break
		}
	}
}
