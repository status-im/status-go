package requests

import "github.com/status-im/status-go/logutils"

type SetLogNamespaces struct {
	LogNamespaces string `json:"logNamespaces"`
}

func (c *SetLogNamespaces) Validate() error {
	return logutils.ValidateNamespaces(c.LogNamespaces)
}
