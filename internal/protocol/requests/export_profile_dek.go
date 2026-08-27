package requests

import "errors"

var (
	ErrExportProfileDEKInvalidKeyUID   = errors.New("export-profile-dek: no keyUID provided")
	ErrExportProfileDEKInvalidPassword = errors.New("export-profile-dek: no password provided")
)

type ExportProfileDEK struct {
	KeyUID   string `json:"keyUID"`
	Password string `json:"password"`
}

func (r *ExportProfileDEK) Validate() error {
	if len(r.KeyUID) == 0 {
		return ErrExportProfileDEKInvalidKeyUID
	}
	if len(r.Password) == 0 {
		return ErrExportProfileDEKInvalidPassword
	}
	return nil
}
