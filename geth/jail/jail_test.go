package jail

import (
	"testing"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/geth/rpc"
	"github.com/stretchr/testify/suite"
)

type testRPCClientProvider struct {
	rpcClient *rpc.Client
}

func (p testRPCClientProvider) RPCClient() *rpc.Client {
	return p.rpcClient
}

func TestJailTestSuite(t *testing.T) {
	suite.Run(t, new(JailTestSuite))
}

type JailTestSuite struct {
	suite.Suite
	Jail *Jail
}

func (s *JailTestSuite) SetupTest() {
	s.Jail = New(nil)
}

func (s *JailTestSuite) TestJailCreateCell() {
	_, err := s.Jail.CreateCell("cell1")
	s.NoError(err)
	_, err = s.Jail.CreateCell("cell1")
	s.EqualError(err, "cell with id 'cell1' already exists")

	// create more cells
	_, err = s.Jail.CreateCell("cell2")
	s.NoError(err)
	_, err = s.Jail.CreateCell("cell3")
	s.NoError(err)

	s.Len(s.Jail.cells, 3)
}

func (s *JailTestSuite) TestJailInitCell() {
	// InitCell on a non-existent cell.
	result := s.Jail.InitCell("cellNonExistent", "")
	s.Equal(`{"error":"cell 'cellNonExistent' not found"}`, result)

	// InitCell on an existing cell.
	cell, err := s.Jail.CreateCell("cell1")
	s.NoError(err)
	result = s.Jail.InitCell("cell1", "")
	// TODO(adam): this is confusing... There should be a separate method to validate this.
	s.Equal(`{"error":"ReferenceError: '_status_catalog' is not defined"}`, result)

	// web3 should be available
	value, err := cell.Run("web3.fromAscii('ethereum')")
	s.NoError(err)
	s.Equal(`0x657468657265756d`, value.String())
}

func (s *JailTestSuite) TestJailStop() {
	_, err := s.Jail.CreateCell("cell1")
	s.NoError(err)
	s.Len(s.Jail.cells, 1)

	s.Jail.Stop()

	s.Len(s.Jail.cells, 0)
}

func (s *JailTestSuite) TestJailCall() {
	cell, err := s.Jail.CreateCell("cell1")
	s.NoError(err)

	propsc := make(chan string, 1)
	argsc := make(chan string, 1)
	err = cell.Set("call", func(call otto.FunctionCall) otto.Value {
		propsc <- call.Argument(0).String()
		argsc <- call.Argument(1).String()

		return otto.UndefinedValue()
	})
	s.NoError(err)

	result := s.Jail.Call("cell1", `["prop1", "prop2"]`, `arg1`)
	s.Equal(`["prop1", "prop2"]`, <-propsc)
	s.Equal(`arg1`, <-argsc)
	s.Equal(`{"result": undefined}`, result)
}

func (s *JailTestSuite) TestCreateAndInitCell() {
	response := s.Jail.CreateAndInitCell("cell1", `var testCreateAndInitCell = true`)
	// TODO(adam): confusing, this check should be in another method
	s.Equal(`{"error":"ReferenceError: '_status_catalog' is not defined"}`, response)

	cell, err := s.Jail.GetCell("cell1")
	s.NoError(err)

	value, err := cell.Get("testCreateAndInitCell")
	s.NoError(err)
	s.Equal(`true`, value.String())
}
