package requests

import (
	"errors"
)

var ErrURLInvalid = errors.New("generate-basic-url-code: invalid or empty URL provided")

type GenerateBasicURLCode struct {
	URL                  string `json:"url"`
	ErrorCorrectionLevel string `json:"errorCorrectionLevel"`
	Capacity             string `json:"capacity"`
}

func (c *GenerateBasicURLCode) Validate() error {
	if len(c.URL) == 0 {
		return ErrURLInvalid
	}

	return nil
}
