package router

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBufferApprovalGas(t *testing.T) {
	// Pre-Glamsterdam approve() to a fresh spender, and the same after EIP-8037 state gas.
	require.Equal(t, uint64(48300), bufferApprovalGas(46000))
	require.Equal(t, uint64(150213), bufferApprovalGas(143060))
	require.Equal(t, uint64(0), bufferApprovalGas(0))
}
