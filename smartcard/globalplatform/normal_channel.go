package globalplatform

import (
	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/hexutils"
)

type Transmitter interface {
	Transmit([]byte) ([]byte, error)
}

type NormalChannel struct {
	t Transmitter
}

func NewNormalChannel(t Transmitter) *NormalChannel {
	return &NormalChannel{t}
}

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
