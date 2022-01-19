package settings

import (
	"encoding/json"

	"github.com/status-im/status-go/eth-node/types"
)

type ProfilePicturesVisibilityType int

const (
	ProfilePicturesVisibilityContactsOnly ProfilePicturesVisibilityType = iota + 1
	ProfilePicturesVisibilityEveryone
	ProfilePicturesVisibilityNone
)

type ProfilePicturesShowToType int

const (
	ProfilePicturesShowToContactsOnly ProfilePicturesShowToType = iota + 1
	ProfilePicturesShowToEveryone
	ProfilePicturesShowToNone
)

type Settings struct {
	// required
	Address                   types.Address    `json:"address"`
	AnonMetricsShouldSend     bool             `json:"anon-metrics/should-send?,omitempty"`
	ChaosMode                 bool             `json:"chaos-mode?,omitempty"`
	Currency                  string           `json:"currency,omitempty"`
	CurrentNetwork            string           `json:"networks/current-network"`
	CustomBootnodes           *json.RawMessage `json:"custom-bootnodes,omitempty"`
	CustomBootnodesEnabled    *json.RawMessage `json:"custom-bootnodes-enabled?,omitempty"`
	DappsAddress              types.Address    `json:"dapps-address"`
	EIP1581Address            types.Address    `json:"eip1581-address"`
	Fleet                     *string          `json:"fleet,omitempty"`
	HideHomeTooltip           bool             `json:"hide-home-tooltip?,omitempty"`
	InstallationID            string           `json:"installation-id"`
	KeyUID                    string           `json:"key-uid"`
	KeycardInstanceUID        string           `json:"keycard-instance-uid,omitempty"`
	KeycardPAiredOn           int64            `json:"keycard-paired-on,omitempty"`
	KeycardPairing            string           `json:"keycard-pairing,omitempty"`
	LastUpdated               *int64           `json:"last-updated,omitempty"`
	LatestDerivedPath         uint             `json:"latest-derived-path"`
	LinkPreviewRequestEnabled bool             `json:"link-preview-request-enabled,omitempty"`
	LinkPreviewsEnabledSites  *json.RawMessage `json:"link-previews-enabled-sites,omitempty"`
	LogLevel                  *string          `json:"log-level,omitempty"`
	MessagesFromContactsOnly  bool             `json:"messages-from-contacts-only"`
	Mnemonic                  *string          `json:"mnemonic,omitempty"`
	Name                      string           `json:"name,omitempty"`
	Networks                  *json.RawMessage `json:"networks/networks"`
	// NotificationsEnabled indicates whether local notifications should be enabled (android only)
	NotificationsEnabled bool             `json:"notifications-enabled?,omitempty"`
	PhotoPath            string           `json:"photo-path"`
	PinnedMailserver     *json.RawMessage `json:"pinned-mailservers,omitempty"`
	PreferredName        *string          `json:"preferred-name,omitempty"`
	PreviewPrivacy       bool             `json:"preview-privacy?"`
	PublicKey            string           `json:"public-key"`
	// PushNotificationsServerEnabled indicates whether we should be running a push notification server
	PushNotificationsServerEnabled bool `json:"push-notifications-server-enabled?,omitempty"`
	// PushNotificationsFromContactsOnly indicates whether we should only receive push notifications from contacts
	PushNotificationsFromContactsOnly bool `json:"push-notifications-from-contacts-only?,omitempty"`
	// PushNotificationsBlockMentions indicates whether we should receive notifications for mentions
	PushNotificationsBlockMentions bool `json:"push-notifications-block-mentions?,omitempty"`
	RememberSyncingChoice          bool `json:"remember-syncing-choice?,omitempty"`
	// RemotePushNotificationsEnabled indicates whether we should be using remote notifications (ios only for now)
	RemotePushNotificationsEnabled bool             `json:"remote-push-notifications-enabled?,omitempty"`
	SigningPhrase                  string           `json:"signing-phrase"`
	StickerPacksInstalled          *json.RawMessage `json:"stickers/packs-installed,omitempty"`
	StickerPacksPending            *json.RawMessage `json:"stickers/packs-pending,omitempty"`
	StickersRecentStickers         *json.RawMessage `json:"stickers/recent-stickers,omitempty"`
	SyncingOnMobileNetwork         bool             `json:"syncing-on-mobile-network?,omitempty"`
	// DefaultSyncPeriod is how far back in seconds we should pull messages from a mailserver
	DefaultSyncPeriod uint `json:"default-sync-period"`
	// SendPushNotifications indicates whether we should send push notifications for other clients
	SendPushNotifications bool `json:"send-push-notifications?,omitempty"`
	Appearance            uint `json:"appearance"`
	// ProfilePicturesShowTo indicates to whom the user shows their profile picture to (contacts, everyone)
	ProfilePicturesShowTo ProfilePicturesShowToType `json:"profile-pictures-show-to"`
	// ProfilePicturesVisibility indicates who we want to see profile pictures of (contacts, everyone or none)
	ProfilePicturesVisibility      ProfilePicturesVisibilityType `json:"profile-pictures-visibility"`
	UseMailservers                 bool                          `json:"use-mailservers?"`
	Usernames                      *json.RawMessage              `json:"usernames,omitempty"`
	WalletRootAddress              types.Address                 `json:"wallet-root-address,omitempty"`
	WalletSetUpPassed              bool                          `json:"wallet-set-up-passed?,omitempty"`
	WalletVisibleTokens            *json.RawMessage              `json:"wallet/visible-tokens,omitempty"`
	WakuBloomFilterMode            bool                          `json:"waku-bloom-filter-mode,omitempty"`
	WebViewAllowPermissionRequests bool                          `json:"webview-allow-permission-requests?,omitempty"`
	SendStatusUpdates              bool                          `json:"send-status-updates?,omitempty"`
	CurrentUserStatus              *json.RawMessage              `json:"current-user-status"`
	GifRecents                     *json.RawMessage              `json:"gifs/recent-gifs"`
	GifFavorites                   *json.RawMessage              `json:"gifs/favorite-gifs"`
	OpenseaEnabled                 bool                          `json:"opensea-enabled?,omitempty"`
	TelemetryServerURL             string                        `json:"telemetry-server-url,omitempty"`
	LastBackup                     uint64                        `json:"last-backup,omitempty"`
	BackupEnabled                  bool                          `json:"backup-enabled?,omitempty"`
	AutoMessageEnabled             bool                          `json:"auto-message-enabled?,omitempty"`
}
