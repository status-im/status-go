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
	cell, err := s.Jail.CreateCell("cell1")
	s.NoError(err)
	s.NotNil(cell)
	// creating another cell with the same id fails
	_, err = s.Jail.CreateCell("cell1")
	s.EqualError(err, "cell with id 'cell1' already exists")

	// create more cells
	_, err = s.Jail.CreateCell("cell2")
	s.NoError(err)
	_, err = s.Jail.CreateCell("cell3")
	s.NoError(err)
	s.Len(s.Jail.cells, 3)
}

func (s *JailTestSuite) TestJailGetCell() {
	// cell1 does not exist
	_, err := s.Jail.Cell("cell1")
	s.EqualError(err, "cell 'cell1' not found")

	// cell 1 exists
	_, err = s.Jail.CreateCell("cell1")
	s.NoError(err)
	cell, err := s.Jail.Cell("cell1")
	s.NoError(err)
	s.NotNil(cell)
}

func (s *JailTestSuite) TestJailInitCell() {
	// InitCell on an existing cell.
	cell, err := s.Jail.createCell("cell1")
	s.NoError(err)
	err = s.Jail.initCell(cell)
	s.NoError(err)

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

func (s *JailTestSuite) TestMakeCatalogVariable() {
	cell, err := s.Jail.createCell("cell1")
	s.NoError(err)

	// no `_status_catalog` variable
	response := s.Jail.makeCatalogVariable(cell)
	s.Equal(`{"error":"ReferenceError: '_status_catalog' is not defined"}`, response)

	// with `_status_catalog` variable
	cell.Run(`var _status_catalog = { test: true }`)
	response = s.Jail.makeCatalogVariable(cell)
	s.Equal(`{"result": {"test":true}}`, response)
}

func (s *JailTestSuite) TestCreateAndInitCell() {
	cell, err := s.Jail.createAndInitCell(
		"cell1",
		`var testCreateAndInitCell1 = true`,
		`var testCreateAndInitCell2 = true`,
	)
	s.NoError(err)
	s.NotNil(cell)

	value, err := cell.Get("testCreateAndInitCell1")
	s.NoError(err)
	s.Equal(`true`, value.String())

	value, err = cell.Get("testCreateAndInitCell2")
	s.NoError(err)
	s.Equal(`true`, value.String())
}

func (s *JailTestSuite) TestPublicCreateAndInitCell() {
	response := s.Jail.CreateAndInitCell("cell1", `var _status_catalog = { test: true }`)
	s.Equal(`{"result": {"test":true}}`, response)
}
