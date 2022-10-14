package requests

import (
	"errors"
)

var ErrCreateAccountInvalidDisplayName = errors.New("create-account: invalid display name")
var ErrCreateAccountInvalidPassword = errors.New("create-account: invalid password")
var ErrCreateAccountInvalidImagePath = errors.New("create-account: invalid image path")
var ErrCreateAccountInvalidColor = errors.New("create-account: invalid color")

type CreateAccount struct {
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	ImagePath   string `json:"imagePath"`
	Color       string `json:"color"`
}

func (c *CreateAccount) Validate() error {
	// TODO(cammellos): Add proper validation for password/displayname/etc
	if len(c.DisplayName) == 0 {
		return ErrCreateAccountInvalidDisplayName
	}

	if len(c.Password) == 0 {
		return ErrCreateAccountInvalidPassword
	}

	if len(c.ImagePath) == 0 {
		return ErrCreateAccountInvalidImagePath
	}

	if len(c.Color) == 0 {
		return ErrCreateAccountInvalidColor
	}

	return nil

}
