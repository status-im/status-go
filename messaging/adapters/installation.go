package adapters

import (
	"github.com/status-im/status-go/messaging/layers/encryption/multidevice"
	"github.com/status-im/status-go/messaging/types"
)

func FromEncryptionInstallation(installation *multidevice.Installation) *types.Installation {
	if installation == nil {
		return nil
	}

	return &types.Installation{
		Identity:             installation.Identity,
		ID:                   installation.ID,
		Version:              installation.Version,
		Enabled:              installation.Enabled,
		Timestamp:            installation.Timestamp,
		InstallationMetadata: FromEncryptionInstallationMetadata(installation.InstallationMetadata),
	}
}

func FromEncryptionInstallationMetadata(metadata *multidevice.InstallationMetadata) *types.InstallationMetadata {
	if metadata == nil {
		return nil
	}

	return &types.InstallationMetadata{
		Name:       metadata.Name,
		DeviceType: metadata.DeviceType,
	}
}

func FromEncryptionInstallations(installations []*multidevice.Installation) []*types.Installation {
	if installations == nil {
		return nil
	}

	result := make([]*types.Installation, 0, len(installations))
	for _, installation := range installations {
		result = append(result, FromEncryptionInstallation(installation))
	}

	return result
}

func ToEncryptionInstallation(installation *types.Installation) *multidevice.Installation {
	if installation == nil {
		return nil
	}

	return &multidevice.Installation{
		Identity:             installation.Identity,
		ID:                   installation.ID,
		Version:              installation.Version,
		Enabled:              installation.Enabled,
		Timestamp:            installation.Timestamp,
		InstallationMetadata: ToEncryptionInstallationMetadata(installation.InstallationMetadata),
	}
}

func ToEncryptionInstallations(installations []*types.Installation) []*multidevice.Installation {
	if installations == nil {
		return nil
	}

	result := make([]*multidevice.Installation, 0, len(installations))
	for _, installation := range installations {
		result = append(result, ToEncryptionInstallation(installation))
	}

	return result
}

func ToEncryptionInstallationMetadata(metadata *types.InstallationMetadata) *multidevice.InstallationMetadata {
	if metadata == nil {
		return nil
	}

	return &multidevice.InstallationMetadata{
		Name:       metadata.Name,
		DeviceType: metadata.DeviceType,
	}
}
