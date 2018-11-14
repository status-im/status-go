package verifier

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/stretchr/testify/require"
)

func TestLocalVerifierForNodeIDTypes(t *testing.T) {
	nodeID := enode.ID{1}

	v := NewLocalVerifier([]enode.ID{{1}})
	require.True(t, v.VerifyNode(context.TODO(), nodeID))
	require.False(t, v.VerifyNode(context.TODO(), enode.ID{2}))
}
