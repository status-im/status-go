package globalplatform

import (
	"bytes"
	"encoding/binary"

	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/globalplatform/crypto"
)

type APDUWrapper struct {
	macKey []byte
	icv    []byte
}

func NewAPDUWrapper(macKey []byte) *APDUWrapper {
	return &APDUWrapper{
		macKey: macKey,
		icv:    crypto.NullBytes8,
	}
}

func (w *APDUWrapper) Wrap(cmd *apdu.Command) (*apdu.Command, error) {
	macData := new(bytes.Buffer)

	cla := cmd.Cla | 0x04
	if err := binary.Write(macData, binary.BigEndian, cla); err != nil {
		return nil, err
	}

	if err := binary.Write(macData, binary.BigEndian, cmd.Ins); err != nil {
		return nil, err
	}

	if err := binary.Write(macData, binary.BigEndian, cmd.P1); err != nil {
		return nil, err
	}

	if err := binary.Write(macData, binary.BigEndian, cmd.P2); err != nil {
		return nil, err
	}

	if err := binary.Write(macData, binary.BigEndian, uint8(len(cmd.Data)+8)); err != nil {
		return nil, err
	}

	if err := binary.Write(macData, binary.BigEndian, cmd.Data); err != nil {
		return nil, err
	}

	var (
		icv []byte
		err error
	)

	if bytes.Equal(w.icv, crypto.NullBytes8) {
		icv = w.icv
	} else {
		icv, err = crypto.EncryptICV(w.macKey, w.icv)
		if err != nil {
			return nil, err
		}
	}

	mac, err := crypto.MacFull3DES(w.macKey, macData.Bytes(), icv)
	if err != nil {
		return nil, err
	}

	newData := make([]byte, 0)
	newData = append(newData, cmd.Data...)
	newData = append(newData, mac...)

	return apdu.NewCommand(cla, cmd.Ins, cmd.P1, cmd.P2, newData), nil
}
