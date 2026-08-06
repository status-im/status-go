package requests

import "errors"

var ErrGetProfileEncryptionInfoInvalidKeyUID = errors.New("get-profile-encryption-info: no keyUID provided")

type GetProfileEncryptionInfo struct {
	KeyUID string `json:"keyUID"`
}

func (r *GetProfileEncryptionInfo) Validate() error {
	if len(r.KeyUID) == 0 {
		return ErrGetProfileEncryptionInfoInvalidKeyUID
	}
	return nil
}
