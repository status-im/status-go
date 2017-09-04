package jail_test

import (
	"fmt"
	"testing"

	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
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

func TestJailRPCTestSuite(t *testing.T) {
	suite.Run(t, new(JailRPCTestSuite))
}

type JailRPCTestSuite struct {
	BaseTestSuite
	Account        common.AccountManager
	Policy         *jail.ExecutionPolicy
	TxQueueManager common.TxQueueManager
}

func (s *JailRPCTestSuite) SetupTest() {
	require := s.Require()

	nodeManager := node.NewNodeManager()

	acctman := node.NewAccountManager(nodeManager)
	require.NotNil(acctman)

	txQueueManager := node.NewTxQueueManager(nodeManager, acctman)

	policy := jail.NewExecutionPolicy(nodeManager, acctman, txQueueManager)
	require.NotNil(policy)

	s.Policy = policy
	s.Account = acctman
	s.NodeManager = nodeManager
	s.TxQueueManager = txQueueManager
}

func (s *JailRPCTestSuite) TestSendTransaction() {
	require := s.Require()

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

	rpcService := new(service)
	rpcService.Handler = func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var txReq txRequest

		err := json.NewDecoder(r.Body).Decode(&txReq)
		require.NoError(err)

		fmt.Printf("Incoming request: %+q\n", txReq)

		switch txReq.Method {
		case "eth_getTransactionCount":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc": "2.0", "status":200, "result": "0x434"}`))
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

		// Validate we are receiving transaction from the proper network chain.
		c, err := s.NodeManager.NodeConfig()
		require.NoError(err)
		require.Equal(tx.ChainId().Uint64(), c.NetworkID)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc": "2.0", "status":200, "result": "3434=done"}`))
	}

	// httpRPCServer will serve as an upstream server accepting transactions.
	httpRPCServer := httptest.NewServer(rpcService)
	s.StartTestNode(params.RopstenNetworkID, WithUpstream(httpRPCServer.URL))
	defer s.StopTestNode()

	client, err := s.NodeManager.RPCClient()
	require.NoError(err)
	require.NotNil(client)

	selectErr := s.Account.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	require.NoError(selectErr)

	res, err := s.Policy.Execute(request, odFunc)
	require.NoError(err)

	// TODO(influx6): How do we move this into ExecutionPolicy.executeSendTransaction since
	// we also need the account's password.
	// TransactionQueueHandler is required to enqueue a transaction.
	s.TxQueueManager.SetTransactionQueueHandler(func(queuedTx *common.QueuedTx) {
		s.TxQueueManager.CompleteTransaction(queuedTx.ID, TestConfig.Account1.Password)
	})

	result, err := res.Get("result")
	require.NoError(err)
	require.NotNil(result)

	exported, err := result.Export()
	require.NoError(err)

	rawJSON, ok := exported.(json.RawMessage)
	require.True(ok, "Expected raw json payload")
	require.Equal(string(rawJSON), "\"3434=done\"")
}

// func (s *JailRPCTestSuite) TestMainnetAcceptance() {
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

// 	nodeConfig, err := MakeTestNodeConfig(params.MainNetworkID)
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

// 	selectErr := s.Account.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
// 	require.NoError(selectErr)

// 	res, err := s.Policy.Execute(request, odFunc)
// 	require.NoError(err)

// 	_, err = res.Get("hash")
// 	require.NoError(err)
// }

// func (s *JailRPCTestSuite) TestRobstenAcceptance() {
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

// 	nodeConfig, err := MakeTestNodeConfig(params.RopstenNetworkID)
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

// 	selectErr := s.Account.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
// 	require.NoError(selectErr)

// 	res, err := s.Policy.Execute(request, odFunc)
// 	require.NoError(err)

// 	_, err = res.Get("hash")
// 	require.NoError(err)
// }

// func (s *JailRPCTestSuite) TestRinkebyAcceptance() {
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

// 	nodeConfig, err := MakeTestNodeConfig(params.RinkebyNetworkID)
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

// 	selectErr := s.Account.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
// 	require.NoError(selectErr)

// 	res, err := s.Policy.Execute(request, odFunc)
// 	require.NoError(err)

// 	_, err = res.Get("hash")
// 	require.NoError(err)
// }
