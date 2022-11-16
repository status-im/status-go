package requests

import (
	"errors"
)

var ErrURLInvalidID = errors.New("generate-qr-code-error: invalid url passed to endpoint, please check the url you are passing to generate the QR code")

type QROptions struct {
	URL                  string `json:"url"`
	ErrorCorrectionLevel string `json:"errorCorrectionLevel"`
	Capacity             string `json:"capacity"`
	AllowProfileImage    bool   `json:"withLogo"`
}

func (c *QROptions) Validate() error {
	if len(c.URL) == 0 {
		return ErrURLInvalidID
	}

	return nil
}
