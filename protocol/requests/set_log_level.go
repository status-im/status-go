package requests

import (
	"github.com/status-im/status-go/logutils"
)

type SetLogLevel struct {
	LogLevel string `json:"logLevel"`
}

func (c *SetLogLevel) Validate() error {
	if _, err := logutils.LvlFromString(c.LogLevel); err != nil {
		return err
	}
	return nil
}
