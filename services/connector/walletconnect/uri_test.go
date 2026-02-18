package walletconnect

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseURI_Valid(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?relay-protocol=irn&symKey=abcd1234&projectId=test-project"

	parsed, err := ParseURI(uri)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", parsed.Topic)
	require.Equal(t, "2", parsed.Version)
	require.Equal(t, "irn", parsed.RelayProtocol)
	require.Equal(t, "abcd1234", parsed.SymKey)
	require.Equal(t, "test-project", parsed.ProjectID)
}

func TestParseURI_MinimalValid(t *testing.T) {
	uri := "wc:0000000000000000000000000000000000000000000000000000000000000000@2?symKey=key123"

	parsed, err := ParseURI(uri)
	require.NoError(t, err)
	require.Equal(t, "0000000000000000000000000000000000000000000000000000000000000000", parsed.Topic)
	require.Equal(t, "2", parsed.Version)
	require.Equal(t, "key123", parsed.SymKey)
	require.Equal(t, "irn", parsed.RelayProtocol)
}

func TestParseURI_WithExpiryTimestamp(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?symKey=key&expiryTimestamp=1234567890"

	parsed, err := ParseURI(uri)
	require.NoError(t, err)
	require.Equal(t, int64(1234567890), parsed.ExpiryTimestamp)
}

func TestParseURI_WithSpaces(t *testing.T) {
	uri := "  wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?symKey=key  "

	parsed, err := ParseURI(uri)
	require.NoError(t, err)
	require.Equal(t, "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", parsed.Topic)
}

func TestParseURI_NoPrefix(t *testing.T) {
	uri := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?symKey=key"

	_, err := ParseURI(uri)
	require.Error(t, err)
	require.Equal(t, ErrInvalidURI, err)
}

func TestParseURI_NoAtSymbol(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef2?symKey=key"

	_, err := ParseURI(uri)
	require.Error(t, err)
	require.Equal(t, ErrInvalidURI, err)
}

func TestParseURI_ShortTopic(t *testing.T) {
	uri := "wc:123@2?symKey=key"

	_, err := ParseURI(uri)
	require.Error(t, err)
	require.Equal(t, ErrInvalidURI, err)
}

func TestParseURI_LongTopic(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef00@2?symKey=key"

	_, err := ParseURI(uri)
	require.Error(t, err)
	require.Equal(t, ErrInvalidURI, err)
}

func TestParseURI_InvalidHexTopic(t *testing.T) {
	uri := "wc:gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg@2?symKey=key"

	_, err := ParseURI(uri)
	require.Error(t, err)
	require.Equal(t, ErrInvalidURI, err)
}

func TestParseURI_NoSymKey(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?relay-protocol=irn"

	_, err := ParseURI(uri)
	require.Error(t, err)
	require.Equal(t, ErrInvalidURI, err)
}

func TestParseURI_EmptySymKey(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?symKey="

	_, err := ParseURI(uri)
	require.Error(t, err)
	require.Equal(t, ErrInvalidURI, err)
}

func TestParseURI_InvalidQuery(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?%%invalid%%"

	_, err := ParseURI(uri)
	require.Error(t, err)
	require.Equal(t, ErrInvalidURI, err)
}

func TestParseURI_NoQuery(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2"

	_, err := ParseURI(uri)
	require.Error(t, err)
	require.Equal(t, ErrInvalidURI, err)
}

func TestParseURI_CaseInsensitiveHex(t *testing.T) {
	uri := "wc:ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890@2?symKey=key"

	parsed, err := ParseURI(uri)
	require.NoError(t, err)
	require.Equal(t, "ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890", parsed.Topic)
}

func TestParseURI_MixedCase(t *testing.T) {
	uri := "wc:AbCdEf1234567890aBcDeF1234567890AbCdEf1234567890aBcDeF1234567890@2?symKey=key"

	parsed, err := ParseURI(uri)
	require.NoError(t, err)
	require.Equal(t, "AbCdEf1234567890aBcDeF1234567890AbCdEf1234567890aBcDeF1234567890", parsed.Topic)
}

func TestParseURI_MultipleQueryParams(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?symKey=k&relay-protocol=waku&projectId=pid&expiryTimestamp=999"

	parsed, err := ParseURI(uri)
	require.NoError(t, err)
	require.Equal(t, "k", parsed.SymKey)
	require.Equal(t, "waku", parsed.RelayProtocol)
	require.Equal(t, "pid", parsed.ProjectID)
	require.Equal(t, int64(999), parsed.ExpiryTimestamp)
}

func TestParseURI_DefaultRelayProtocol(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?symKey=key"

	parsed, err := ParseURI(uri)
	require.NoError(t, err)
	require.Equal(t, "irn", parsed.RelayProtocol)
}

func TestParseURI_InvalidExpiryTimestamp(t *testing.T) {
	uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@2?symKey=key&expiryTimestamp=notanumber"

	parsed, err := ParseURI(uri)
	require.NoError(t, err)
	require.Equal(t, int64(0), parsed.ExpiryTimestamp)
}

func TestParseURI_VersionVariants(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"v1", "1"},
		{"v2", "2"},
		{"v2.0", "2.0"},
		{"beta", "beta"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "wc:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef@" + tt.version + "?symKey=key"
			parsed, err := ParseURI(uri)
			require.NoError(t, err)
			require.Equal(t, tt.version, parsed.Version)
		})
	}
}
