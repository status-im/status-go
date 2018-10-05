package globalplatform

import (
	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/hexutils"
)

// Transmitter defines an interface with one method to transmit raw commands and receive raw responses.
type Transmitter interface {
	Transmit([]byte) ([]byte, error)
}

// NormalChannel implements a normal channel to send apdu commands and receive apdu responses.
type NormalChannel struct {
	t Transmitter
}

// NewNormalChannel returns a new NormalChannel that sends commands to Transmitter t.
func NewNormalChannel(t Transmitter) *NormalChannel {
	return &NormalChannel{t}
}

// Send sends apdu commands to the current Transmitter.
// Based on the smartcard transport protocol (T=0, T=1), it checks responses and sends a Get Response
// command in case of T=0.
func (c *NormalChannel) Send(cmd *apdu.Command) (*apdu.Response, error) {
	rawCmd, err := cmd.Serialize()
	if err != nil {
		return nil, err
	}

	logger.Debug("apdu command", "hex", hexutils.BytesToHexWithSpaces(rawCmd))
	rawResp, err := c.t.Transmit(rawCmd)
	if err != nil {
		return nil, err
	}
	logger.Debug("apdu response", "hex", hexutils.BytesToHexWithSpaces(rawResp))

	resp, err := apdu.ParseResponse(rawResp)
	if err != nil {
		return nil, err
	}

	if resp.Sw1 == Sw1ResponseDataIncomplete && (cmd.Cla != ClaISO7816 || cmd.Ins != InsGetResponse) {
		getResponse := NewCommandGetResponse(resp.Sw2)
		return c.Send(getResponse)
	}

	return apdu.ParseResponse(rawResp)
}
