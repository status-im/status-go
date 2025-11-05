package backupsync

import (
	"encoding/json"

	accsmanagementtypes "github.com/status-im/status-go/internal/accounts-management/types"
	"github.com/status-im/status-go/multiaccounts/settings"
)

type BackedUpDataResponse struct {
	Clock            uint64
	Profile          *BackedUpProfile
	Setting          *settings.SyncSettingField
	Keypair          *accsmanagementtypes.Keypair
	WatchOnlyAccount *accsmanagementtypes.Account
}

func (sfwr *BackedUpDataResponse) MarshalJSON() ([]byte, error) {
	responseItem := struct {
		Clock            uint64                       `json:"clock,omitempty"`
		Profile          *BackedUpProfile             `json:"backedUpProfile,omitempty"`
		Setting          *settings.SyncSettingField   `json:"backedUpSettings,omitempty"`
		Keypair          *accsmanagementtypes.Keypair `json:"backedUpKeypair,omitempty"`
		WatchOnlyAccount *accsmanagementtypes.Account `json:"backedUpWatchOnlyAccount,omitempty"`
	}{
		Clock:            sfwr.Clock,
		Profile:          sfwr.Profile,
		Setting:          sfwr.Setting,
		Keypair:          sfwr.Keypair,
		WatchOnlyAccount: sfwr.WatchOnlyAccount,
	}

	return json.Marshal(responseItem)
}
