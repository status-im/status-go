package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHexDecoding(t *testing.T) {
	const s = "0x48656c6c6f20476f7068657221"
	decodedString, err := DecodeHexString(s)
	require.NoError(t, err)
	require.Equal(t, decodedString, []byte("Hello Gopher!"))

	const s1 = "48656c6c6f20476f7068657221"
	_, err = DecodeHexString(s1)
	require.NoError(t, err)
	require.Equal(t, decodedString, []byte("Hello Gopher!"))

	const s2 = "jk"
	_, err = DecodeHexString(s2)
	require.Error(t, err)

	const s3 = "48656c6c6f20476f706865722"
	_, err = DecodeHexString(s3)
	require.Error(t, err)
}
