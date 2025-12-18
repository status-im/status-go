package ethclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeth_HeaderHash(t *testing.T) {
	number, hash, header := getTestBlockHeader()
	require.Equal(t, number.String(), header.Number.String())
	require.Equal(t, hash, header.Hash())
}
