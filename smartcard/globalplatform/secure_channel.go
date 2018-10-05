package globalplatform

import (
	"github.com/status-im/status-go/smartcard/apdu"
	"github.com/status-im/status-go/smartcard/hexutils"
)

type SecureChannel struct {
	session *Session
	c       Channel
	w       *APDUWrapper
}

func NewSecureChannel(session *Session, c Channel) *SecureChannel {
	return &SecureChannel{
		session: session,
		c:       c,
		w:       NewAPDUWrapper(session.KeyProvider().Mac()),
	}
}

func (c *SecureChannel) Send(cmd *apdu.Command) (*apdu.Response, error) {
	rawCmd, err := cmd.Serialize()
	if err != nil {
		return nil, err
	}

	logger.Debug("wrapping apdu command", "hex", hexutils.BytesToHexWithSpaces(rawCmd))
	wrappedCmd, err := c.w.Wrap(cmd)
	if err != nil {
		return nil, err
	}

	return c.c.Send(wrappedCmd)
}
