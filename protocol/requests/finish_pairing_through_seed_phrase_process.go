package requests

import (
	"errors"
)

var ErrFinishPairingThroughSeedPhraseProcessInvalidID = errors.New("finish pairing through seed phrase process: invalid installation id")

type FinishPairingThroughSeedPhraseProcess struct {
	InstallationID string `json:"installationId"`
}

func (j *FinishPairingThroughSeedPhraseProcess) Validate() error {
	if len(j.InstallationID) == 0 {
		return ErrFinishPairingThroughSeedPhraseProcessInvalidID
	}

	return nil
}
