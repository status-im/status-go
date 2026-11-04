package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateRequestId(t *testing.T) {
	// Force 2 reseed to ensure this is working as expected
	for i := 1; i < 20001; i++ {
		bytes := GenerateRequestID()
		require.Equal(t, 32, len(bytes))
	}
}
