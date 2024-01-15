package requests

import "github.com/status-im/status-go/multiaccounts/common"

type SetCustomizationColor struct {
	CustomizationColor common.CustomizationColor `json:"customizationColor"`
	KeyUID             string                    `json:"keyUid"`
}
