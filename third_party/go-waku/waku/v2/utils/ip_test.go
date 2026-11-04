package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIPValidation(t *testing.T) {
	require.Equal(t, IsIPv4("127.0.0.1"), true)
	require.Equal(t, IsIPv4("abcd"), false)
	require.Equal(t, IsIPv4("::1"), false)
	require.Equal(t, IsIPv4("1000.40.210.253"), false)

	require.Equal(t, IsIPv6("::1"), true)
	require.Equal(t, IsIPv6("fe80::6c4a:6aff:fecf:8097"), true)
	require.Equal(t, IsIPv6("2001:0db8:85a3:0000:0000:8a2e:0370:7334:3445"), false)
	require.Equal(t, IsIPv6("127.0.0.1"), false)

}
