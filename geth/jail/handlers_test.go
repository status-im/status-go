package jail

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertkrimen/otto"

	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/status-im/status-go/geth/signal"
	"github.com/stretchr/testify/suite"
)

type HandlersTestSuite struct {
	suite.Suite
	testServerResponseFixture string
	ts                        *httptest.Server
	client                    *gethrpc.Client
}

func TestHandlersTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

func (s *HandlersTestSuite) SetupTest() {
	s.testServerResponseFixture = `{"json-rpc":"2.0","id":10,"result":true}`
	s.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, s.testServerResponseFixture)
	}))

	client, err := gethrpc.Dial(s.ts.URL)
	s.NoError(err)
	s.client = client
}

func (s *HandlersTestSuite) TearDownTest() {
	s.ts.Close()
}

func (s *HandlersTestSuite) TestWeb3SendHandlerSuccess() {
	client, err := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	s.NoError(err)

	jail := New(func() *rpc.Client { return client })
	cell, err := jail.CreateCell("cell1")
	s.NoError(err)
	jail.InitCell("cell1", "")

	value, err := cell.Run("web3.eth.syncing")
	s.NoError(err)
	result, err := value.ToBoolean()
	s.NoError(err)
	s.True(result)
}

func (s *HandlersTestSuite) TestWeb3SendHandlerFailure() {
	jail := New(func() *rpc.Client { return nil })
	cell, err := jail.CreateCell("cell1")
	s.NoError(err)
	jail.InitCell("cell1", "")

	_, err = cell.Run("web3.eth.syncing")
	s.Error(err, ErrNoRPCClient.Error())
}

func (s *HandlersTestSuite) TestWeb3SendAsyncHandlerSuccess() {
	client, err := rpc.NewClient(s.client, params.UpstreamRPCConfig{})
	s.NoError(err)

	jail := New(func() *rpc.Client { return client })

	cell, err := jail.CreateCell("cell1")
	s.NoError(err)
	jail.InitCell("cell1", "")

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

func (s *HandlersTestSuite) TestWeb3SendAsyncHandlerFailure() {
	jail := New(func() *rpc.Client { return nil })

	cell, err := jail.CreateCell("cell1")
	s.NoError(err)
	jail.InitCell("cell1", "")

	errc := make(chan otto.Value)
	resultc := make(chan string)
	err = cell.Set("__getSyncingCallback", func(call otto.FunctionCall) otto.Value {
		errc <- call.Argument(0)
		resultc <- call.Argument(1).String()
		return otto.UndefinedValue()
	})
	s.NoError(err)

	_, err = cell.Run(`web3.eth.getSyncing(function (err, result) {
		__getSyncingCallback(err, result);
	})`)
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

	jail := New(func() *rpc.Client { return client })
	cell, err := jail.CreateCell("cell1")
	s.NoError(err)
	jail.InitCell("cell1", "")

	// When result is true.
	value, err := cell.Run("web3.isConnected()")
	s.NoError(err)
	result, err := value.Object().Get("result")
	s.NoError(err)
	resultBool, err := result.ToBoolean()
	s.NoError(err)
	s.True(resultBool)

	// When result is false.
	s.testServerResponseFixture = `{"json-rpc":"2.0","id":10,"result":false}`
	value, err = cell.Run("web3.isConnected()")
	s.NoError(err)
	result, err = value.Object().Get("error")
	s.NoError(err)
	s.Equal("there is no running node", result.String())
}

func (s *HandlersTestSuite) TestSendSignalHandler() {
	jail := New(func() *rpc.Client { return nil })
	cell, err := jail.CreateCell("cell1")
	s.NoError(err)
	jail.InitCell("cell1", "")

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
