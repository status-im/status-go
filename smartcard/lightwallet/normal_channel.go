package lightwallet

import (
	"fmt"

	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/globalplatform"
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

	fmt.Printf("COMMAND  %s\n", hexutils.BytesToHexWithSpaces(rawCmd))
	rawResp, err := c.t.Transmit(rawCmd)
	if err != nil {
		return nil, err
	}
	fmt.Printf("RESPONSE %s\n", hexutils.BytesToHexWithSpaces(rawResp))

	resp, err := apdu.ParseResponse(rawResp)
	if err != nil {
		return nil, err
	}

	if resp.Sw1 == globalplatform.Sw1ResponseDataIncomplete && (cmd.Cla != globalplatform.ClaISO7816 || cmd.Ins != globalplatform.InsGetResponse) {
		getResponse := globalplatform.NewCommandGetResponse(resp.Sw2)
		return c.Send(getResponse)
	}

	return apdu.ParseResponse(rawResp)
}
