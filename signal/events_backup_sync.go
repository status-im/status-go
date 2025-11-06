package signal

import "encoding/json"

const (
	// EventSyncFromWakuProfile is emitted while applying backed up profile data
	EventBackedUpProfile = "backedup.profile"

	// EventBackedUpSettings is emitted while applying backed up settings
	EventBackedUpSettings = "backedup.settings"

	EventLocalBackUpCompleted = "backup.local.completed"
)

func SendBackedUpProfile(obj json.Marshaler) {
	send(EventBackedUpProfile, obj)
}

func SendBackedUpSettings(obj json.Marshaler) {
	send(EventBackedUpSettings, obj)
}

func SendLocalBackUpCompleted(obj json.Marshaler) {
	send(EventLocalBackUpCompleted, obj)
}
