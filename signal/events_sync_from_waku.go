package signal

import "encoding/json"

const (
	// EventWakuFetchingBackupProgress is emitted while applying fetched data is ongoing
	EventWakuFetchingBackupProgress = "waku.fetching.backup.progress"

	// EventSyncFromWakuProfile is emitted while applying fetched profile data from waku
	EventWakuBackedUpProfile = "waku.backedup.profile"

	// EventWakuBackedUpSettings is emitted while applying fetched settings from waku
	EventWakuBackedUpSettings = "waku.backedup.settings"

	// EventWakuBackedUpKeycards is emitted while applying fetched keycard data from waku
	EventWakuBackedUpKeycards = "waku.backedup.keycards"
)

func SendWakuFetchingBackupProgress(obj json.Marshaler) {
	send(EventWakuFetchingBackupProgress, obj)
}

func SendWakuBackedUpProfile(obj json.Marshaler) {
	send(EventWakuBackedUpProfile, obj)
}

func SendWakuBackedUpSettings(obj json.Marshaler) {
	send(EventWakuBackedUpSettings, obj)
}

func SendWakuBackedUpKeycards(obj json.Marshaler) {
	send(EventWakuBackedUpKeycards, obj)
}
