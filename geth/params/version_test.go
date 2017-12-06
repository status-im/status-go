package params

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVersionWithMeta(t *testing.T) {
	var version = buildVersionString(1, 1, 0, "alpha.1")
	var expectedVersion = "1.1.0-alpha.1"
	require.Equal(t, expectedVersion, version)
}

func TestVersionWithoutMeta(t *testing.T) {
	var version = buildVersionString(0, 40, 100, "")
	var expectedVersion = "0.40.100"
	require.Equal(t, expectedVersion, version)
}
