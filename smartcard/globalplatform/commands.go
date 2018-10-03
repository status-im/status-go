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
	InsDelete               = uint8(0xE4)
	InsLoad                 = uint8(0xE8)
	InsInstall              = uint8(0xE6)

	P1ExternalAuthenticateCMAC = uint8(0x01)
	P1InstallForLoad           = uint8(0x02)
	P1InstallForInstall        = uint8(0x04)
	P1InstallForMakeSelectable = uint8(0x08)
	P1LoadMoreBlocks           = uint8(0x00)
	P1LoadLastBlock            = uint8(0x80)

	Sw1ResponseDataIncomplete = uint8(0x61)

	SwOK                            = uint16(0x9000)
	SwReferencedDataNotFound        = uint16(0x6A88)
	SwSecurityConditionNotSatisfied = uint16(0x6982)
	SwAuthenticationMethodBlocked   = uint16(0x6983)

	tagDeleteAID         = byte(0x4F)
	tagLoadFileDataBlock = byte(0xC4)
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
		P1ExternalAuthenticateCMAC,
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

func NewCommandDelete(aid []byte) *apdu.Command {
	data := []byte{tagDeleteAID, byte(len(aid))}
	data = append(data, aid...)

	return apdu.NewCommand(
		ClaGp,
		InsDelete,
		uint8(0x00),
		uint8(0x00),
		data,
	)
}

func NewCommandInstallForLoad(aid, sdaid []byte) *apdu.Command {
	data := []byte{byte(len(aid))}
	data = append(data, aid...)
	data = append(data, byte(len(sdaid)))
	data = append(data, sdaid...)
	// empty hash length and hash
	data = append(data, []byte{0x00, 0x00, 0x00}...)

	return apdu.NewCommand(
		ClaGp,
		InsInstall,
		P1InstallForLoad,
		uint8(0x00),
		data,
	)
}

func NewCommandInstallForInstall(pkgAID, appletAID, instanceAID, params []byte) *apdu.Command {
	data := []byte{byte(len(pkgAID))}
	data = append(data, pkgAID...)
	data = append(data, byte(len(appletAID)))
	data = append(data, appletAID...)
	data = append(data, byte(len(instanceAID)))
	data = append(data, instanceAID...)

	// privileges
	priv := []byte{0x00}
	data = append(data, byte(len(priv)))
	data = append(data, priv...)

	// params
	fullParams := []byte{byte(0xC9), byte(len(params))}
	fullParams = append(fullParams, params...)

	data = append(data, byte(len(fullParams)))
	data = append(data, fullParams...)

	// empty perform token
	data = append(data, byte(0x00))

	return apdu.NewCommand(
		ClaGp,
		InsInstall,
		P1InstallForInstall|P1InstallForMakeSelectable,
		uint8(0x00),
		data,
	)
}

func calculateHostCryptogram(encKey, cardChallenge, hostChallenge []byte) ([]byte, error) {
	var data []byte
	data = append(data, cardChallenge...)
	data = append(data, hostChallenge...)
	data = crypto.AppendDESPadding(data)

	return crypto.Mac3DES(encKey, data, crypto.NullBytes8)
}
