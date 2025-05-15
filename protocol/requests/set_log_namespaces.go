package requests

import "github.com/status-im/status-go/v10/logutils"

type SetLogNamespaces struct {
	LogNamespaces string `json:"logNamespaces"`
}

func (c *SetLogNamespaces) Validate() error {
	return logutils.ValidateNamespaces(c.LogNamespaces)
}
