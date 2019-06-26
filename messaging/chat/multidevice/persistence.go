package multidevice

type Persistence interface {
	// GetActiveInstallations returns the active installations for a given identity.
	GetActiveInstallations(maxInstallations int, identity []byte) ([]*Installation, error)
	// EnableInstallation enables the installation.
	EnableInstallation(identity []byte, installationID string) error
	// DisableInstallation disable the installation.
	DisableInstallation(identity []byte, installationID string) error
	// AddInstallations adds the installations for a given identity, maintaining the enabled flag
	AddInstallations(identity []byte, timestamp int64, installations []*Installation, defaultEnabled bool) ([]*Installation, error)
	// GetInstallations returns all the installations for a given identity
	GetInstallations(identity []byte) ([]*Installation, error)
	// SetInstallationMetadata sets the metadata for a given installation
	SetInstallationMetadata(identity []byte, installationID string, data *InstallationMetadata) error
}
