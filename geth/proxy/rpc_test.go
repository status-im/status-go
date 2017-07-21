package proxy_test

import (
	"context"
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

	// create a new client and issue a request.
	// client, err := s.NodeManager.RPCClient()
	// require.NoError(err)
	// require.NotNil(client)

}

func (s *RPCRouterTestSuite) TestRPCClientConnection() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	// validate default state of UpstreamConfig.Enable.
	require.NotEqual(nodeConfig.UpstreamConfig.Enabled, true)
	require.NotEmpty(nodeConfig.UpstreamConfig.URL)
	require.Equal(nodeConfig.UpstreamConfig.URL, params.UpstreamRopstenEthereumNetworkURL)

	rpcService := service{Handler: func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var req map[string]interface{}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			require.NoError(err)
			return
		}

		method, ok := req["method"]
		require.NotEqual(ok, false)
		require.IsType((string)(""), method)
		require.Equal(method, "eth_swapspace")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc": "2.0", "status":200, "result": "3434=done"}`))
	}}

	httpRPCServer := httptest.NewServer(rpcService)

	nodeConfig.UpstreamConfig.URL = httpRPCServer.URL
	nodeConfig.UpstreamConfig.Enabled = true

	started, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)

	// Attempt to find out if we started well.
	select {
	case <-started:
		break
	case <-time.After(1 * time.Second):
		s.T().Fatal("failed to start node manager")
		break
	}

	defer s.NodeManager.StopNode()

	// create a new client and issue a request.
	client, err := s.NodeManager.RPCClient()
	require.NoError(err)
	require.NotNil(client)

	ctx, canceller := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))

	defer canceller()

	var result interface{}

	// Ignore error since am only interested in reception here.
	err2 := client.CallContext(ctx, &result, "eth_swapspace", "Lock")
	require.NoError(err2)
}

func (s *RPCRouterTestSuite) TestSendTransaction() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	// validate default state of UpstreamConfig.Enable.
	require.NotEqual(nodeConfig.UpstreamConfig.Enabled, true)
	require.NotEmpty(nodeConfig.UpstreamConfig.URL)
	require.Equal(nodeConfig.UpstreamConfig.URL, params.UpstreamRopstenEthereumNetworkURL)

	rpcService := new(service)
	httpRPCServer := httptest.NewServer(rpcService)

	nodeConfig.UpstreamConfig.URL = httpRPCServer.URL
	nodeConfig.UpstreamConfig.Enabled = true

	started, err := s.NodeManager.StartNode(nodeConfig)
	require.NoError(err)

	// Attempt to find out if we started well.
	select {
	case <-started:
		break
	case <-time.After(1 * time.Second):
		s.T().Fatal("failed to start node manager")
		break
	}

	rpcNodeManager, ok := s.NodeManager.(common.RPCNodeManager)
	require.Equal(ok, true)

	accountManager := rpcNodeManager.Account()
	require.NotNil(accountManager)

	accountPassword := "fieldMarshal"
	address, _, _, err := accountManager.CreateAccount(accountPassword)
	require.NoError(err)

	selectErr := accountManager.SelectAccount(address, accountPassword)
	require.NoError(selectErr)

	rpcService.Handler = func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var txReq txRequest

		if err := json.NewDecoder(r.Body).Decode(&txReq); err != nil {
			require.NoError(err)
			return
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
		require.NotNil(tx)

		require.Equal(tx.ChainId().Int64(), int64(nodeConfig.NetworkID))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc": "2.0", "status":200, "result": "3434=done"}`))
	}

	defer s.NodeManager.StopNode()

	odFunc := otto.FunctionCall{
		Otto: otto.New(),
		This: otto.NullValue(),
	}

	// create a new client and issue a request.
	client, err := s.NodeManager.RPCClient()
	require.NoError(err)
	require.NotNil(client)

	request := common.RPCCall{
		ID:     65454545334343,
		Method: "eth_sendTransaction",
		Params: []interface{}{
			map[string]interface{}{
				"from":     address,
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

	res, err := rpcNodeManager.Exec(request, odFunc)
	require.NoError(err)

	result, err := res.Get("result")
	require.NoError(err)
	require.NotNil(result)

	exported, err := result.Export()
	require.NoError(err)

	rawJSON, ok := exported.(json.RawMessage)
	require.Equal(ok, true)
	require.IsType((json.RawMessage)(nil), rawJSON)

	require.Equal(string(rawJSON), "\"3434=done\"")
}

// func (s *RPCRouterTestSuite) TestMainnetAcceptance() {
// 	require := s.Require()
// 	require.NotNil(s.NodeManager)
//
// 	nodeConfig, err := MakeTestNodeConfig(params.MainNetworkID)
// 	require.NoError(err)
//
// 	// validate default state of UpstreamConfig.Enable.
// 	require.NotEmpty(nodeConfig.UpstreamConfig.URL)
// 	require.NotEqual(nodeConfig.UpstreamConfig.Enabled, true)
// 	require.Equal(nodeConfig.UpstreamConfig.URL, params.UpstreamMainNetEthereumNetworkURL)
//
// 	nodeConfig.UpstreamConfig.Enabled = true
//
// 	require.Equal(nodeConfig.UpstreamConfig.Enabled, true)
//
// 	started, err := s.NodeManager.StartNode(nodeConfig)
// 	require.NoError(err)
//
// 	defer s.NodeManager.StopNode()
//
// 	// Attempt to find out if we started well.
// 	select {
// 	case <-started:
// 		break
// 	case <-time.After(1 * time.Second):
// 		s.T().Fatal("Failed to start node manager")
// 		break
// 	}
//
// 	rpcNodeManager, ok := s.NodeManager.(common.RPCNodeManager)
// 	require.Equal(ok, true)
//
// 	accountManager := rpcNodeManager.Account()
// 	require.NotNil(accountManager)
//
// 	accountPassword := "fieldMarshal"
// 	address, _, _, err := accountManager.CreateAccount(accountPassword)
// 	require.NoError(err)
//
// 	selectErr := accountManager.SelectAccount(address, accountPassword)
// 	require.NoError(selectErr)
//
// 	odFunc := otto.FunctionCall{
// 		Otto: otto.New(),
// 		This: otto.NullValue(),
// 	}
//
// 	// create a new client and issue a request.
// 	client, err := rpcNodeManager.RPCClient()
// 	require.NoError(err)
// 	require.NotNil(client)
//
// 	request := common.RPCCall{
// 		ID:     65454545334343,
// 		Method: "eth_sendTransaction",
// 		Params: []interface{}{
// 			map[string]interface{}{
// 				"from":     address,
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
//
// 	res, err := rpcNodeManager.Exec(request, odFunc)
// 	require.NoError(err)
//
// 	result, err := res.Get("result")
// 	require.NoError(err)
// 	require.NotNil(result)
//
// 	hash, err := res.Get("hash")
// 	require.NoError(err)
// 	require.NotNil(result)
//
// 	exported, err := result.Export()
// 	require.NoError(err)
//
// 	fmt.Printf("Res: %+q\n", result)
// 	fmt.Printf("ResHash: %+q\n", hash)
// 	fmt.Printf("ResExported: %+q\n", exported)
// }
//
// func (s *RPCRouterTestSuite) TestRopstenAcceptance() {
// 	require := s.Require()
// 	require.NotNil(s.NodeManager)
//
// 	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
// 	require.NoError(err)
//
// 	// validate default state of UpstreamConfig.Enable.
// 	require.NotEmpty(nodeConfig.UpstreamConfig.URL)
// 	require.NotEqual(nodeConfig.UpstreamConfig.Enabled, true)
// 	require.Equal(nodeConfig.UpstreamConfig.URL, params.UpstreamRopstenEthereumNetworkURL)
//
// 	nodeConfig.UpstreamConfig.Enabled = true
//
// 	require.Equal(nodeConfig.UpstreamConfig.Enabled, true)
//
// 	started, err := s.NodeManager.StartNode(nodeConfig)
// 	require.NoError(err)
//
// 	defer s.NodeManager.StopNode()
//
// 	// Attempt to find out if we started well.
// 	select {
// 	case <-started:
// 		break
// 	case <-time.After(1 * time.Second):
// 		s.T().Fatal("Failed to start node manager")
// 		break
// 	}
//
// 	rpcNodeManager, ok := s.NodeManager.(common.RPCNodeManager)
// 	require.Equal(ok, true)
//
// 	accountManager := rpcNodeManager.Account()
// 	require.NotNil(accountManager)
//
// 	accountPassword := "fieldMarshal"
// 	address, _, _, err := accountManager.CreateAccount(accountPassword)
// 	require.NoError(err)
//
// 	selectErr := accountManager.SelectAccount(address, accountPassword)
// 	require.NoError(selectErr)
//
// 	odFunc := otto.FunctionCall{
// 		Otto: otto.New(),
// 		This: otto.NullValue(),
// 	}
//
// 	// create a new client and issue a request.
// 	client, err := rpcNodeManager.RPCClient()
// 	require.NoError(err)
// 	require.NotNil(client)
//
// 	request := common.RPCCall{
// 		ID:     65454545334343,
// 		Method: "eth_sendTransaction",
// 		Params: []interface{}{
// 			map[string]interface{}{
// 				"from":     address,
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
//
// 	res, err := rpcNodeManager.Exec(request, odFunc)
// 	require.NoError(err)
//
// 	result, err := res.Get("result")
// 	require.NoError(err)
// 	require.NotNil(result)
//
// 	hash, err := res.Get("hash")
// 	require.NoError(err)
// 	require.NotNil(result)
//
// 	exported, err := result.Export()
// 	require.NoError(err)
//
// 	fmt.Printf("Res: %+q\n", result)
// 	fmt.Printf("ResHash: %+q\n", hash)
// 	fmt.Printf("ResExported: %+q\n", exported)
// }
//
// func (s *RPCRouterTestSuite) TestRinkebyAcceptance() {
// 	require := s.Require()
// 	require.NotNil(s.NodeManager)
//
// 	nodeConfig, err := MakeTestNodeConfig(params.RinkebyNetworkID)
// 	require.NoError(err)
//
// 	// validate default state of UpstreamConfig.Enable.
// 	require.NotEmpty(nodeConfig.UpstreamConfig.URL)
// 	require.NotEqual(nodeConfig.UpstreamConfig.Enabled, true)
// 	require.Equal(nodeConfig.UpstreamConfig.URL, params.UpstreamRinkebyEthereumNetworkURL)
//
// 	nodeConfig.UpstreamConfig.Enabled = true
//
// 	require.Equal(nodeConfig.UpstreamConfig.Enabled, true)
//
// 	started, err := s.NodeManager.StartNode(nodeConfig)
// 	require.NoError(err)
//
// 	defer s.NodeManager.StopNode()
//
// 	// Attempt to find out if we started well.
// 	select {
// 	case <-started:
// 		break
// 	case <-time.After(1 * time.Second):
// 		s.T().Fatal("Failed to start node manager")
// 		break
// 	}
//
// 	rpcNodeManager, ok := s.NodeManager.(common.RPCNodeManager)
// 	require.Equal(ok, true)
//
// 	accountManager := rpcNodeManager.Account()
// 	require.NotNil(accountManager)
//
// 	accountPassword := "fieldMarshal"
// 	address, _, _, err := accountManager.CreateAccount(accountPassword)
// 	require.NoError(err)
//
// 	selectErr := accountManager.SelectAccount(address, accountPassword)
// 	require.NoError(selectErr)
//
// 	odFunc := otto.FunctionCall{
// 		Otto: otto.New(),
// 		This: otto.NullValue(),
// 	}
//
// 	// create a new client and issue a request.
// 	client, err := rpcNodeManager.RPCClient()
// 	require.NoError(err)
// 	require.NotNil(client)
//
// 	request := common.RPCCall{
// 		ID:     65454545334343,
// 		Method: "eth_sendTransaction",
// 		Params: []interface{}{
// 			map[string]interface{}{
// 				"from":     address,
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
//
// 	res, err := rpcNodeManager.Exec(request, odFunc)
// 	require.NoError(err)
//
// 	result, err := res.Get("result")
// 	require.NoError(err)
// 	require.NotNil(result)
//
// 	hash, err := res.Get("hash")
// 	require.NoError(err)
// 	require.NotNil(result)
//
// 	exported, err := result.Export()
// 	require.NoError(err)
//
// 	fmt.Printf("Res: %+q\n", result)
// 	fmt.Printf("ResHash: %+q\n", hash)
// 	fmt.Printf("ResExported: %+q\n", exported)
// }
