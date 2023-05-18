package wakusync

import (
	"encoding/json"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/multiaccounts/keycards"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/protobuf"
)

type WakuBackedUpDataResponse struct {
	FetchingDataProgress map[string]*protobuf.FetchingBackedUpDataDetails // key represents the data/section backup details refer to
	Profile              *BackedUpProfile
	Setting              *settings.SyncSettingField
	Keycards             []*keycards.Keycard
	WalletAccount        *accounts.Account
}

func (sfwr *WakuBackedUpDataResponse) MarshalJSON() ([]byte, error) {
	responseItem := struct {
		FetchingDataProgress map[string]FetchingBackupedDataDetails `json:"fetchingBackedUpDataProgress,omitempty"`
		Profile              *BackedUpProfile                       `json:"backedUpProfile,omitempty"`
		Setting              *settings.SyncSettingField             `json:"backedUpSettings,omitempty"`
		Keycards             []*keycards.Keycard                    `json:"backedUpKeycards,omitempty"`
		WalletAccount        *accounts.Account                      `json:"backedUpWalletAccount,omitempty"`
	}{
		Profile:       sfwr.Profile,
		Setting:       sfwr.Setting,
		Keycards:      sfwr.Keycards,
		WalletAccount: sfwr.WalletAccount,
	}

	responseItem.FetchingDataProgress = sfwr.FetchingBackedUpDataDetails()

	return json.Marshal(responseItem)
}
