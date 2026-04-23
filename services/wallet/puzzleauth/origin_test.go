package puzzleauth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOriginForURL(t *testing.T) {
	o, err := OriginForURL("https://Example.com:8443/path/to/rpc")
	require.NoError(t, err)
	require.Equal(t, "https://example.com:8443", o)

	_, err = OriginForURL("not-a-url")
	require.Error(t, err)
}
