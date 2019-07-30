package statusproto

// ContactDeviceInfo is a struct containing information about a particular device owned by a contact
type ContactDeviceInfo struct {
	// The installation id of the device
	InstallationID string `json:"id"`
	// Timestamp represents the last time we received this info
	Timestamp int64 `json:"timestamp"`
	// FCMToken is to be used for push notifications
	FCMToken string `json:"fcmToken"`
}

// Contact has information about a "Contact". A contact is not necessarily one
// that we added or added us, that's based on SystemTags.
type Contact struct {
	// ID of the contact
	ID string `json:"id"`
	// Ethereum address of the contact
	Address string `json:"address"`
	// Name of contact
	Name string `json:"name"`
	// Photo is the base64 encoded photo
	Photo string `json:"photoPath"`
	// LastUpdated is the last time we received an update from the contact
	// updates should be discarded if last updated is less than the one stored
	LastUpdated int64 `json:"lastUpdated"`
	// SystemTags contains information about whether we blocked/added/have been
	// added.
	SystemTags []string `json:"systemTags"`

	DeviceInfo    []ContactDeviceInfo `json:"deviceInfo"`
	TributeToTalk string              `json:"tributeToTalk"`
}
