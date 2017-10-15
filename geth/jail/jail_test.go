package jail

import (
	"testing"

	"github.com/robertkrimen/otto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestJailTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersTestSuite))
}

type JailTestSuite struct {
	suite.Suite
	Jail *Jail
}

func (s *JailTestSuite) SetupTest() {
	s.Jail = New(nil)
}

func (s *JailTestSuite) TestNewJailProvidesDefaultClientProvider() {
	s.NotNil(s.Jail.rpcClientProvider)
	s.Nil(s.Jail.rpcClientProvider())
}

func (s *JailTestSuite) TestJailCreateCell(t *testing.T) {
	_, err := s.Jail.CreateCell("cell1")
	require.NoError(t, err)
	_, err = s.Jail.CreateCell("cell1")
	require.EqualError(t, err, "cell with id 'cell1' already exists")

	// create more cells
	_, err = s.Jail.CreateCell("cell2")
	require.NoError(t, err)
	_, err = s.Jail.CreateCell("cell3")
	require.NoError(t, err)

	require.Len(t, s.Jail.cells, 3)
}

func (s *JailTestSuite) TestJailInitCell(t *testing.T) {
	// InitCell on a non-existent cell.
	result := s.Jail.InitCell("cellNonExistent", "")
	require.Equal(t, `{"error":"cell 'cellNonExistent' not found"}`, result)

	// InitCell on an existing cell.
	cell, err := s.Jail.CreateCell("cell1")
	require.NoError(t, err)
	result = s.Jail.InitCell("cell1", "")
	// TODO(adam): this is confusing... There should be a separate method to validate this.
	require.Equal(t, `{"error":"ReferenceError: '_status_catalog' is not defined"}`, result)

	// web3 should be available
	value, err := cell.Run("web3.fromAscii('ethereum')")
	require.NoError(t, err)
	require.Equal(t, `0x657468657265756d`, value.String())
}

func (s *JailTestSuite) TestJailStop(t *testing.T) {
	cell, err := s.Jail.CreateCell("cell1")
	require.NoError(t, err)
	require.Len(t, s.Jail.cells, 1)

	s.Jail.Stop()

	// Verify that cell's loop was canceled.
	<-cell.loopStopped
	require.Nil(t, s.Jail.cells)
}

func (s *JailTestSuite) TestJailCall(t *testing.T) {
	cell, err := s.Jail.CreateCell("cell1")
	require.NoError(t, err)

	propsc := make(chan string, 1)
	argsc := make(chan string, 1)
	err = cell.Set("call", func(call otto.FunctionCall) {
		propsc <- call.Argument(0).String()
		argsc <- call.Argument(1).String()
	})
	require.NoError(t, err)

	result := s.Jail.Call("cell1", `["prop1", "prop2"]`, `arg1`)
	require.Equal(t, `["prop1", "prop2"]`, <-propsc)
	require.Equal(t, `arg1`, <-argsc)
	require.Equal(t, `{"result": undefined}`, result)
}
