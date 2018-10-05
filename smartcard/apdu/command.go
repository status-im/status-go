package apdu

import (
	"bytes"
	"encoding/binary"
)

// Command struct represent the data sent as an APDU command with CLA, Ins, P1, P2, Lc, Data, and Le.
type Command struct {
	Cla        uint8
	Ins        uint8
	P1         uint8
	P2         uint8
	Data       []byte
	le         uint8
	requiresLe bool
}

// NewCommand returns a new apdu Command.
func NewCommand(cla, ins, p1, p2 uint8, data []byte) *Command {
	return &Command{
		Cla:        cla,
		Ins:        ins,
		P1:         p1,
		P2:         p2,
		Data:       data,
		requiresLe: false,
	}
}

// SetLe sets the expected Le value and makes sure the Le value is sent in the apdu Command.
func (c *Command) SetLe(le uint8) {
	c.requiresLe = true
	c.le = le
}

// Le returns if Le is set and its value.
func (c *Command) Le() (bool, uint8) {
	return c.requiresLe, c.le
}

// Serialize serielizes the command into a raw bytes sequence.
func (c *Command) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, c.Cla); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, c.Ins); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, c.P1); err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.BigEndian, c.P2); err != nil {
		return nil, err
	}

	if len(c.Data) > 0 {
		if err := binary.Write(buf, binary.BigEndian, uint8(len(c.Data))); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, c.Data); err != nil {
			return nil, err
		}
	}

	if c.requiresLe {
		if err := binary.Write(buf, binary.BigEndian, c.le); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
