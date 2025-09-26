package signal

import "encoding/json"

// TODO rename those events to remove the waku part as we are not using waku anymore for backups
const (
	// EventSyncFromWakuProfile is emitted while applying fetched profile data from waku
	EventWakuBackedUpProfile = "waku.backedup.profile"

	// EventWakuBackedUpSettings is emitted while applying fetched settings from waku
	EventWakuBackedUpSettings = "waku.backedup.settings"

	// EventWakuBackedUpKeypair is emitted while applying fetched keypair data from waku
	EventWakuBackedUpKeypair = "waku.backedup.keypair"

	// EventWakuBackedUpWatchOnlyAccount is emitted while applying fetched watch only account data from waku
	EventWakuBackedUpWatchOnlyAccount = "waku.backedup.watch-only-account" // #nosec G101
)

func SendWakuBackedUpProfile(obj json.Marshaler) {
	send(EventWakuBackedUpProfile, obj)
}

func SendWakuBackedUpSettings(obj json.Marshaler) {
	send(EventWakuBackedUpSettings, obj)
}

func SendWakuBackedUpKeypair(obj json.Marshaler) {
	send(EventWakuBackedUpKeypair, obj)
}

func SendWakuBackedUpWatchOnlyAccount(obj json.Marshaler) {
	send(EventWakuBackedUpWatchOnlyAccount, obj)
}
