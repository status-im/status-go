package requests

import (
	"errors"
)

var ErrEnableInstallationAndPairInvalidID = errors.New("enable installation and pair: invalid installation id")

type EnableInstallationAndPair struct {
	InstallationID string `json:"installationId"`
}

func (j *EnableInstallationAndPair) Validate() error {
	if len(j.InstallationID) == 0 {
		return ErrEnableInstallationAndPairInvalidID
	}

	return nil
}

func (j *EnableInstallationAndPair) GetInstallationID() string {
	return j.InstallationID
}
