package ethclient_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeth_HeaderHash(t *testing.T) {
	header, number, hash := getTestBlockHeader()
	require.Equal(t, number.String(), header.Number.String())
	require.Equal(t, hash, header.Hash())
}
