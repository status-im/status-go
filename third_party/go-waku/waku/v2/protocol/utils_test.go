package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFulltextMatch(t *testing.T) {
	expectedProtocol := "/vac/waku/relay/2.0.0"
	fn := FulltextMatch(expectedProtocol)

	require.True(t, fn(expectedProtocol))
	require.False(t, fn("/some/random/protocol/1.0.0"))
}

func TestPrefixTextMatch(t *testing.T) {
	expectedPrefix := "/vac/waku/relay/2.0"
	fn := PrefixTextMatch(expectedPrefix)

	require.True(t, fn("/vac/waku/relay/2.0.0"))
	require.True(t, fn("/vac/waku/relay/2.0.2"))
	require.False(t, fn("/vac/waku/relay/2.1.0"))
	require.False(t, fn("/some/random/protocol/1.0.0"))
}
