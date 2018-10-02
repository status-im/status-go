package smartcard

import (
	"fmt"

	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/globalplatform"
	"github.com/status-im/status-go/smartcard/hexutils"
)

type SecureChannel struct {
	session *globalplatform.Session
	c       Channel
	w       *globalplatform.APDUWrapper
}

func NewSecureChannel(session *globalplatform.Session, c Channel) *SecureChannel {
	return &SecureChannel{
		session: session,
		c:       c,
		w:       globalplatform.NewAPDUWrapper(session.KeyProvider().Mac()),
	}
}

func (c *SecureChannel) Send(cmd *apdu.Command) (*apdu.Response, error) {
	rawCmd, err := cmd.Serialize()
	if err != nil {
		return nil, err
	}

	fmt.Printf("WRAPPING  %s\n", hexutils.BytesToHexWithSpaces(rawCmd))
	wrappedCmd, err := c.w.Wrap(cmd)
	if err != nil {
		return nil, err
	}

	rawWrappedCmd, err := wrappedCmd.Serialize()
	if err != nil {
		return nil, err
	}

	fmt.Printf("WRAPPED  %s\n", hexutils.BytesToHexWithSpaces(rawWrappedCmd))

	return c.c.Send(wrappedCmd)
}
