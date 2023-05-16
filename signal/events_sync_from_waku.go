package signal

import "encoding/json"

const (
	// EventWakuFetchingBackupProgress is emitted while applying fetched data is ongoing
	EventWakuFetchingBackupProgress = "waku.fetching.backup.progress"

	// EventSyncFromWakuProfile is emitted while applying fetched profile data from waku
	EventWakuBackedUpProfile = "waku.backedup.profile"

	// EventWakuBackedUpSettings is emitted while applying fetched settings from waku
	EventWakuBackedUpSettings = "waku.backedup.settings"

	// EventWakuBackedUpKeypair is emitted while applying fetched keypair data from waku
	EventWakuBackedUpKeypair = "waku.backedup.keypair"

	// EventWakuBackedUpKeycards is emitted while applying fetched keycard data from waku
	EventWakuBackedUpKeycards = "waku.backedup.keycards"

	// EventWakuBackedUpWatchOnlyAccount is emitted while applying fetched watch only account data from waku
	EventWakuBackedUpWatchOnlyAccount = "waku.backedup.watch-only-account" // #nosec G101
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

func SendWakuBackedUpKeypair(obj json.Marshaler) {
	send(EventWakuBackedUpKeypair, obj)
}

func SendWakuBackedUpKeycards(obj json.Marshaler) {
	send(EventWakuBackedUpKeycards, obj)
}

func SendWakuBackedUpWatchOnlyAccount(obj json.Marshaler) {
	send(EventWakuBackedUpWatchOnlyAccount, obj)
}
