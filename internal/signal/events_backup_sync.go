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
	CurrentSession().SendBackedUpProfile(obj)
}

// SendBackedUpProfile sends in the session s.
func (s Session) SendBackedUpProfile(obj json.Marshaler) {
	s.send(EventBackedUpProfile, obj)
}

func SendBackedUpSettings(obj json.Marshaler) {
	CurrentSession().SendBackedUpSettings(obj)
}

// SendBackedUpSettings sends in the session s.
func (s Session) SendBackedUpSettings(obj json.Marshaler) {
	s.send(EventBackedUpSettings, obj)
}

func SendLocalBackUpCompleted(obj json.Marshaler) {
	CurrentSession().SendLocalBackUpCompleted(obj)
}

// SendLocalBackUpCompleted sends in the session s.
func (s Session) SendLocalBackUpCompleted(obj json.Marshaler) {
	s.send(EventLocalBackUpCompleted, obj)
}
