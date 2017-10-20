package jail

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/robertkrimen/otto"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/signal"
	"github.com/stretchr/testify/suite"
)

func TestHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

type HandlersTestSuite struct {
	suite.Suite
	responseFixture string
	ts              *httptest.Server
	tsCalls         int
	client          *gethrpc.Client
}

func (s *HandlersTestSuite) SetupTest() {
	s.responseFixture = `{"json-rpc":"2.0","id":10,"result":true}`
	s.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.tsCalls++
		fmt.Fprintln(w, s.responseFixture)
	}))

	client, err := gethrpc.Dial(s.ts.URL)
	s.NoError(err)
	s.client = client
}

func (s *HandlersTestSuite) TearDownTest() {
	s.ts.Close()
	s.tsCalls = 0
}

func (s *HandlersTestSuite) TestWeb3SendHandlerSuccess() {
	client, err := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	s.NoError(err)

	jail := New(&testRPCClientProvider{client})

	cell, err := jail.createAndInitCell("cell1")
	s.NoError(err)

	// web3.eth.syncing is an arbitrary web3 sync RPC call.
	value, err := cell.Run("web3.eth.syncing")
	s.NoError(err)
	result, err := value.ToBoolean()
	s.NoError(err)
	s.True(result)
}

func (s *HandlersTestSuite) TestWeb3SendHandlerFailure() {
	jail := New(nil)

	cell, err := jail.createAndInitCell("cell1")
	s.NoError(err)

	_, err = cell.Run("web3.eth.syncing")
	s.Error(err, ErrNoRPCClient.Error())
}

func (s *HandlersTestSuite) TestWeb3SendAsyncHandlerSuccess() {
	client, err := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	s.NoError(err)

	jail := New(&testRPCClientProvider{client})

	cell, err := jail.createAndInitCell("cell1")
	s.NoError(err)

	errc := make(chan string)
	resultc := make(chan string)
	err = cell.Set("__getSyncingCallback", func(call otto.FunctionCall) otto.Value {
		errc <- call.Argument(0).String()
		resultc <- call.Argument(1).String()
		return otto.UndefinedValue()
	})
	s.NoError(err)

	_, err = cell.Run(`web3.eth.getSyncing(__getSyncingCallback)`)
	s.NoError(err)

	s.Equal(`null`, <-errc)
	s.Equal(`true`, <-resultc)
}

func (s *HandlersTestSuite) TestWeb3SendAsyncHandlerWithoutCallbackSuccess() {
	client, err := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	s.NoError(err)

	jail := New(&testRPCClientProvider{client})

	cell, err := jail.createAndInitCell("cell1")
	s.NoError(err)

	_, err = cell.Run(`web3.eth.getSyncing()`)
	s.NoError(err)

	// As there is no callback, it's not possible to detect when
	// the request hit the server.
	time.Sleep(time.Millisecond * 100)
	s.Equal(1, s.tsCalls)
}

func (s *HandlersTestSuite) TestWeb3SendAsyncHandlerFailure() {
	jail := New(nil)

	cell, err := jail.createAndInitCell("cell1")
	s.NoError(err)

	errc := make(chan otto.Value)
	resultc := make(chan string)
	err = cell.Set("__getSyncingCallback", func(call otto.FunctionCall) otto.Value {
		errc <- call.Argument(0)
		resultc <- call.Argument(1).String()
		return otto.UndefinedValue()
	})
	s.NoError(err)

	_, err = cell.Run(`web3.eth.getSyncing(__getSyncingCallback)`)
	s.NoError(err)

	errValue := <-errc
	message, err := errValue.Object().Get("message")
	s.NoError(err)

	s.Equal(ErrNoRPCClient.Error(), message.String())
	s.Equal(`undefined`, <-resultc)
}

func (s *HandlersTestSuite) TestWeb3IsConnectedHandler() {
	client, err := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	s.NoError(err)

	jail := New(&testRPCClientProvider{client})

	cell, err := jail.createAndInitCell("cell1")
	s.NoError(err)

	// When result is true.
	value, err := cell.Run("web3.isConnected()")
	s.NoError(err)
	result, err := value.Object().Get("result")
	s.NoError(err)
	resultBool, err := result.ToBoolean()
	s.NoError(err)
	s.True(resultBool)

	// When result is false.
	s.responseFixture = `{"json-rpc":"2.0","id":10,"result":false}`
	value, err = cell.Run("web3.isConnected()")
	s.NoError(err)
	result, err = value.Object().Get("error")
	s.NoError(err)
	s.Equal(node.ErrNoRunningNode.Error(), result.String())
}

func (s *HandlersTestSuite) TestSendSignalHandler() {
	jail := New(nil)

	cell, err := jail.createAndInitCell("cell1")
	s.NoError(err)

	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		s.Contains(jsonEvent, "test signal message")
	})

	value, err := cell.Run(`statusSignals.sendSignal("test signal message")`)
	s.NoError(err)
	result, err := value.Object().Get("result")
	s.NoError(err)
	resultBool, err := result.ToBoolean()
	s.NoError(err)
	s.True(resultBool)
}
