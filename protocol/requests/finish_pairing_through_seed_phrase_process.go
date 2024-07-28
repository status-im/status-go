package requests

import (
	"errors"
)

var ErrFinishPairingThroughSeedPhraseProcessInvalidID = errors.New("deactivate-chat: invalid id")

type FinishPairingThroughSeedPhraseProcess struct {
	InstallationID string `json:"installationId"`
}

func (j *FinishPairingThroughSeedPhraseProcess) Validate() error {
	if len(j.InstallationID) == 0 {
		return ErrFinishPairingThroughSeedPhraseProcessInvalidID
	}

	return nil
}
