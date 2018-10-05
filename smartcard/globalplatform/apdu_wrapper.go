package globalplatform

import (
	"bytes"
	"encoding/binary"

	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/globalplatform/crypto"
)

// APDUWrapper is a wrapper for apdu commands inside a global platform secure channel.
type APDUWrapper struct {
	macKey []byte
	icv    []byte
}

// NewAPDUWrapper returns a new APDUWrapper using the specified key for MAC generation.
func NewAPDUWrapper(macKey []byte) *APDUWrapper {
	return &APDUWrapper{
		macKey: macKey,
		icv:    crypto.NullBytes8,
	}
}

// Wrap wraps the apdu command adding the MAC to the end of the command.
// Future implementations will encrypt the message when needed.
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

	w.icv = mac

	newCmd := apdu.NewCommand(cla, cmd.Ins, cmd.P1, cmd.P2, newData)
	if ok, le := cmd.Le(); ok {
		newCmd.SetLe(le)
	}

	return newCmd, nil
}
