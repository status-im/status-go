package requests

import (
	"github.com/status-im/status-go/internal/db/multiaccounts"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
)

type ConvertToKeycardAccount struct {
	Account     multiaccounts.Account `json:"account"`
	Settings    settings.Settings     `json:"settings"`
	KeycardUID  string                `json:"keycardUID"`
	OldPassword string                `json:"oldPassword"`
	NewPassword string                `json:"newPassword"`

	// KeycardSecureChannelV2 is set when the keycard (applet 4.x) uses SecureChannelV2 and so has no
	// pairing; Account.KeycardPairing is then ignored.
	KeycardSecureChannelV2 bool `json:"keycardSecureChannelV2"`
}
