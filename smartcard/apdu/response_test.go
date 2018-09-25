package apdu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseResponse(t *testing.T) {
	raw := hexToBytes("000002650183039536622002003b5e508f751c0af3016e3fbc23d3a69000")
	resp, err := ParseResponse(raw)

	assert.NoError(t, err)
	assert.Equal(t, uint8(0x90), resp.Sw1)
	assert.Equal(t, uint8(0x00), resp.Sw2)
	assert.Equal(t, uint16(0x9000), resp.Sw)

	expected := "000002650183039536622002003B5E508F751C0AF3016E3FBC23D3A6"
	assert.Equal(t, expected, bytesToHex(resp.Data))
}

func TestParseResponse_BadData(t *testing.T) {
	raw := hexToBytes("")
	_, err := ParseResponse(raw)
	assert.Equal(t, ErrBadRawResponse, err)
}

func TestResp_IsOK(t *testing.T) {
	raw := hexToBytes("01029000")
	resp, err := ParseResponse(raw)
	assert.NoError(t, err)
	assert.True(t, resp.IsOK())
}
