package requests

import (
	"errors"
)

var ErrEnableAndSyncInstallationInvalidID = errors.New("enable and sync installation: invalid installation id")

type EnableAndSyncInstallation struct {
	InstallationID string `json:"installationId"`
}

func (j *EnableAndSyncInstallation) Validate() error {
	if len(j.InstallationID) == 0 {
		return ErrEnableAndSyncInstallationInvalidID
	}

	return nil
}
