package signal

import "encoding/json"

const (
	// EventWakuFetchingBackupProgress is triggered during the syncing from waku
	EventWakuFetchingBackupProgress = "waku.fetching.backup.progress"

	// EventSyncFromWakuProfile is triggered during the syncing user profile from waku
	EventWakuBackedUpProfile = "waku.backedup.profile"

	// EventWakuBackedUpSettings is triggered during the syncing user settings from waku
	EventWakuBackedUpSettings = "waku.backedup.settings"
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
