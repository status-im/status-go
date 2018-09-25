package globalplatform

import "github.com/status-im/status-go/smartcard/apdu"

const (
	Cla   = uint8(0x00)
	ClaGp = uint8(0x80)

	InsSelect           = uint8(0xA4)
	InsInitializeUpdate = uint8(0x50)
)

func NewCommandSelect(aid []byte) *apdu.Command {
	return apdu.NewCommand(
		Cla,
		InsSelect,
		uint8(0x04),
		uint8(0x00),
		aid,
	)
}

func NewCommandInitializeUpdate(challenge []byte) *apdu.Command {
	return apdu.NewCommand(
		ClaGp,
		InsInitializeUpdate,
		uint8(0x00),
		uint8(0x00),
		challenge,
	)
}
