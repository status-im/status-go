package jail

import (
	"encoding/json"
	"testing"

	"github.com/robertkrimen/otto"
	"github.com/status-im/status-go/rpc"
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
	cell, err := s.Jail.obtainCell("cell1", false)
	s.NoError(err)
	err = s.Jail.initCell(cell)
	s.NoError(err)

	// web3 should be available
	value, err := cell.Run("web3.fromAscii('ethereum')")
	s.NoError(err)
	s.Equal(`0x657468657265756d`, value.Value().String())
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
	s.Equal(`{"result":null}`, result)
}

func (s *JailTestSuite) TestCreateAndInitCell() {
	cell, result, err := s.Jail.createAndInitCell(
		"cell1",
		`var testCreateAndInitCell1 = true`,
		`var testCreateAndInitCell2 = true`,
		`testCreateAndInitCell2`,
	)
	s.NoError(err)
	s.NotNil(cell)
	s.Equal(`{"result":true}`, result)

	value, err := cell.Get("testCreateAndInitCell1")
	s.NoError(err)
	s.Equal(`true`, value.Value().String())

	value, err = cell.Get("testCreateAndInitCell2")
	s.NoError(err)
	s.Equal(`true`, value.Value().String())
}

func (s *JailTestSuite) TestPublicCreateAndInitCell() {
	var createAndInitTests = []struct {
		chatID      string
		input       []string
		expectation string
	}{
		{"cell1", []string{}, EmptyResponse},
		{"cell1", []string{"var a = 2", "a"}, `{"result":2}`},
		{"cell1", []string{`var a = "hello"`, "a"}, `{"result":"hello"}`},
		{"cell1", []string{`var b = "2"; var a = b * b`, "a"}, `{"result":4}`},
	}
	for _, v := range createAndInitTests {
		response := s.Jail.CreateAndInitCell(v.chatID, v.input...)
		s.Equal(v.expectation, response)
	}
}

func (s *JailTestSuite) TestNewJailResultResponseReturnsValidJson() {
	var newJailResultResponseTests = []interface{}{
		`Double quoted "success" response`,
		float64(1),
		true,
	}
	for _, input := range newJailResultResponseTests {
		v, err := otto.ToValue(input)
		s.NoError(err)

		output := newJailResultResponse(formatOttoValue(v))
		var response struct {
			Result interface{} `json:"result"`
		}
		err = json.Unmarshal([]byte(output), &response)

		s.NoError(err)
		s.Equal(input, response.Result)
	}
}

func (s *JailTestSuite) TestPublicCreateAndInitCellConsecutive() {
	response1 := s.Jail.CreateAndInitCell("cell1", `var _status_catalog = { test: true }; JSON.stringify(_status_catalog);`)
	s.Contains(response1, "test")
	cell1, err := s.Jail.Cell("cell1")
	s.NoError(err)

	// Create it again
	response2 := s.Jail.CreateAndInitCell("cell1", `var _status_catalog = { test: true, foo: 5 }; JSON.stringify(_status_catalog);`)
	s.Contains(response2, "test", "foo")
	cell2, err := s.Jail.Cell("cell1")
	s.NoError(err)

	// Second cell has to be the same object as the first one
	s.Equal(cell1, cell2)

	// Second cell must have been reinitialized
	s.NotEqual(response1, response2)
}

func (s *JailTestSuite) TestExecute() {
	// cell does not exist
	response := s.Jail.Execute("cell1", "('some string')")
	s.Equal(`{"error":"cell 'cell1' not found"}`, response)

	_, err := s.Jail.obtainCell("cell1", false)
	s.NoError(err)

	// cell exists
	response = s.Jail.Execute("cell1", `
		var obj = { test: true };
		JSON.stringify(obj);
	`)
	s.Equal(`{"test":true}`, response)
}

func (s *JailTestSuite) TestGetObjectValue() {
	cell, result, err := s.Jail.createAndInitCell(
		"cell1",
		`var testCreateAndInitCell1 = {obj: 'objValue'}`,
		`var testCreateAndInitCell2 = true`,
		`testCreateAndInitCell2`,
	)
	s.NoError(err)
	s.NotNil(cell)
	s.Equal(`{"result":true}`, result)

	testCreateAndInitCell1, err := cell.Get("testCreateAndInitCell1")
	s.NoError(err)
	s.True(testCreateAndInitCell1.Value().IsObject())
	value, err := cell.GetObjectValue(testCreateAndInitCell1.Value(), "obj")
	s.NoError(err)
	s.Equal("objValue", value.Value().String())

	value, err = cell.Get("testCreateAndInitCell2")
	s.NoError(err)
	s.Equal(`true`, value.Value().String())
}
