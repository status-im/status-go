package requests

import (
	"github.com/status-im/status-go/v10/multiaccounts"
	"github.com/status-im/status-go/v10/multiaccounts/settings"
)

type ConvertToKeycardAccount struct {
	Account     multiaccounts.Account `json:"account"`
	Settings    settings.Settings     `json:"settings"`
	KeycardUID  string                `json:"keycardUID"`
	OldPassword string                `json:"oldPassword"`
	NewPassword string                `json:"newPassword"`
}
