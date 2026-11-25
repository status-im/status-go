package requests

import "errors"

var (
	ErrSetZeroOnboardedProfilePasswordInvalidKeyUID      = errors.New("set-zero-onboarded-profile-password: no keyUID provided")
	ErrSetZeroOnboardedProfilePasswordInvalidNewPassword = errors.New("set-zero-onboarded-profile-password: no new password provided")
)

type SetZeroOnboardedProfilePassword struct {
	KeyUID      string `json:"keyUID"`
	NewPassword string `json:"newPassword"`
}

func (r *SetZeroOnboardedProfilePassword) Validate() error {
	if len(r.KeyUID) == 0 {
		return ErrSetZeroOnboardedProfilePasswordInvalidKeyUID
	}
	if len(r.NewPassword) == 0 {
		return ErrSetZeroOnboardedProfilePasswordInvalidNewPassword
	}
	return nil
}
