package node_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/jail"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

//==================================================================================================

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
	TxQueueManager common.TxQueueManager
	Account        common.AccountManager
	Policy         *jail.ExecutionPolicy
}

func (s *RPCTestSuite) SetupTest() {
	require := s.Require()

	nodeManager := node.NewNodeManager()
	require.NotNil(nodeManager)

	acctman := node.NewAccountManager(nodeManager)
	require.NotNil(acctman)

	txQueueManager := node.NewTxQueueManager(nodeManager, acctman)
	require.NotNil(txQueueManager)

	policy := jail.NewExecutionPolicy(nodeManager, acctman, txQueueManager)
	require.NotNil(policy)

	s.Policy = policy
	s.Account = acctman
	s.NodeManager = nodeManager
	s.TxQueueManager = txQueueManager
}

func (s *RPCTestSuite) TestRPCSendTransaction() {
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
	s.TxQueueManager.Start()

	defer s.StopTestNode()
	defer s.TxQueueManager.Stop()

	client, err := s.NodeManager.RPCClient()
	require.NoError(err)
	require.NotNil(client)

	selectErr := s.Account.SelectAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	require.NoError(selectErr)

	s.TxQueueManager.SetTransactionQueueHandler(func(queuedTx *common.QueuedTx) {
		s.T().Logf("queued a new transaction: %s", queuedTx.ID)
		_, err := s.TxQueueManager.CompleteTransaction(queuedTx.ID, TestConfig.Account1.Password)
		require.NoError(err)
	})

	res, err := s.Policy.Execute(request, odFunc)
	require.NoError(err)

	result, err := res.Get("result")
	require.NoError(err)
	require.NotNil(result)
	require.True(result.IsString())
	require.NotEmpty(result.String())
}

func (s *RPCTestSuite) TestCallRPC() {
	require := s.Require()
	require.NotNil(s.NodeManager)

	rpcClient := node.NewRPCManager(s.NodeManager)
	require.NotNil(rpcClient)

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
				"data": "0xd46e8dd67c5d32be8d46e8dd67c5d32be8058bb8eb970870f072445675058bb8eb970870f072445675"}],"id":1}`,
			func(resultJSON string) {
				log.Info("eth_sendTransaction")
				s.T().Log("GOT: ", resultJSON)
				progress <- struct{}{}
			},
		},
		{
			`{"jsonrpc":"2.0","method":"shh_version","params":[],"id":67}`,
			func(resultJSON string) {
				expected := `{"jsonrpc":"2.0","id":67,"result":"5.0"}` + "\n"
				s.Equal(expected, resultJSON)
				s.T().Log("shh_version: ", resultJSON)
				progress <- struct{}{}
			},
		},
		{
			`{"jsonrpc":"2.0","method":"web3_sha3","params":["0x68656c6c6f20776f726c64"],"id":64}`,
			func(resultJSON string) {
				expected := `{"jsonrpc":"2.0","id":64,"result":"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad"}` + "\n"
				s.Equal(expected, resultJSON)
				s.T().Log("web3_sha3: ", resultJSON)
				progress <- struct{}{}
			},
		},
		{
			`{"jsonrpc":"2.0","method":"net_version","params":[],"id":67}`,
			func(resultJSON string) {
				expected := `{"jsonrpc":"2.0","id":67,"result":"4"}` + "\n"
				s.Equal(expected, resultJSON)
				s.T().Log("net_version: ", resultJSON)
				progress <- struct{}{}
			},
		},
	}

	cnt := len(rpcCalls) - 1 // send transaction blocks up until complete/discarded/times out
	for _, r := range rpcCalls {
		go func(r rpcCall) {
			s.T().Logf("Run test: %v", r.inputJSON)
			resultJSON := rpcClient.Call(r.inputJSON)
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
