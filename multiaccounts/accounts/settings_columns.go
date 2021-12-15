package accounts

import (
	"encoding/json"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/sqlite"
)

type SettingsValueHandler func(interface{}) (interface{}, error)

type SettingField struct {
	ReactFieldName string
	DBColumnName string
	ValueHandler SettingsValueHandler
	ShouldSync   bool
}

var (
	AnonMetricsShouldSend = SettingField{
		"anon-metrics/should-send?",
		"anon_metrics_should_send",
		BoolHandler,
		false,
	}
	Appearance = SettingField{
		"appearance",
		"appearance",
		PassThroughHandler,
		false,
	}
	AutoMessageEnabled = SettingField{
		"auto-message-enabled?",
		"auto_message_enabled",
		BoolHandler,
		false,
	}
	BackupEnabled = SettingField{
		"backup-enabled?",
		"backup_enabled",
		BoolHandler,
		false,
	}
	ChaosMode = SettingField{
		"chaos-mode?",
		"chaos_mode",
		BoolHandler,
		false,
	}
	Currency = SettingField{
		"currency",
		"currency",
		PassThroughHandler,
		true,
	}
	CurrentUserStatus = SettingField{
		"current-user-status",
		"current_user_status",
		JSONBlobHandler,
		true,
	}
	CustomBootNodes = SettingField{
		ReactFieldName: "custom-bootnodes",
		DBColumnName:   "custom_bootnodes",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
	}
	CustomBootNodesEnabled = SettingField{
		ReactFieldName: "custom-bootnodes-enabled?",
		DBColumnName:   "custom_bootnodes_enabled",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
	}
	DappsAddress = SettingField{
		ReactFieldName: "dapps-address",
		DBColumnName:   "dapps_address",
		ValueHandler:   AddressHandler,
		ShouldSync:     false,
	}
	DefaultSyncPeriod = SettingField{
		ReactFieldName: "default-sync-period",
		DBColumnName:   "default_sync_period",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
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
	}
	Fleet = SettingField{
		ReactFieldName: "fleet",
		DBColumnName:   "fleet",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	GifAPIKey = SettingField{
		ReactFieldName: "gifs/api-key",
		DBColumnName:   "gif_api_key",
		ValueHandler:   BoolHandler,
		ShouldSync:     true,
	}
	GifRecents = SettingField{
		"gifs/recent-gifs",
		"gif_recents",
		JSONBlobHandler,
		true,
	}
	GifFavourites = SettingField{
		"gifs/favorite-gifs",
		"gif_favorites",
		JSONBlobHandler,
		true,
	}
	HideHomeTooltip = SettingField{
		ReactFieldName: "hide-home-tooltip?",
		DBColumnName:   "hide_home_tooltip",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	KeycardInstanceUID = SettingField{
		ReactFieldName: "keycard-instance_uid",
		DBColumnName:   "keycard_instance_uid",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	KeycardPairedOn = SettingField{
		ReactFieldName: "keycard-paired_on",
		DBColumnName:   "keycard_paired_on",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	KeycardPairing = SettingField{
		ReactFieldName: "keycard-pairing",
		DBColumnName:   "keycard_pairing",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	LastUpdated = SettingField{
		ReactFieldName: "last-updated",
		DBColumnName:   "last_updated",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	LatestDerivedPath = SettingField{
		ReactFieldName: "latest-derived-path",
		DBColumnName:   "latest_derived_path",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	LinkPreviewRequestEnabled = SettingField{
		ReactFieldName: "link-preview-request-enabled",
		DBColumnName:   "link_preview_request_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	LinkPreviewsEnabledSites = SettingField{
		ReactFieldName: "link-previews-enabled-sites",
		DBColumnName:   "link_previews_enabled_sites",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
	}
	LogLevel = SettingField{
		ReactFieldName: "log-level",
		DBColumnName:   "log_level",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	MessagesFromContactsOnly = SettingField{
		"messages-from-contacts-only",
		"messages_from_contacts_only",
		BoolHandler,
		true,
	}
	Mnemonic = SettingField{
		ReactFieldName: "mnemonic",
		DBColumnName:   "mnemonic",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	Name = SettingField{
		ReactFieldName: "name",
		DBColumnName:   "name",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	NetworksCurrentNetwork = SettingField{
		ReactFieldName: "networks/current-network",
		DBColumnName:   "current_network",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	NetworksNetworks = SettingField{
		ReactFieldName: "networks/networks",
		DBColumnName:   "networks",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
	}
	NodeConfig = SettingField{
		ReactFieldName: "node-config",
		DBColumnName:   "node_config",
		ValueHandler:   NodeConfigHandler,
		ShouldSync:     false,
	}
	NotificationsEnabled = SettingField{
		ReactFieldName: "notifications-enabled?",
		DBColumnName:   "notifications_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	OpenseaEnabled = SettingField{
		ReactFieldName: "opensea-enabled?",
		DBColumnName:   "opensea_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	PhotoPath = SettingField{
		ReactFieldName: "photo-path",
		DBColumnName:   "photo_path",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	PinnedMailservers = SettingField{
		ReactFieldName: "pinned-mailservers",
		DBColumnName:   "pinned_mailservers",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
	}
	PreferredName = SettingField{
		"preferred-name",
		"preferred_name",
		PassThroughHandler,
		true,
	}
	PreviewPrivacy = SettingField{
		"preview-privacy?",
		"preview_privacy",
		BoolHandler,
		true,
	}
	ProfilePicturesShowTo = SettingField{
		"profile-pictures-show-to",
		"profile_pictures_show_to",
		PassThroughHandler,
		true,
	}
	ProfilePicturesVisibility = SettingField{
		"profile-pictures-visibility",
		"profile_pictures_visibility",
		PassThroughHandler,
		true,
	}
	PublicKey = SettingField{
		ReactFieldName: "public-key",
		DBColumnName:   "public_key",
		ValueHandler:   PassThroughHandler,
		ShouldSync:     false,
	}
	PushNotificationsBlockMentions = SettingField{
		ReactFieldName: "push-notifications-block-mentions?",
		DBColumnName:   "push_notifications_block_mentions",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	PushNotificationsFromContactsOnly = SettingField{
		ReactFieldName: "push-notifications-from-contacts-only?",
		DBColumnName:   "push_notifications_from_contacts_only",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	PushNotificationsServerEnabled = SettingField{
		ReactFieldName: "push-notifications-server-enabled?",
		DBColumnName:   "push_notifications_server_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	RememberSyncingChoice = SettingField{
		ReactFieldName: "remember-syncing-choice?",
		DBColumnName:   "remember_syncing_choice",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	RemotePushNotificationsEnabled = SettingField{
		ReactFieldName: "remote-push-notifications-enabled?",
		DBColumnName:   "remote_push_notifications_enabled",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	SendPushNotifications = SettingField{
		ReactFieldName: "send-push-notifications?",
		DBColumnName:   "send_push_notifications",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	SendStatusUpdates = SettingField{
		"send-status-updates?",
		"send_status_updates",
		BoolHandler,
		true,
	}
	StickersPacksInstalled = SettingField{
		"stickers/packs-installed",
		"stickers_packs_installed",
		JSONBlobHandler,
		true,
	}
	StickersPacksPending = SettingField{
		"stickers/packs-pending",
		"stickers_packs_pending",
		JSONBlobHandler,
		true,
	}
	StickersRecentStickers = SettingField{
		"stickers/recent-stickers",
		"stickers_recent_stickers",
		JSONBlobHandler,
		true,
	}
	SyncingOnMobileNetwork = SettingField{
		ReactFieldName: "syncing-on-mobile-network?",
		DBColumnName:   "syncing_on_mobile_network",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	TelemetryServerURL = SettingField{
		"telemetry-server-url",
		"telemetry_server_url",
		PassThroughHandler,
		true,
	}
	UseMailservers = SettingField{
		ReactFieldName: "use-mailservers?",
		DBColumnName:   "use_mailservers",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	Usernames = SettingField{
		ReactFieldName: "usernames",
		DBColumnName:   "usernames",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
	}
	WakuBloomFilterMode = SettingField{
		ReactFieldName: "waku-bloom-filter-mode",
		DBColumnName:   "waku_bloom_filter_mode",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	WalletSetUpPassed = SettingField{
		ReactFieldName: "wallet-set-up-passed?",
		DBColumnName:   "wallet_set_up_passed",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
	}
	WalletVisibleTokens = SettingField{
		ReactFieldName: "wallet/visible-tokens",
		DBColumnName:   "wallet_visible_tokens",
		ValueHandler:   JSONBlobHandler,
		ShouldSync:     false,
	}
	WebviewAllowPermissionRequests = SettingField{
		ReactFieldName: "webview-allow-permission-requests?",
		DBColumnName:   "webview_allow_permission_requests",
		ValueHandler:   BoolHandler,
		ShouldSync:     false,
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

func PassThroughHandler(value interface{}) (interface{}, error) {
	return value, nil
}

func BoolHandler(value interface{}) (interface{}, error) {
	_, ok := value.(bool)
	if !ok {
		return value, ErrInvalidConfig
	}

	return value, nil
}

func JSONBlobHandler(value interface{}) (interface{}, error) {
	return &sqlite.JSONBlob{Data: value}, nil
}

func AddressHandler(value interface{}) (interface{}, error){
	str, ok := value.(string)
	if ok {
		value = types.HexToAddress(str)
	} else {
		return value, ErrInvalidConfig
	}
	return value, nil
}

func NodeConfigHandler(value interface{}) (interface{}, error){
	jsonString, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	var nodeConfig params.NodeConfig
	err = json.Unmarshal(jsonString, &nodeConfig)
	if err != nil {
		return nil, err
	}

	return nodeConfig, nil
}
