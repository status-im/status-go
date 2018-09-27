package globalplatform

import (
	"testing"

	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/globalplatform/crypto"
	"github.com/status-im/status-go/smartcard/hexutils"
	"github.com/stretchr/testify/assert"
)

func TestAPDUWrapper_Wrap(t *testing.T) {
	macKey := hexutils.HexToBytes("2983BA77D709C2DAA1E6000ABCCAC951")
	data := hexutils.HexToBytes("1d4de92eaf7a2c9f")

	cmd := apdu.NewCommand(uint8(0x80), uint8(0x82), uint8(0x01), uint8(0x00), data)
	w := NewAPDUWrapper(macKey)

	assert.Equal(t, crypto.NullBytes8, w.icv)

	wrappedCmd, err := w.Wrap(cmd)
	assert.NoError(t, err)
	raw, err := wrappedCmd.Serialize()
	assert.NoError(t, err)

	expected := "84 82 01 00 10 1D 4D E9 2E AF 7A 2C 9F 8F 9B 0D F6 81 C1 D3 EC"
	assert.Equal(t, expected, hexutils.BytesToHexWithSpaces(raw))
}
