package accounts

import (
	"github.com/status-im/status-go/protocol/common"
)

type SettingsValueHandler func(interface{}) (interface{}, error)
type SyncSettingProtobufFactory func(string, interface{}, uint64) (*common.RawMessage, error)

type SyncSettingField struct {
	Field SettingField
	Value interface{}
}

type SettingField struct {
	ReactFieldName      string
	DBColumnName        string
	ValueHandler        SettingsValueHandler
	SyncProtobufFactory SyncSettingProtobufFactory
}

var (
	AnonMetricsShouldSend = SettingField{
		ReactFieldName: "anon-metrics/should-send?",
		DBColumnName:   "anon_metrics_should_send",
		ValueHandler:   BoolHandler,
	}
	Appearance = SettingField{
		ReactFieldName: "appearance",
		DBColumnName:   "appearance",
	}
	AutoMessageEnabled = SettingField{
		ReactFieldName: "auto-message-enabled?",
		DBColumnName: "auto_message_enabled",
		ValueHandler: BoolHandler,
	}
	BackupEnabled = SettingField{
		ReactFieldName: "backup-enabled?",
		DBColumnName:   "backup_enabled",
		ValueHandler:   BoolHandler,
	}
	ChaosMode = SettingField{
		ReactFieldName: "chaos-mode?",
		DBColumnName:   "chaos_mode",
		ValueHandler:   BoolHandler,
	}
	Currency = SettingField{
		ReactFieldName:      "currency",
		DBColumnName:        "currency",
		SyncProtobufFactory: currencyProtobufFactory,
	}
	CurrentUserStatus = SettingField{
		ReactFieldName: "current-user-status",
		DBColumnName:   "current_user_status",
		ValueHandler:   JSONBlobHandler,
	}
	CustomBootNodes = SettingField{
		ReactFieldName: "custom-bootnodes",
		DBColumnName:   "custom_bootnodes",
		ValueHandler:   JSONBlobHandler,
	}
	CustomBootNodesEnabled = SettingField{
		ReactFieldName: "custom-bootnodes-enabled?",
		DBColumnName:   "custom_bootnodes_enabled",
		ValueHandler:   JSONBlobHandler,
	}
	DappsAddress = SettingField{
		ReactFieldName: "dapps-address",
		DBColumnName:   "dapps_address",
		ValueHandler:   AddressHandler,
	}
	DefaultSyncPeriod = SettingField{
		ReactFieldName: "default-sync-period",
		DBColumnName:   "default_sync_period",
	}
	DisplayName = SettingField{
		ReactFieldName: "display-name",
		DBColumnName:   "display_name",
	}
	EIP1581Address = SettingField{
		ReactFieldName: "eip1581-address",
		DBColumnName:   "eip1581_address",
		ValueHandler:   AddressHandler,
	}
	Fleet = SettingField{
		ReactFieldName: "fleet",
		DBColumnName:   "fleet",
	}
	GifAPIKey = SettingField{
		ReactFieldName: "gifs/api-key",
		DBColumnName:   "gif_api_key",
		ValueHandler:   BoolHandler,
		ShouldSync:     true,
	}
	GifRecents = SettingField{
		ReactFieldName:      "gifs/recent-gifs",
		DBColumnName:        "gif_recents",
		ValueHandler:        JSONBlobHandler,
		SyncProtobufFactory: gifRecentsProtobufFactory,
	}
	GifFavourites = SettingField{
		ReactFieldName:      "gifs/favorite-gifs",
		DBColumnName:        "gif_favorites",
		ValueHandler:        JSONBlobHandler,
		SyncProtobufFactory: gifFavouritesProtobufFactory,
	}
	HideHomeTooltip = SettingField{
		ReactFieldName: "hide-home-tooltip?",
		DBColumnName:   "hide_home_tooltip",
		ValueHandler:   BoolHandler,
	}
	KeycardInstanceUID = SettingField{
		ReactFieldName: "keycard-instance_uid",
		DBColumnName:   "keycard_instance_uid",
	}
	KeycardPairedOn = SettingField{
		ReactFieldName: "keycard-paired_on",
		DBColumnName:   "keycard_paired_on",
	}
	KeycardPairing = SettingField{
		ReactFieldName: "keycard-pairing",
		DBColumnName:   "keycard_pairing",
	}
	LastUpdated = SettingField{
		ReactFieldName: "last-updated",
		DBColumnName:   "last_updated",
	}
	LatestDerivedPath = SettingField{
		ReactFieldName: "latest-derived-path",
		DBColumnName:   "latest_derived_path",
	}
	LinkPreviewRequestEnabled = SettingField{
		ReactFieldName: "link-preview-request-enabled",
		DBColumnName:   "link_preview_request_enabled",
		ValueHandler:   BoolHandler,
	}
	LinkPreviewsEnabledSites = SettingField{
		ReactFieldName: "link-previews-enabled-sites",
		DBColumnName:   "link_previews_enabled_sites",
		ValueHandler:   JSONBlobHandler,
	}
	LogLevel = SettingField{
		ReactFieldName: "log-level",
		DBColumnName:   "log_level",
	}
	MessagesFromContactsOnly = SettingField{
		ReactFieldName:      "messages-from-contacts-only",
		DBColumnName:        "messages_from_contacts_only",
		ValueHandler:        BoolHandler,
		SyncProtobufFactory: messagesFromContactsOnlyProtobufFactory,
	}
	Mnemonic = SettingField{
		ReactFieldName: "mnemonic",
		DBColumnName:   "mnemonic",
	}
	Name = SettingField{
		ReactFieldName: "name",
		DBColumnName:   "name",
	}
	NetworksCurrentNetwork = SettingField{
		ReactFieldName: "networks/current-network",
		DBColumnName:   "current_network",
	}
	NetworksNetworks = SettingField{
		ReactFieldName: "networks/networks",
		DBColumnName:   "networks",
		ValueHandler:   JSONBlobHandler,
	}
	NodeConfig = SettingField{
		ReactFieldName: "node-config",
		DBColumnName:   "node_config",
		ValueHandler:   NodeConfigHandler,
	}
	NotificationsEnabled = SettingField{
		ReactFieldName: "notifications-enabled?",
		DBColumnName:   "notifications_enabled",
		ValueHandler:   BoolHandler,
	}
	OpenseaEnabled = SettingField{
		ReactFieldName: "opensea-enabled?",
		DBColumnName:   "opensea_enabled",
		ValueHandler:   BoolHandler,
	}
	PhotoPath = SettingField{
		ReactFieldName: "photo-path",
		DBColumnName:   "photo_path",
	}
	PinnedMailservers = SettingField{
		ReactFieldName: "pinned-mailservers",
		DBColumnName:   "pinned_mailservers",
		ValueHandler:   JSONBlobHandler,
	}
	PreferredName = SettingField{
		ReactFieldName:      "preferred-name",
		DBColumnName:        "preferred_name",
		SyncProtobufFactory: preferredNameProtobufFactory,
	}
	PreviewPrivacy = SettingField{
		ReactFieldName:      "preview-privacy?",
		DBColumnName:        "preview_privacy",
		ValueHandler:        BoolHandler,
		SyncProtobufFactory: previewPrivacyProtobufFactory,
	}
	ProfilePicturesShowTo = SettingField{
		ReactFieldName:      "profile-pictures-show-to",
		DBColumnName:        "profile_pictures_show_to",
		SyncProtobufFactory: profilePicturesShowToProtobufFactory,
	}
	ProfilePicturesVisibility = SettingField{
		ReactFieldName:      "profile-pictures-visibility",
		DBColumnName:        "profile_pictures_visibility",
		SyncProtobufFactory: profilePicturesVisibilityProtobufFactory,
	}
	PublicKey = SettingField{
		ReactFieldName: "public-key",
		DBColumnName:   "public_key",
	}
	PushNotificationsBlockMentions = SettingField{
		ReactFieldName: "push-notifications-block-mentions?",
		DBColumnName:   "push_notifications_block_mentions",
		ValueHandler:   BoolHandler,
	}
	PushNotificationsFromContactsOnly = SettingField{
		ReactFieldName: "push-notifications-from-contacts-only?",
		DBColumnName:   "push_notifications_from_contacts_only",
		ValueHandler:   BoolHandler,
	}
	PushNotificationsServerEnabled = SettingField{
		ReactFieldName: "push-notifications-server-enabled?",
		DBColumnName:   "push_notifications_server_enabled",
		ValueHandler:   BoolHandler,
	}
	RememberSyncingChoice = SettingField{
		ReactFieldName: "remember-syncing-choice?",
		DBColumnName:   "remember_syncing_choice",
		ValueHandler:   BoolHandler,
	}
	RemotePushNotificationsEnabled = SettingField{
		ReactFieldName: "remote-push-notifications-enabled?",
		DBColumnName:   "remote_push_notifications_enabled",
		ValueHandler:   BoolHandler,
	}
	SendPushNotifications = SettingField{
		ReactFieldName: "send-push-notifications?",
		DBColumnName:   "send_push_notifications",
		ValueHandler:   BoolHandler,
	}
	SendStatusUpdates = SettingField{
		ReactFieldName:      "send-status-updates?",
		DBColumnName:        "send_status_updates",
		ValueHandler:        BoolHandler,
		SyncProtobufFactory: sendStatusUpdatesProtobufFactory,
	}
	StickersPacksInstalled = SettingField{
		ReactFieldName:      "stickers/packs-installed",
		DBColumnName:        "stickers_packs_installed",
		ValueHandler:        JSONBlobHandler,
		SyncProtobufFactory: stickersPacksInstalledProtobufFactory,
	}
	StickersPacksPending = SettingField{
		ReactFieldName:      "stickers/packs-pending",
		DBColumnName:        "stickers_packs_pending",
		ValueHandler:        JSONBlobHandler,
		SyncProtobufFactory: stickersPacksPendingProtobufFactory,
	}
	StickersRecentStickers = SettingField{
		ReactFieldName:      "stickers/recent-stickers",
		DBColumnName:        "stickers_recent_stickers",
		ValueHandler:        JSONBlobHandler,
		SyncProtobufFactory: stickersRecentStickersProtobufFactory,
	}
	SyncingOnMobileNetwork = SettingField{
		ReactFieldName: "syncing-on-mobile-network?",
		DBColumnName:   "syncing_on_mobile_network",
		ValueHandler:   BoolHandler,
	}
	TelemetryServerURL = SettingField{
		ReactFieldName:      "telemetry-server-url",
		DBColumnName:        "telemetry_server_url",
		SyncProtobufFactory: telemetryServerURLProtobufFactory,
	}
	UseMailservers = SettingField{
		ReactFieldName: "use-mailservers?",
		DBColumnName:   "use_mailservers",
		ValueHandler:   BoolHandler,
	}
	Usernames = SettingField{
		ReactFieldName: "usernames",
		DBColumnName:   "usernames",
		ValueHandler:   JSONBlobHandler,
	}
	WakuBloomFilterMode = SettingField{
		ReactFieldName: "waku-bloom-filter-mode",
		DBColumnName:   "waku_bloom_filter_mode",
		ValueHandler:   BoolHandler,
	}
	WalletSetUpPassed = SettingField{
		ReactFieldName: "wallet-set-up-passed?",
		DBColumnName:   "wallet_set_up_passed",
		ValueHandler:   BoolHandler,
	}
	WalletVisibleTokens = SettingField{
		ReactFieldName: "wallet/visible-tokens",
		DBColumnName:   "wallet_visible_tokens",
		ValueHandler:   JSONBlobHandler,
	}
	WebviewAllowPermissionRequests = SettingField{
		ReactFieldName: "webview-allow-permission-requests?",
		DBColumnName:   "webview_allow_permission_requests",
		ValueHandler:   BoolHandler,
	}

	SettingFieldRegister = []SettingField{
		AnonMetricsShouldSend,
		Appearance,
		AutoMessageEnabled,
		BackupEnabled,
		ChaosMode,
		Currency,
		CurrentUserStatus,
		CustomBootNodes,
		CustomBootNodesEnabled,
		DappsAddress,
		DefaultSyncPeriod,
		DisplayName,
		EIP1581Address,
		Fleet,
		GifAPIKey,
		GifRecents,
		GifFavourites,
		HideHomeTooltip,
		KeycardInstanceUID,
		KeycardPairedOn,
		KeycardPairing,
		LastUpdated,
		LatestDerivedPath,
		LinkPreviewRequestEnabled,
		LinkPreviewsEnabledSites,
		LogLevel,
		MessagesFromContactsOnly,
		Mnemonic,
		Name,
		NetworksCurrentNetwork,
		NetworksNetworks,
		NodeConfig,
		NotificationsEnabled,
		OpenseaEnabled,
		PhotoPath,
		PinnedMailservers,
		PreferredName,
		PreviewPrivacy,
		ProfilePicturesShowTo,
		ProfilePicturesVisibility,
		PublicKey,
		PushNotificationsBlockMentions,
		PushNotificationsFromContactsOnly,
		PushNotificationsServerEnabled,
		RememberSyncingChoice,
		RemotePushNotificationsEnabled,
		SendPushNotifications,
		SendStatusUpdates,
		StickersPacksInstalled,
		StickersPacksPending,
		StickersRecentStickers,
		SyncingOnMobileNetwork,
		TelemetryServerURL,
		UseMailservers,
		Usernames,
		WakuBloomFilterMode,
		WalletSetUpPassed,
		WalletVisibleTokens,
		WebviewAllowPermissionRequests,
	}
)
