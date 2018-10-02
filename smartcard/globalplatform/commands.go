package globalplatform

import (
	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/globalplatform/crypto"
)

const (
	ClaISO7816 = uint8(0x00)
	ClaGp      = uint8(0x80)
	ClaMac     = uint8(0x84)

	InsSelect               = uint8(0xA4)
	InsInitializeUpdate     = uint8(0x50)
	InsExternalAuthenticate = uint8(0x82)
	InsGetResponse          = uint8(0xC0)

	Sw1ResponseDataIncomplete = uint8(0x61)
)

func NewCommandSelect(aid []byte) *apdu.Command {
	c := apdu.NewCommand(
		ClaISO7816,
		InsSelect,
		uint8(0x04),
		uint8(0x00),
		aid,
	)

	c.SetLe(0x00)

	return c
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

func NewCommandExternalAuthenticate(encKey, cardChallenge, hostChallenge []byte) (*apdu.Command, error) {
	hostCryptogram, err := calculateHostCryptogram(encKey, cardChallenge, hostChallenge)
	if err != nil {
		return nil, err
	}

	return apdu.NewCommand(
		ClaMac,
		InsExternalAuthenticate,
		uint8(0x01), // C-MAC
		uint8(0x00),
		hostCryptogram,
	), nil
}

func NewCommandGetResponse(length uint8) *apdu.Command {
	c := apdu.NewCommand(
		ClaISO7816,
		InsGetResponse,
		uint8(0),
		uint8(0),
		nil,
	)

	c.SetLe(length)

	return c
}

func calculateHostCryptogram(encKey, cardChallenge, hostChallenge []byte) ([]byte, error) {
	var data []byte
	data = append(data, cardChallenge...)
	data = append(data, hostChallenge...)
	data = crypto.AppendDESPadding(data)

	return crypto.Mac3DES(encKey, data, crypto.NullBytes8)
}
