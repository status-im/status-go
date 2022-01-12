package accounts

import (
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/status-im/status-go/protocol/protobuf"
)

type SyncSettingProtobuf interface {
	proto.Message

	GetClock() *timestamppb.Timestamp
	//GetValue() interface{} TODO maybe generics can help make this work
}

type SettingsValueHandler func(interface{}) (interface{}, error)
type SyncSettingProtobufFactory func(interface{}, time.Time) SyncSettingProtobuf

type SyncSettingField struct {
	Field SettingField
	Value interface{}
}

type SettingField struct {
	ReactFieldName string
	DBColumnName   string
	ValueHandler   SettingsValueHandler
	ShouldSync     bool
	SyncProtobuf   SyncSettingProtobufFactory
}

var (
	AnonMetricsShouldSend = SettingField{
		ReactFieldName: "anon-metrics/should-send?",
		DBColumnName: "anon_metrics_should_send",
		ValueHandler: BoolHandler,
		ShouldSync: false,
		SyncProtobuf: nil,
	}
	Appearance = SettingField{
		ReactFieldName: "appearance",
		DBColumnName: "appearance",
		ValueHandler: nil,
		ShouldSync: false,
		SyncProtobuf: nil,
	}
	AutoMessageEnabled = SettingField{
		"auto-message-enabled?",
		"auto_message_enabled",
		BoolHandler,
		false,
		nil,
	}
	BackupEnabled = SettingField{
		ReactFieldName: "backup-enabled?",
		DBColumnName: "backup_enabled",
		ValueHandler: BoolHandler,
		ShouldSync: false,
		SyncProtobuf: nil,
	}
	ChaosMode = SettingField{
		ReactFieldName: "chaos-mode?",
		DBColumnName: "chaos_mode",
		ValueHandler: BoolHandler,
		ShouldSync: false,
		SyncProtobuf: nil,
	}
	Currency = SettingField{
		ReactFieldName: "currency",
		DBColumnName:   "currency",
		ValueHandler:   nil,
		ShouldSync:     true,
		SyncProtobuf: currencyProtobufFactory,
	}
	CurrentUserStatus = SettingField{
		ReactFieldName: "current-user-status",
		DBColumnName:   "current_user_status",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	CustomBootNodes = SettingField{
		ReactFieldName: "custom-bootnodes",
		DBColumnName:   "custom_bootnodes",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	CustomBootNodesEnabled = SettingField{
		ReactFieldName: "custom-bootnodes-enabled?",
		DBColumnName:   "custom_bootnodes_enabled",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	DappsAddress = SettingField{
		ReactFieldName: "dapps-address",
		DBColumnName:   "dapps_address",
		ValueHandler:   AddressHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	DefaultSyncPeriod = SettingField{
		ReactFieldName: "default-sync-period",
		DBColumnName:   "default_sync_period",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	DisplayName = SettingField{
		ReactFieldName: "display-name",
		DBColumnName:   "display_name",
	}
	EIP1581Address = SettingField{
		ReactFieldName: "eip1581-address",
		DBColumnName:   "eip1581_address",
		ValueHandler:   AddressHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	Fleet = SettingField{
		ReactFieldName: "fleet",
		DBColumnName:   "fleet",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	GifAPIKey = SettingField{
		ReactFieldName: "gifs/api-key",
		DBColumnName:   "gif_api_key",
		ValueHandler:   BoolHandler,
		ShouldSync:     true,
	}
	GifRecents = SettingField{
		ReactFieldName: "gifs/recent-gifs",
		DBColumnName:   "gif_recents",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     true,
		SyncProtobuf:   gifRecentsProtobufFactory,
	}
	GifFavourites = SettingField{
		ReactFieldName: "gifs/favorite-gifs",
		DBColumnName:   "gif_favorites",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     true,
		SyncProtobuf:   gifFavouritesProtobufFactory,
	}
	HideHomeTooltip = SettingField{
		ReactFieldName: "hide-home-tooltip?",
		DBColumnName:   "hide_home_tooltip",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	KeycardInstanceUID = SettingField{
		ReactFieldName: "keycard-instance_uid",
		DBColumnName:   "keycard_instance_uid",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	KeycardPairedOn = SettingField{
		ReactFieldName: "keycard-paired_on",
		DBColumnName:   "keycard_paired_on",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	KeycardPairing = SettingField{
		ReactFieldName: "keycard-pairing",
		DBColumnName:   "keycard_pairing",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	LastUpdated = SettingField{
		ReactFieldName: "last-updated",
		DBColumnName:   "last_updated",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	LatestDerivedPath = SettingField{
		ReactFieldName: "latest-derived-path",
		DBColumnName:   "latest_derived_path",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	LinkPreviewRequestEnabled = SettingField{
		ReactFieldName: "link-preview-request-enabled",
		DBColumnName:   "link_preview_request_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	LinkPreviewsEnabledSites = SettingField{
		ReactFieldName: "link-previews-enabled-sites",
		DBColumnName:   "link_previews_enabled_sites",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	LogLevel = SettingField{
		ReactFieldName: "log-level",
		DBColumnName:   "log_level",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	MessagesFromContactsOnly = SettingField{
		ReactFieldName: "messages-from-contacts-only",
		DBColumnName:   "messages_from_contacts_only",
		ValueHandler:   BoolHandler,
		ShouldSync:     true,
		SyncProtobuf:   messagesFromContactsOnlyProtobufFactory,
	}
	Mnemonic = SettingField{
		ReactFieldName: "mnemonic",
		DBColumnName:   "mnemonic",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	Name = SettingField{
		ReactFieldName: "name",
		DBColumnName:   "name",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	NetworksCurrentNetwork = SettingField{
		ReactFieldName: "networks/current-network",
		DBColumnName:   "current_network",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	NetworksNetworks = SettingField{
		ReactFieldName: "networks/networks",
		DBColumnName:   "networks",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	NodeConfig = SettingField{
		ReactFieldName: "node-config",
		DBColumnName:   "node_config",
		ValueHandler:   NodeConfigHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	NotificationsEnabled = SettingField{
		ReactFieldName: "notifications-enabled?",
		DBColumnName:   "notifications_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	OpenseaEnabled = SettingField{
		ReactFieldName: "opensea-enabled?",
		DBColumnName:   "opensea_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	PhotoPath = SettingField{
		ReactFieldName: "photo-path",
		DBColumnName:   "photo_path",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	PinnedMailservers = SettingField{
		ReactFieldName: "pinned-mailservers",
		DBColumnName:   "pinned_mailservers",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	PreferredName = SettingField{
		ReactFieldName: "preferred-name",
		DBColumnName:   "preferred_name",
		ValueHandler:   nil,
		ShouldSync:     true,
		SyncProtobuf:   preferredNameProtobufFactory,
	}
	PreviewPrivacy = SettingField{
		ReactFieldName: "preview-privacy?",
		DBColumnName:   "preview_privacy",
		ValueHandler:   BoolHandler,
		ShouldSync:     true,
		SyncProtobuf:   previewPrivacyProtobufFactory,
	}
	ProfilePicturesShowTo = SettingField{
		ReactFieldName: "profile-pictures-show-to",
		DBColumnName:   "profile_pictures_show_to",
		ValueHandler:   nil,
		ShouldSync:     true,
		SyncProtobuf:   new(protobuf.SyncSettingProfilePicturesShowTo),
	}
	ProfilePicturesVisibility = SettingField{
		ReactFieldName: "profile-pictures-visibility",
		DBColumnName:   "profile_pictures_visibility",
		ValueHandler:   nil,
		ShouldSync:     true,
		SyncProtobuf:   new(protobuf.SyncSettingProfilePicturesVisibility),
	}
	PublicKey = SettingField{
		ReactFieldName: "public-key",
		DBColumnName:   "public_key",
		ValueHandler:   nil,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	PushNotificationsBlockMentions = SettingField{
		ReactFieldName: "push-notifications-block-mentions?",
		DBColumnName:   "push_notifications_block_mentions",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	PushNotificationsFromContactsOnly = SettingField{
		ReactFieldName: "push-notifications-from-contacts-only?",
		DBColumnName:   "push_notifications_from_contacts_only",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	PushNotificationsServerEnabled = SettingField{
		ReactFieldName: "push-notifications-server-enabled?",
		DBColumnName:   "push_notifications_server_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	RememberSyncingChoice = SettingField{
		ReactFieldName: "remember-syncing-choice?",
		DBColumnName:   "remember_syncing_choice",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	RemotePushNotificationsEnabled = SettingField{
		ReactFieldName: "remote-push-notifications-enabled?",
		DBColumnName:   "remote_push_notifications_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	SendPushNotifications = SettingField{
		ReactFieldName: "send-push-notifications?",
		DBColumnName:   "send_push_notifications",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	SendStatusUpdates = SettingField{
		ReactFieldName: "send-status-updates?",
		DBColumnName:   "send_status_updates",
		ValueHandler:   BoolHandler,
		ShouldSync:     true,
		SyncProtobuf:   new(protobuf.SyncSettingSendStatusUpdates),
	}
	StickersPacksInstalled = SettingField{
		ReactFieldName: "stickers/packs-installed",
		DBColumnName:   "stickers_packs_installed",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     true,
		SyncProtobuf:   new(protobuf.SyncSettingStickerPacksInstalled),
	}
	StickersPacksPending = SettingField{
		ReactFieldName: "stickers/packs-pending",
		DBColumnName:   "stickers_packs_pending",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     true,
		SyncProtobuf:   new(protobuf.SyncSettingStickerPacksPending),
	}
	StickersRecentStickers = SettingField{
		ReactFieldName: "stickers/recent-stickers",
		DBColumnName:   "stickers_recent_stickers",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     true,
		SyncProtobuf:   new(protobuf.SyncSettingStickersRecentStickers),
	}
	SyncingOnMobileNetwork = SettingField{
		ReactFieldName: "syncing-on-mobile-network?",
		DBColumnName:   "syncing_on_mobile_network",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	TelemetryServerURL = SettingField{
		ReactFieldName: "telemetry-server-url",
		DBColumnName:   "telemetry_server_url",
		ValueHandler:   nil,
		ShouldSync:     true,
		SyncProtobuf:   new(protobuf.SyncSettingTelemetryServerURL),
	}
	UseMailservers = SettingField{
		ReactFieldName: "use-mailservers?",
		DBColumnName:   "use_mailservers",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	Usernames = SettingField{
		ReactFieldName: "usernames",
		DBColumnName:   "usernames",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	WakuBloomFilterMode = SettingField{
		ReactFieldName: "waku-bloom-filter-mode",
		DBColumnName:   "waku_bloom_filter_mode",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	WalletSetUpPassed = SettingField{
		ReactFieldName: "wallet-set-up-passed?",
		DBColumnName:   "wallet_set_up_passed",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	WalletVisibleTokens = SettingField{
		ReactFieldName: "wallet/visible-tokens",
		DBColumnName:   "wallet_visible_tokens",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
	}
	WebviewAllowPermissionRequests = SettingField{
		ReactFieldName: "webview-allow-permission-requests?",
		DBColumnName:   "webview_allow_permission_requests",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
		SyncProtobuf:   nil,
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
