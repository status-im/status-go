package requests

import (
	"github.com/status-im/status-go/v10/logutils"
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
