package globalplatform

import "github.com/status-im/status-go/smartcard/apdu"

const Cla = uint8(0x00)
const ClaGp = uint8(0x80)

const InsSelect = uint8(0xA4)

func NewCommandSelect(aid []byte) *apdu.Command {
	return apdu.NewCommand(
		Cla,
		InsSelect,
		uint8(0x04),
		uint8(0x00),
		aid,
	)
}
