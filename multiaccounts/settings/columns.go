package settings

import (
	"encoding/json"
	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/protobuf"
)

type ValueHandler func(interface{}) (interface{}, error)
type SyncSettingProtobufFactory func(interface{}, uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error)

type SyncSettingField struct {
	SettingField
	Value interface{}
}

func (s SyncSettingField) MarshalJSON() ([]byte, error) {
	alias := struct{
		Name string `json:"name"`
		Value interface{} `json:"value"`
	}{
		s.reactFieldName,
		s.Value,
	}

	return json.Marshal(alias)
}

type SettingField struct {
	reactFieldName      string
	dBColumnName        string
	valueHandler        ValueHandler
	syncProtobufFactory SyncSettingProtobufFactory
	//storeHandler
}

func (s SettingField) GetReactName() string {
	return s.reactFieldName
}

func (s SettingField) GetDBName() string {
	return s.dBColumnName
}

func (s SettingField) ValueHandler() ValueHandler {
	return s.valueHandler
}

func (s SettingField) SyncProtobufFactory() SyncSettingProtobufFactory {
	return s.syncProtobufFactory
}

var (
	AnonMetricsShouldSend = SettingField{
		reactFieldName: "anon-metrics/should-send?",
		dBColumnName:   "anon_metrics_should_send",
		valueHandler:   BoolHandler,
	}
	Appearance = SettingField{
		reactFieldName: "appearance",
		dBColumnName:   "appearance",
	}
	AutoMessageEnabled = SettingField{
		reactFieldName: "auto-message-enabled?",
		dBColumnName:   "auto_message_enabled",
		valueHandler:   BoolHandler,
	}
	BackupEnabled = SettingField{
		reactFieldName: "backup-enabled?",
		dBColumnName:   "backup_enabled",
		valueHandler:   BoolHandler,
	}
	ChaosMode = SettingField{
		reactFieldName: "chaos-mode?",
		dBColumnName:   "chaos_mode",
		valueHandler:   BoolHandler,
	}
	Currency = SettingField{
		reactFieldName:      "currency",
		dBColumnName:        "currency",
		syncProtobufFactory: currencyProtobufFactory,
	}
	CurrentUserStatus = SettingField{
		reactFieldName: "current-user-status",
		dBColumnName:   "current_user_status",
		valueHandler:   JSONBlobHandler,
	}
	CustomBootNodes = SettingField{
		reactFieldName: "custom-bootnodes",
		dBColumnName:   "custom_bootnodes",
		valueHandler:   JSONBlobHandler,
	}
	CustomBootNodesEnabled = SettingField{
		reactFieldName: "custom-bootnodes-enabled?",
		dBColumnName:   "custom_bootnodes_enabled",
		valueHandler:   JSONBlobHandler,
	}
	DappsAddress = SettingField{
		reactFieldName: "dapps-address",
		dBColumnName:   "dapps_address",
		valueHandler:   AddressHandler,
	}
	DefaultSyncPeriod = SettingField{
		reactFieldName: "default-sync-period",
		dBColumnName:   "default_sync_period",
	}
	DisplayName = SettingField{
		reactFieldName: "display-name",
		dBColumnName:   "display_name",
	}
	EIP1581Address = SettingField{
		reactFieldName: "eip1581-address",
		dBColumnName:   "eip1581_address",
		valueHandler:   AddressHandler,
	}
	Fleet = SettingField{
		reactFieldName: "fleet",
		dBColumnName:   "fleet",
	}
	GifAPIKey = SettingField{
		reactFieldName:      "gifs/api-key",
		dBColumnName:        "gif_api_key",
		syncProtobufFactory: gifAPIKeyProtobufFactory,
	}
	GifFavourites = SettingField{
		reactFieldName:      "gifs/favorite-gifs",
		dBColumnName:        "gif_favorites",
		valueHandler:        JSONBlobHandler,
		syncProtobufFactory: gifFavouritesProtobufFactory,
	}
	GifRecents = SettingField{
		reactFieldName:      "gifs/recent-gifs",
		dBColumnName:        "gif_recents",
		valueHandler:        JSONBlobHandler,
		syncProtobufFactory: gifRecentsProtobufFactory,
	}
	HideHomeTooltip = SettingField{
		reactFieldName: "hide-home-tooltip?",
		dBColumnName:   "hide_home_tooltip",
		valueHandler:   BoolHandler,
	}
	KeycardInstanceUID = SettingField{
		reactFieldName: "keycard-instance_uid",
		dBColumnName:   "keycard_instance_uid",
	}
	KeycardPairedOn = SettingField{
		reactFieldName: "keycard-paired_on",
		dBColumnName:   "keycard_paired_on",
	}
	KeycardPairing = SettingField{
		reactFieldName: "keycard-pairing",
		dBColumnName:   "keycard_pairing",
	}
	LastUpdated = SettingField{
		reactFieldName: "last-updated",
		dBColumnName:   "last_updated",
	}
	LatestDerivedPath = SettingField{
		reactFieldName: "latest-derived-path",
		dBColumnName:   "latest_derived_path",
	}
	LinkPreviewRequestEnabled = SettingField{
		reactFieldName: "link-preview-request-enabled",
		dBColumnName:   "link_preview_request_enabled",
		valueHandler:   BoolHandler,
	}
	LinkPreviewsEnabledSites = SettingField{
		reactFieldName: "link-previews-enabled-sites",
		dBColumnName:   "link_previews_enabled_sites",
		valueHandler:   JSONBlobHandler,
	}
	LogLevel = SettingField{
		reactFieldName: "log-level",
		dBColumnName:   "log_level",
	}
	MessagesFromContactsOnly = SettingField{
		reactFieldName:      "messages-from-contacts-only",
		dBColumnName:        "messages_from_contacts_only",
		valueHandler:        BoolHandler,
		syncProtobufFactory: messagesFromContactsOnlyProtobufFactory,
	}
	Mnemonic = SettingField{
		reactFieldName: "mnemonic",
		dBColumnName:   "mnemonic",
	}
	Name = SettingField{
		reactFieldName: "name",
		dBColumnName:   "name",
	}
	NetworksCurrentNetwork = SettingField{
		reactFieldName: "networks/current-network",
		dBColumnName:   "current_network",
	}
	NetworksNetworks = SettingField{
		reactFieldName: "networks/networks",
		dBColumnName:   "networks",
		valueHandler:   JSONBlobHandler,
	}
	NodeConfig = SettingField{
		reactFieldName: "node-config",
		dBColumnName:   "node_config",
		valueHandler:   NodeConfigHandler,
	}
	NotificationsEnabled = SettingField{
		reactFieldName: "notifications-enabled?",
		dBColumnName:   "notifications_enabled",
		valueHandler:   BoolHandler,
	}
	OpenseaEnabled = SettingField{
		reactFieldName: "opensea-enabled?",
		dBColumnName:   "opensea_enabled",
		valueHandler:   BoolHandler,
	}
	PhotoPath = SettingField{
		reactFieldName: "photo-path",
		dBColumnName:   "photo_path",
	}
	PinnedMailservers = SettingField{
		reactFieldName: "pinned-mailservers",
		dBColumnName:   "pinned_mailservers",
		valueHandler:   JSONBlobHandler,
	}
	PreferredName = SettingField{
		reactFieldName:      "preferred-name",
		dBColumnName:        "preferred_name",
		syncProtobufFactory: preferredNameProtobufFactory,
	}
	PreviewPrivacy = SettingField{
		reactFieldName:      "preview-privacy?",
		dBColumnName:        "preview_privacy",
		valueHandler:        BoolHandler,
		syncProtobufFactory: previewPrivacyProtobufFactory,
	}
	ProfilePicturesShowTo = SettingField{
		reactFieldName:      "profile-pictures-show-to",
		dBColumnName:        "profile_pictures_show_to",
		syncProtobufFactory: profilePicturesShowToProtobufFactory,
	}
	ProfilePicturesVisibility = SettingField{
		reactFieldName:      "profile-pictures-visibility",
		dBColumnName:        "profile_pictures_visibility",
		syncProtobufFactory: profilePicturesVisibilityProtobufFactory,
	}
	PublicKey = SettingField{
		reactFieldName: "public-key",
		dBColumnName:   "public_key",
	}
	PushNotificationsBlockMentions = SettingField{
		reactFieldName: "push-notifications-block-mentions?",
		dBColumnName:   "push_notifications_block_mentions",
		valueHandler:   BoolHandler,
	}
	PushNotificationsFromContactsOnly = SettingField{
		reactFieldName: "push-notifications-from-contacts-only?",
		dBColumnName:   "push_notifications_from_contacts_only",
		valueHandler:   BoolHandler,
	}
	PushNotificationsServerEnabled = SettingField{
		reactFieldName: "push-notifications-server-enabled?",
		dBColumnName:   "push_notifications_server_enabled",
		valueHandler:   BoolHandler,
	}
	RememberSyncingChoice = SettingField{
		reactFieldName: "remember-syncing-choice?",
		dBColumnName:   "remember_syncing_choice",
		valueHandler:   BoolHandler,
	}
	RemotePushNotificationsEnabled = SettingField{
		reactFieldName: "remote-push-notifications-enabled?",
		dBColumnName:   "remote_push_notifications_enabled",
		valueHandler:   BoolHandler,
	}
	SendPushNotifications = SettingField{
		reactFieldName: "send-push-notifications?",
		dBColumnName:   "send_push_notifications",
		valueHandler:   BoolHandler,
	}
	SendStatusUpdates = SettingField{
		reactFieldName:      "send-status-updates?",
		dBColumnName:        "send_status_updates",
		valueHandler:        BoolHandler,
		syncProtobufFactory: sendStatusUpdatesProtobufFactory,
	}
	StickersPacksInstalled = SettingField{
		reactFieldName:      "stickers/packs-installed",
		dBColumnName:        "stickers_packs_installed",
		valueHandler:        JSONBlobHandler,
		syncProtobufFactory: stickersPacksInstalledProtobufFactory,
	}
	StickersPacksPending = SettingField{
		reactFieldName:      "stickers/packs-pending",
		dBColumnName:        "stickers_packs_pending",
		valueHandler:        JSONBlobHandler,
		syncProtobufFactory: stickersPacksPendingProtobufFactory,
	}
	StickersRecentStickers = SettingField{
		reactFieldName:      "stickers/recent-stickers",
		dBColumnName:        "stickers_recent_stickers",
		valueHandler:        JSONBlobHandler,
		syncProtobufFactory: stickersRecentStickersProtobufFactory,
	}
	SyncingOnMobileNetwork = SettingField{
		reactFieldName: "syncing-on-mobile-network?",
		dBColumnName:   "syncing_on_mobile_network",
		valueHandler:   BoolHandler,
	}
	TelemetryServerURL = SettingField{
		reactFieldName:      "telemetry-server-url",
		dBColumnName:        "telemetry_server_url",
		syncProtobufFactory: telemetryServerURLProtobufFactory,
	}
	UseMailservers = SettingField{
		reactFieldName: "use-mailservers?",
		dBColumnName:   "use_mailservers",
		valueHandler:   BoolHandler,
	}
	Usernames = SettingField{
		reactFieldName: "usernames",
		dBColumnName:   "usernames",
		valueHandler:   JSONBlobHandler,
	}
	WakuBloomFilterMode = SettingField{
		reactFieldName: "waku-bloom-filter-mode",
		dBColumnName:   "waku_bloom_filter_mode",
		valueHandler:   BoolHandler,
	}
	WalletSetUpPassed = SettingField{
		reactFieldName: "wallet-set-up-passed?",
		dBColumnName:   "wallet_set_up_passed",
		valueHandler:   BoolHandler,
	}
	WalletVisibleTokens = SettingField{
		reactFieldName: "wallet/visible-tokens",
		dBColumnName:   "wallet_visible_tokens",
		valueHandler:   JSONBlobHandler,
	}
	WebviewAllowPermissionRequests = SettingField{
		reactFieldName: "webview-allow-permission-requests?",
		dBColumnName:   "webview_allow_permission_requests",
		valueHandler:   BoolHandler,
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
		GifFavourites,
		GifRecents,
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
