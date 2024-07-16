package requests

import (
	"errors"
)

var (
	ErrInitializeApplicationInvalidDataDir = errors.New("initialize-centralized-metric: no dataDir")
)

type InitializeApplication struct {
	DataDir       string `json:"dataDir"`
	MixpanelAppID string `json:"mixpanelAppId"`
	MixpanelToken string `json:"mixpanelToken"`
}

func (i *InitializeApplication) Validate() error {
	if len(i.DataDir) == 0 {
		return ErrInitializeApplicationInvalidDataDir
	}
	return nil
}
