package proxy_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/proxy"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

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
		Params: []interface{}{},
	}

	rpcNodeManager, ok := s.NodeManager.(common.RPCNodeManager)
	require.Equal(ok, true)

	res, err := rpcNodeManager.Exec(request, odFunc)
	require.NoError(err)

	fmt.Printf("Res: %+q\n", res)
}
