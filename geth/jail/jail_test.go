package jail

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewJailProvidesDefaultClientProvider(t *testing.T) {
	j := New(nil)
	require.NotNil(t, j.rpcClientProvider)
}

func TestJailCreateCell(t *testing.T) {
	j := New(nil)

	_, err := j.CreateCell("cell1")
	require.NoError(t, err)
	_, err = j.CreateCell("cell1")
	require.EqualError(t, err, "cell with id 'cell1' already exists")

	// create more cells
	_, err = j.CreateCell("cell2")
	require.NoError(t, err)
	_, err = j.CreateCell("cell3")
	require.NoError(t, err)

	require.Len(t, j.cells, 3)
}

func TestJailInitCell(t *testing.T) {
	j := New(nil)

	// InitCell on a non-existent cell.
	result := j.InitCell("cellNonExistent", "")
	require.Equal(t, `{"error":"cell 'cellNonExistent' not found"}`, result)

	// InitCell on an existing cell.
	cell, err := j.CreateCell("cell1")
	require.NoError(t, err)
	result = j.InitCell("cell1", "")
	// TODO(adam): this is confusing... There should be a separate method to validate this.
	require.Equal(t, `{"error":"ReferenceError: '_status_catalog' is not defined"}`, result)

	// web3 should be available
	value, err := cell.Run("web3.fromAscii('ethereum')")
	require.NoError(t, err)
	require.Equal(t, `0x657468657265756d`, value.String())
}

func TestJailStop(t *testing.T) {
	j := New(nil)

	cell, err := j.CreateCell("cell1")
	require.NoError(t, err)
	require.Len(t, j.cells, 1)

	j.Stop()

	// Verify that cell's loop was canceled.
	<-cell.loopStopped
	require.Nil(t, j.cells)
}

func TestJailCall(t *testing.T) {
	j := New(nil)

	cell, err := j.CreateCell("cell1")
	require.NoError(t, err)

	propsc := make(chan string, 1)
	argsc := make(chan string, 1)
	err = cell.Set("call", func(props, args string) {
		propsc <- props
		argsc <- args
	})
	require.NoError(t, err)

	result := j.Call("cell1", `["prop1", "prop2"]`, `arg1`)
	require.Equal(t, `["prop1", "prop2"]`, <-propsc)
	require.Equal(t, `arg1`, <-argsc)
	require.Equal(t, `{"result": undefined}`, result)
}
