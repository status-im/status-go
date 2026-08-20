package requests

import (
	"errors"
)

var ErrEnableInstallationAndSyncInvalidID = errors.New("enable installation and sync : invalid installation id")

type EnableInstallationAndSync struct {
	InstallationID string `json:"installationId"`
}

func (j *EnableInstallationAndSync) Validate() error {
	if len(j.InstallationID) == 0 {
		return ErrEnableInstallationAndSyncInvalidID
	}

	return nil
}
