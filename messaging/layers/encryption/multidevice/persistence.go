package multidevice

type Persistence interface {
	GetActiveInstallations(maxInstallations int, identity []byte) ([]*Installation, error)
	GetInstallations(identity []byte) ([]*Installation, error)
	AddInstallations(identity []byte, timestamp int64, installations []*Installation, defaultEnabled bool) ([]*Installation, error)
	EnableInstallation(identity []byte, installationID string) error
	DisableInstallation(identity []byte, installationID string) error
	SetInstallationMetadata(identity []byte, installationID string, metadata *InstallationMetadata) error
	SetInstallationName(identity []byte, installationID string, name string) error
}
