package proxy_test

import (
	"testing"
	"time"

	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/proxy"
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

//==================================================================================================

func TestRPCRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RPCRouterTestSuite))
}

type RPCRouterTestSuite struct {
	BaseTestSuite
}

func (s *RPCRouterTestSuite) SetupTest() {
	require := s.Require()

	nodeman := node.NewNodeManager()
	acctman := node.NewAccountManager(nodeman)

	s.NodeManager = proxy.NewRPCRouter(nodeman, acctman)

	require.NotNil(s.NodeManager)
	require.IsType(&proxy.RPCRouter{}, s.NodeManager)
}

func (s *RPCRouterTestSuite) TestSendTransaction() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	odFunc := otto.FunctionCall{
		Otto: otto.New(),
		This: otto.NullValue(),
	}

	request := common.RPCCall{
		ID:     65454545334343,
		Method: "eth_sendTransaction",
		Params: []interface{}{
			map[string]interface{}{
				"from":     TestConfig.Account1.Address,
				"to":       "0xe410006cad020e3690c8ba21ed8b0f065dde2453",
				"value":    "0x2",
				"nonce":    "0x1",
				"data":     "Will-power",
				"gasPrice": "0x4a817c800",
				"gasLimit": "0x5208",
				"chainId":  3391,
			},
		},
	}

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	nodeConfig.UpstreamConfig.Enabled = true

	rpcService := new(service)
	rpcService.Handler = func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var txReq txRequest

		if err := json.NewDecoder(r.Body).Decode(&txReq); err != nil {
			require.NoError(err)
			return
		}

		switch txReq.Method {
		case "eth_getBlockTransactionCountByHash":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc": "2.0", "status":200, "result": "0x434"}`))
			return
		default:
		}

		payload := ([]byte)(txReq.Payload)

		var bu []interface{}

		jserr := json.Unmarshal(payload, &bu)
		require.NoError(jserr)
		require.NotNil(bu)
		require.Len(bu, 1)

		buElem, ok := bu[0].(string)
		require.Equal(ok, true)

		decoded, err := hexutil.Decode(buElem)
		require.NoError(err)
		require.NotNil(decoded)

		var tx types.Transaction
		decodeErr := rlp.DecodeBytes(decoded, &tx)
		require.NoError(decodeErr)

		require.Equal(tx.ChainId().Int64(), int64(nodeConfig.NetworkID))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc": "2.0", "status":200, "result": "3434=done"}`))
	}

	httpRPCServer := httptest.NewServer(rpcService)
	nodeConfig.UpstreamConfig.URL = httpRPCServer.URL

	// Start NodeManagers Node
	started, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)

	select {
	case <-started:
		break
	case <-time.After(1 * time.Second):
		require.Fail("Failed to start NodeManager")
		break
	}

	defer s.NodeManager.StopNode()

	client, err := s.NodeManager.RPCClient()
	require.NoError(err)
	require.NotNil(client)

	rpcNodeManager := s.NodeManager.(common.RPCNodeManager)
	accountManager := rpcNodeManager.Account()

	selectErr := accountManager.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	require.NoError(selectErr)

	res, err := rpcNodeManager.Exec(request, odFunc)
	require.NoError(err)

	result, err := res.Get("result")
	require.NoError(err)
	require.NotNil(result)

	exported, err := result.Export()
	require.NoError(err)

	rawJSON, ok := exported.(json.RawMessage)
	require.True(ok, "Expected raw json payload")
	require.Equal(string(rawJSON), "\"3434=done\"")
}

// func (s *RPCRouterTestSuite) TestMainnetAcceptance() {
// 	require := s.Require()
// 	require.NotNil(s.NodeManager)

// 	odFunc := otto.FunctionCall{
// 		Otto: otto.New(),
// 		This: otto.NullValue(),
// 	}

// 	request := common.RPCCall{
// 		ID:     65454545334343,
// 		Method: "eth_sendTransaction",
// 		Params: []interface{}{
// 			map[string]interface{}{
// 				"from":     TestConfig.Account1.Address,
// 				"to":       "0xe410006cad020e3690c8ba21ed8b0f065dde2453",
// 				"value":    "0x2",
// 				"nonce":    "0x1",
// 				"data":     "Will-power",
// 				"gasPrice": "0x4a817c800",
// 				"gasLimit": "0x5208",
// 				"chainId":  3391,
// 			},
// 		},
// 	}

// 	nodeConfig, err := MakeTestNodeConfig(params.UpstreamMainNetEthereumNetworkURL)
// 	require.NoError(err)

// 	nodeConfig.UpstreamConfig.Enabled = true

// 	// Start NodeManagers Node
// 	started, err := s.NodeManager.StartNode(nodeConfig)
// 	require.NoError(err)

// 	select {
// 	case <-started:
// 		break
// 	case <-time.After(1 * time.Second):
// 		require.Fail("Failed to start NodeManager")
// 		break
// 	}

// 	defer s.NodeManager.StopNode()

// 	client, err := s.NodeManager.RPCClient()
// 	require.NoError(err)
// 	require.NotNil(client)

// 	rpcNodeManager := s.NodeManager.(common.RPCNodeManager)
// 	accountManager := rpcNodeManager.Account()

// 	selectErr := accountManager.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
// 	require.NoError(selectErr)

// 	res, err := rpcNodeManager.Exec(request, odFunc)
// 	require.NoError(err)

// 	_, err = res.Get("hash")
// 	require.NoError(err)
// }

// func (s *RPCRouterTestSuite) TestRinkebyAcceptance() {
// 	require := s.Require()
// 	require.NotNil(s.NodeManager)

// 	odFunc := otto.FunctionCall{
// 		Otto: otto.New(),
// 		This: otto.NullValue(),
// 	}

// 	request := common.RPCCall{
// 		ID:     65454545334343,
// 		Method: "eth_sendTransaction",
// 		Params: []interface{}{
// 			map[string]interface{}{
// 				"from":     TestConfig.Account1.Address,
// 				"to":       "0xe410006cad020e3690c8ba21ed8b0f065dde2453",
// 				"value":    "0x2",
// 				"nonce":    "0x1",
// 				"data":     "Will-power",
// 				"gasPrice": "0x4a817c800",
// 				"gasLimit": "0x5208",
// 				"chainId":  3391,
// 			},
// 		},
// 	}

// 	nodeConfig, err := MakeTestNodeConfig(params.UpstreamRinkebyEthereumNetworkURL)
// 	require.NoError(err)

// 	nodeConfig.UpstreamConfig.Enabled = true

// 	// Start NodeManagers Node
// 	started, err := s.NodeManager.StartNode(nodeConfig)
// 	require.NoError(err)

// 	select {
// 	case <-started:
// 		break
// 	case <-time.After(1 * time.Second):
// 		require.Fail("Failed to start NodeManager")
// 		break
// 	}

// 	defer s.NodeManager.StopNode()

// 	client, err := s.NodeManager.RPCClient()
// 	require.NoError(err)
// 	require.NotNil(client)

// 	rpcNodeManager := s.NodeManager.(common.RPCNodeManager)
// 	accountManager := rpcNodeManager.Account()

// 	selectErr := accountManager.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
// 	require.NoError(selectErr)

// 	res, err := rpcNodeManager.Exec(request, odFunc)
// 	require.NoError(err)

// 	_, err = res.Get("hash")
// 	require.NoError(err)
// }

// 	require.NotNil(s.NodeManager)

// 	odFunc := otto.FunctionCall{
// 		Otto: otto.New(),
// 		This: otto.NullValue(),
// 	}

// 	request := common.RPCCall{
// 		ID:     65454545334343,
// 		Method: "eth_sendTransaction",
// 		Params: []interface{}{
// 			map[string]interface{}{
// 				"from":     TestConfig.Account1.Address,
// 				"to":       "0xe410006cad020e3690c8ba21ed8b0f065dde2453",
// 				"value":    "0x2",
// 				"nonce":    "0x1",
// 				"data":     "Will-power",
// 				"gasPrice": "0x4a817c800",
// 				"gasLimit": "0x5208",
// 				"chainId":  3391,
// 			},
// 		},
// 	}

// 	nodeConfig, err := MakeTestNodeConfig(params.UpstreamRopstenEthereumNetworkURL)
// 	require.NoError(err)

// 	nodeConfig.UpstreamConfig.Enabled = true

// 	// Start NodeManagers Node
// 	started, err := s.NodeManager.StartNode(nodeConfig)
// 	require.NoError(err)

// 	select {
// 	case <-started:
// 		break
// 	case <-time.After(1 * time.Second):
// 		require.Fail("Failed to start NodeManager")
// 		break
// 	}

// 	defer s.NodeManager.StopNode()

// 	client, err := s.NodeManager.RPCClient()
// 	require.NoError(err)
// 	require.NotNil(client)

// 	rpcNodeManager := s.NodeManager.(common.RPCNodeManager)
// 	accountManager := rpcNodeManager.Account()

// 	selectErr := accountManager.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
// 	require.NoError(selectErr)

// 	res, err := rpcNodeManager.Exec(request, odFunc)
// 	require.NoError(err)

// 	_, err = res.Get("hash")
// 	require.NoError(err)
