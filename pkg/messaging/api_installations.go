package messaging

import (
	"crypto/ecdsa"

	"github.com/google/uuid"

	"github.com/status-im/status-go/pkg/messaging/adapters"
	"github.com/status-im/status-go/pkg/messaging/types"
)

func GenerateInstallationID() string {
	return uuid.New().String()
}

func (a *API) AddInstallation(identity []byte, timestamp int64, installation *types.Installation, enabled bool) ([]*types.Installation, error) {
	allInstallations, err := a.core.stack.Encryption.AddInstallation(identity, timestamp, adapters.ToEncryptionInstallation(installation), enabled)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(allInstallations), nil
}

func (a *API) AddInstallations(identity []byte, timestamp int64, installations []*types.Installation, enabled bool) ([]*types.Installation, error) {
	allInstallations, err := a.core.stack.Encryption.AddInstallations(identity, timestamp, adapters.ToEncryptionInstallations(installations), enabled)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(allInstallations), nil
}

func (a *API) GetOurInstallations(myIdentityKey *ecdsa.PublicKey) ([]*types.Installation, error) {
	installations, err := a.core.stack.Encryption.GetOurInstallations(myIdentityKey)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(installations), nil
}

func (a *API) GetOurActiveInstallations(myIdentityKey *ecdsa.PublicKey) ([]*types.Installation, error) {
	installations, err := a.core.stack.Encryption.GetOurActiveInstallations(myIdentityKey)
	if err != nil {
		return nil, err
	}
	return adapters.FromEncryptionInstallations(installations), nil
}

func (a *API) SetInstallationMetadata(myIdentityKey *ecdsa.PublicKey, installationID string, data *types.InstallationMetadata) error {
	return a.core.stack.Encryption.SetInstallationMetadata(myIdentityKey, installationID, adapters.ToEncryptionInstallationMetadata(data))
}

func (a *API) SetInstallationName(myIdentityKey *ecdsa.PublicKey, installationID string, name string) error {
	return a.core.stack.Encryption.SetInstallationName(myIdentityKey, installationID, name)
}

func (a *API) EnableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return a.core.stack.Encryption.EnableInstallation(myIdentityKey, installationID)
}

func (a *API) DisableInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return a.core.stack.Encryption.DisableInstallation(myIdentityKey, installationID)
}

func (a *API) DeleteInstallation(myIdentityKey *ecdsa.PublicKey, installationID string) error {
	return a.core.stack.Encryption.DeleteInstallation(myIdentityKey, installationID)
}
