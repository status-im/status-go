package globalplatform

import (
	"testing"

	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/hexutils"
	"github.com/stretchr/testify/assert"
)

func TestNewSession(t *testing.T) {
	key := hexutils.HexToBytes("404142434445464748494a4b4c4d4e4f")
	keys := NewKeyProvider(key, key)

	raw := hexutils.HexToBytes("000002650183039536622002000de9c62ba1c4c8e55fcb91b6654ce49000")
	resp, err := apdu.ParseResponse(raw)
	assert.NoError(t, err)

	hostChallenge := hexutils.HexToBytes("f0467f908e5ca23f")
	_, err = NewSession(keys, resp, hostChallenge)
	assert.NoError(t, err)
}

func TestNewSession_BadResponse(t *testing.T) {
	raw := hexutils.HexToBytes("01026982")
	resp, err := apdu.ParseResponse(raw)
	assert.NoError(t, err)
	_, err = NewSession(&KeyProvider{}, resp, []byte{})
	assert.Error(t, err)

	raw = hexutils.HexToBytes("01026983")
	resp, err = apdu.ParseResponse(raw)
	assert.NoError(t, err)
	_, err = NewSession(&KeyProvider{}, resp, []byte{})
	assert.Error(t, err)

	// bad data length
	raw = hexutils.HexToBytes("01029000")
	resp, err = apdu.ParseResponse(raw)
	assert.NoError(t, err)
	_, err = NewSession(&KeyProvider{}, resp, []byte{})
	assert.Error(t, err)
}
