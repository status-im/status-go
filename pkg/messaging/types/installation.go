package types

type InstallationMetadata struct {
	// The name of the device
	Name string `json:"name"`
	// The type of device
	DeviceType string `json:"deviceType"`
}

type Installation struct {
	// Identity is the string identity of the owner
	Identity string `json:"identity"`
	// The installation-id of the device
	ID string `json:"id"`
	// The last known protocol version of the device
	Version uint32 `json:"version"`
	// Enabled is whether the installation is enabled
	Enabled bool `json:"enabled"`
	// Timestamp is the last time we saw this device
	Timestamp int64 `json:"timestamp"`
	// InstallationMetadata
	InstallationMetadata *InstallationMetadata `json:"metadata"`
}

func (i *Installation) UniqueKey() string {
	return i.ID + i.Identity
}
