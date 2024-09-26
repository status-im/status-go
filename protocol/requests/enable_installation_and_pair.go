package requests

import (
	"errors"
)

var ErrEnableInstallationAndPairInvalidID = errors.New("enable installation and pair: invalid installation id")

type EnableInstallationAndPair struct {
	InstallationID    string `json:"installationId"`
	getInstallationID func() string
}

func (j *EnableInstallationAndPair) Validate() error {
	if len(j.InstallationID) == 0 {
		return ErrEnableInstallationAndPairInvalidID
	}

	return nil
}

func (j *EnableInstallationAndPair) GetInstallationId() string {
	if j.getInstallationID != nil {
		return j.getInstallationID()
	}
	return j.InstallationID
}

func NewMockEnableInstallationAndPair(installationID string, mockGetInstallationID func() string) *EnableInstallationAndPair {
	return &EnableInstallationAndPair{
		InstallationID:    installationID,
		getInstallationID: mockGetInstallationID,
	}
}
