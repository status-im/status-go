package settings

import (
	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	AnonMetricsShouldSend = SettingField{
		reactFieldName: "anon-metrics/should-send?",
		dBColumnName:   "anon_metrics_should_send",
		fieldName:      "AnonMetricsShouldSend",
		valueHandler:   BoolHandler,
	}
	Appearance = SettingField{
		reactFieldName: "appearance",
		dBColumnName:   "appearance",
		fieldName:      "Appearance",
	}
	AutoMessageEnabled = SettingField{
		reactFieldName:   "auto-message-enabled?",
		dBColumnName:     "auto_message_enabled",
		fieldName:        "AutoMessageEnabled",
		valueHandler:     BoolHandler,
		pairingBootstrap: true,
	}
	BackupEnabled = SettingField{
		reactFieldName:   "backup-enabled?",
		dBColumnName:     "backup_enabled",
		fieldName:        "BackupEnabled",
		valueHandler:     BoolHandler,
		pairingBootstrap: true,
	}
	BackupFetched = SettingField{
		reactFieldName: "backup-fetched?",
		dBColumnName:   "backup_fetched",
		fieldName:      "BackupFetched",
		valueHandler:   BoolHandler,
		// pairingBootstrap: true,
		// TODO DOES NOT APPEAR IN STRUCT
	}
	Bio = SettingField{
		reactFieldName:   "bio",
		dBColumnName:     "bio",
		fieldName:        "Bio",
		pairingBootstrap: true,
	}
	ChaosMode = SettingField{
		reactFieldName: "chaos-mode?",
		dBColumnName:   "chaos_mode",
		fieldName:      "ChaosMode",
		valueHandler:   BoolHandler,
	}
	Currency = SettingField{
		reactFieldName: "currency",
		dBColumnName:   "currency",
		fieldName:      "Currency",
		syncProtobufFactory: &SyncProtobufFactory{
			fromInterface:     currencyProtobufFactory,
			fromStruct:        currencyProtobufFactoryStruct,
			valueFromProtobuf: StringFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_CURRENCY,
		},
		pairingBootstrap: true,
	}
	CurrentUserStatus = SettingField{
		reactFieldName: "current-user-status",
		dBColumnName:   "current_user_status",
		fieldName:      "CurrentUserStatus",
		valueHandler:   JSONBlobHandler,
	}
	CustomBootNodes = SettingField{
		reactFieldName:   "custom-bootnodes",
		dBColumnName:     "custom_bootnodes",
		fieldName:        "CustomBootnodes",
		valueHandler:     JSONBlobHandler,
		pairingBootstrap: true,
	}
	CustomBootNodesEnabled = SettingField{
		reactFieldName:   "custom-bootnodes-enabled?",
		dBColumnName:     "custom_bootnodes_enabled",
		fieldName:        "CustomBootnodesEnabled",
		valueHandler:     JSONBlobHandler,
		pairingBootstrap: true,
	}
	DappsAddress = SettingField{
		reactFieldName:   "dapps-address",
		dBColumnName:     "dapps_address",
		fieldName:        "DappsAddress",
		valueHandler:     AddressHandler,
		pairingBootstrap: true,
	}
	DefaultSyncPeriod = SettingField{
		reactFieldName: "default-sync-period",
		dBColumnName:   "default_sync_period",
		fieldName:      "DefaultSyncPeriod",
	}
	DisplayName = SettingField{
		reactFieldName:   "display-name",
		dBColumnName:     "display_name",
		fieldName:        "DisplayName",
		pairingBootstrap: true,
	}
	EIP1581Address = SettingField{
		reactFieldName:   "eip1581-address",
		dBColumnName:     "eip1581_address",
		fieldName:        "EIP1581Address",
		valueHandler:     AddressHandler,
		pairingBootstrap: true,
	}
	Fleet = SettingField{
		reactFieldName: "fleet",
		dBColumnName:   "fleet",
		fieldName:      "Fleet",
	}
	GifAPIKey = SettingField{
		reactFieldName: "gifs/api-key",
		dBColumnName:   "gif_api_key",
		fieldName:      "GifAPIKey",
	}
	GifFavourites = SettingField{
		reactFieldName: "gifs/favorite-gifs",
		dBColumnName:   "gif_favorites",
		fieldName:      "GifFavorites",
		valueHandler:   JSONBlobHandler,
		// TODO resolve issue 8 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1065179963
		//  The reported issue is not directly related, but I suspect that gifs suffer the same issue
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     gifFavouritesProtobufFactory,
			fromStruct:        gifFavouritesProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_GIF_FAVOURITES,
		},
		pairingBootstrap: true,
	}
	GifRecents = SettingField{
		reactFieldName: "gifs/recent-gifs",
		dBColumnName:   "gif_recents",
		fieldName:      "GifRecents",
		valueHandler:   JSONBlobHandler,
		// TODO resolve issue 8 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1065179963
		//  The reported issue is not directly related, but I suspect that gifs suffer the same issue
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     gifRecentsProtobufFactory,
			fromStruct:        gifRecentsProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_GIF_RECENTS,
		},
		pairingBootstrap: true,
	}
	HideHomeTooltip = SettingField{
		reactFieldName: "hide-home-tooltip?",
		dBColumnName:   "hide_home_tooltip",
		fieldName:      "HideHomeTooltip",
		valueHandler:   BoolHandler,
	}
	InstallationID = SettingField{
		reactFieldName: "installation-id",
		dBColumnName:   "installation_id",
		fieldName:      "InstallationID",
	}
	KeycardInstanceUID = SettingField{
		reactFieldName: "keycard-instance_uid",
		dBColumnName:   "keycard_instance_uid",
		fieldName:      "KeycardInstanceUID",
	}
	KeycardPairedOn = SettingField{
		reactFieldName: "keycard-paired_on",
		dBColumnName:   "keycard_paired_on",
		fieldName:      "KeycardPairedOn",
	}
	KeycardPairing = SettingField{
		reactFieldName: "keycard-pairing",
		dBColumnName:   "keycard_pairing",
		fieldName:      "KeycardPairing",
	}
	LastBackup = SettingField{
		reactFieldName: "last-backup",
		dBColumnName:   "last_backup",
		fieldName:      "LastBackup",
	}
	LastUpdated = SettingField{
		reactFieldName: "last-updated",
		dBColumnName:   "last_updated",
		fieldName:      "LastUpdated",
	}
	LatestDerivedPath = SettingField{
		reactFieldName:   "latest-derived-path",
		dBColumnName:     "latest_derived_path",
		fieldName:        "LatestDerivedPath",
		pairingBootstrap: true,
	}
	LinkPreviewRequestEnabled = SettingField{
		reactFieldName:   "link-preview-request-enabled",
		dBColumnName:     "link_preview_request_enabled",
		fieldName:        "LinkPreviewRequestEnabled",
		valueHandler:     BoolHandler,
		pairingBootstrap: true,
	}
	LinkPreviewsEnabledSites = SettingField{
		reactFieldName:   "link-previews-enabled-sites",
		dBColumnName:     "link_previews_enabled_sites",
		fieldName:        "LinkPreviewsEnabledSites",
		valueHandler:     JSONBlobHandler,
		pairingBootstrap: true,
	}
	LogLevel = SettingField{
		reactFieldName: "log-level",
		dBColumnName:   "log_level",
		fieldName:      "LogLevel",
	}
	MessagesFromContactsOnly = SettingField{
		reactFieldName: "messages-from-contacts-only",
		dBColumnName:   "messages_from_contacts_only",
		fieldName:      "MessagesFromContactsOnly",
		valueHandler:   BoolHandler,
		syncProtobufFactory: &SyncProtobufFactory{
			fromInterface:     messagesFromContactsOnlyProtobufFactory,
			fromStruct:        messagesFromContactsOnlyProtobufFactoryStruct,
			valueFromProtobuf: BoolFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_MESSAGES_FROM_CONTACTS_ONLY,
		},
		pairingBootstrap: true,
	}
	Mnemonic = SettingField{
		reactFieldName:   "mnemonic",
		dBColumnName:     "mnemonic",
		fieldName:        "Mnemonic",
		pairingBootstrap: true,
	}
	MutualContactEnabled = SettingField{
		reactFieldName:   "mutual-contact-enabled?",
		dBColumnName:     "mutual_contact_enabled",
		fieldName:        "MutualContactEnabled",
		valueHandler:     BoolHandler,
		pairingBootstrap: true,
	}
	Name = SettingField{
		reactFieldName:   "name",
		dBColumnName:     "name",
		fieldName:        "Name",
		pairingBootstrap: true,
	}
	NetworksCurrentNetwork = SettingField{
		reactFieldName:   "networks/current-network",
		dBColumnName:     "current_network",
		fieldName:        "CurrentNetwork",
		pairingBootstrap: true,
	}
	NetworksNetworks = SettingField{
		reactFieldName:   "networks/networks",
		dBColumnName:     "networks",
		fieldName:        "Networks",
		valueHandler:     JSONBlobHandler,
		pairingBootstrap: true,
	}
	NodeConfig = SettingField{
		reactFieldName: "node-config",
		dBColumnName:   "node_config",
		fieldName:      "NodeConfig",
		valueHandler:   NodeConfigHandler,
		// TODO requires special handling because node config is no longer a setting
	}
	// NotificationsEnabled - we should remove this and realated things once mobile team starts usign `settings_notifications` package
	NotificationsEnabled = SettingField{
		reactFieldName: "notifications-enabled?",
		dBColumnName:   "notifications_enabled",
		fieldName:      "NotificationsEnabled",
		valueHandler:   BoolHandler,
	}
	OpenseaEnabled = SettingField{
		reactFieldName:   "opensea-enabled?",
		dBColumnName:     "opensea_enabled",
		fieldName:        "OpenseaEnabled",
		valueHandler:     BoolHandler,
		pairingBootstrap: true,
	}
	PhotoPath = SettingField{
		// TODO needs to be removed from code ASAP, because it just confuses everyone
		reactFieldName: "photo-path",
		dBColumnName:   "photo_path",
		fieldName:      "PhotoPath",
	}
	PinnedMailservers = SettingField{
		reactFieldName:   "pinned-mailservers",
		dBColumnName:     "pinned_mailservers",
		fieldName:        "PinnedMailserver",
		valueHandler:     JSONBlobHandler,
		pairingBootstrap: true,
	}
	PreferredName = SettingField{
		reactFieldName: "preferred-name",
		dBColumnName:   "preferred_name",
		fieldName:      "PreferredName",
		// TODO resolve issue 9 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1075336559
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     preferredNameProtobufFactory,
			fromStruct:        preferredNameProtobufFactoryStruct,
			valueFromProtobuf: StringFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_PREFERRED_NAME,
		},
		pairingBootstrap: true,
	}
	PreviewPrivacy = SettingField{
		reactFieldName: "preview-privacy?",
		dBColumnName:   "preview_privacy",
		fieldName:      "PreviewPrivacy",
		valueHandler:   BoolHandler,
		// TODO resolved issue 7 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1065179963
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     previewPrivacyProtobufFactory,
			fromStruct:        previewPrivacyProtobufFactoryStruct,
			valueFromProtobuf: BoolFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_PREVIEW_PRIVACY,
		},
	}
	ProfilePicturesShowTo = SettingField{
		reactFieldName: "profile-pictures-show-to",
		dBColumnName:   "profile_pictures_show_to",
		fieldName:      "ProfilePicturesShowTo",
		syncProtobufFactory: &SyncProtobufFactory{
			fromInterface:     profilePicturesShowToProtobufFactory,
			fromStruct:        profilePicturesShowToProtobufFactoryStruct,
			valueFromProtobuf: Int64FromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_PROFILE_PICTURES_SHOW_TO,
		},
		pairingBootstrap: true,
	}
	ProfilePicturesVisibility = SettingField{
		reactFieldName: "profile-pictures-visibility",
		dBColumnName:   "profile_pictures_visibility",
		fieldName:      "ProfilePicturesVisibility",
		syncProtobufFactory: &SyncProtobufFactory{
			fromInterface:     profilePicturesVisibilityProtobufFactory,
			fromStruct:        profilePicturesVisibilityProtobufFactoryStruct,
			valueFromProtobuf: Int64FromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_PROFILE_PICTURES_VISIBILITY,
		},
		pairingBootstrap: true,
	}
	PublicKey = SettingField{
		reactFieldName:   "public-key",
		dBColumnName:     "public_key",
		fieldName:        "PublicKey",
		pairingBootstrap: true,
	}
	PushNotificationsBlockMentions = SettingField{
		reactFieldName: "push-notifications-block-mentions?",
		dBColumnName:   "push_notifications_block_mentions",
		fieldName:      "PushNotificationsBlockMentions",
		valueHandler:   BoolHandler,
	}
	PushNotificationsFromContactsOnly = SettingField{
		reactFieldName: "push-notifications-from-contacts-only?",
		dBColumnName:   "push_notifications_from_contacts_only",
		fieldName:      "PushNotificationsFromContactsOnly",
		valueHandler:   BoolHandler,
	}
	PushNotificationsServerEnabled = SettingField{
		reactFieldName: "push-notifications-server-enabled?",
		dBColumnName:   "push_notifications_server_enabled",
		fieldName:      "PushNotificationsServerEnabled",
		valueHandler:   BoolHandler,
	}
	RememberSyncingChoice = SettingField{
		reactFieldName: "remember-syncing-choice?",
		dBColumnName:   "remember_syncing_choice",
		fieldName:      "RememberSyncingChoice",
		valueHandler:   BoolHandler,
	}
	RemotePushNotificationsEnabled = SettingField{
		reactFieldName: "remote-push-notifications-enabled?",
		dBColumnName:   "remote_push_notifications_enabled",
		fieldName:      "RemotePushNotificationsEnabled",
		valueHandler:   BoolHandler,
	}
	SendPushNotifications = SettingField{
		reactFieldName: "send-push-notifications?",
		dBColumnName:   "send_push_notifications",
		fieldName:      "SendPushNotifications",
		valueHandler:   BoolHandler,
	}
	SendStatusUpdates = SettingField{
		reactFieldName: "send-status-updates?",
		dBColumnName:   "send_status_updates",
		fieldName:      "SendStatusUpdates",
		valueHandler:   BoolHandler,
		// TODO resolve issue 10 https://github.com/status-im/status-mobile/pull/13053#issuecomment-1075352256
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // Remove after issue is resolved
			fromInterface:     sendStatusUpdatesProtobufFactory,
			fromStruct:        sendStatusUpdatesProtobufFactoryStruct,
			valueFromProtobuf: BoolFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_SEND_STATUS_UPDATES,
		},
	}
	StickersPacksInstalled = SettingField{
		reactFieldName: "stickers/packs-installed",
		dBColumnName:   "stickers_packs_installed",
		fieldName:      "StickerPacksInstalled",
		valueHandler:   JSONBlobHandler,
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // TODO current version of stickers introduces a regression on deleting sticker packs
			fromInterface:     stickersPacksInstalledProtobufFactory,
			fromStruct:        stickersPacksInstalledProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_STICKERS_PACKS_INSTALLED,
		},
		pairingBootstrap: true,
	}
	StickersPacksPending = SettingField{
		reactFieldName: "stickers/packs-pending",
		dBColumnName:   "stickers_packs_pending",
		fieldName:      "StickerPacksPending",
		valueHandler:   JSONBlobHandler,
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // TODO current version of stickers introduces a regression on deleting sticker packs
			fromInterface:     stickersPacksPendingProtobufFactory,
			fromStruct:        stickersPacksPendingProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_STICKERS_PACKS_PENDING,
		},
		pairingBootstrap: true,
	}
	StickersRecentStickers = SettingField{
		reactFieldName: "stickers/recent-stickers",
		dBColumnName:   "stickers_recent_stickers",
		fieldName:      "StickersRecentStickers",
		valueHandler:   JSONBlobHandler,
		syncProtobufFactory: &SyncProtobufFactory{
			inactive:          true, // TODO current version of stickers introduces a regression on deleting sticker packs
			fromInterface:     stickersRecentStickersProtobufFactory,
			fromStruct:        stickersRecentStickersProtobufFactoryStruct,
			valueFromProtobuf: BytesFromSyncProtobuf,
			protobufType:      protobuf.SyncSetting_STICKERS_RECENT_STICKERS,
		},
		pairingBootstrap: true,
	}
	SyncingOnMobileNetwork = SettingField{
		reactFieldName: "syncing-on-mobile-network?",
		dBColumnName:   "syncing_on_mobile_network",
		fieldName:      "SyncingOnMobileNetwork",
		valueHandler:   BoolHandler,
	}
	TelemetryServerURL = SettingField{
		reactFieldName: "telemetry-server-url",
		dBColumnName:   "telemetry_server_url",
		fieldName:      "TelemetryServerURL",
	}
	TestNetworksEnabled = SettingField{
		reactFieldName: "test-networks-enabled?",
		dBColumnName:   "test_networks_enabled",
		fieldName:      "TestNetworksEnabled",
		valueHandler:   BoolHandler,
	}
	UseMailservers = SettingField{
		reactFieldName:   "use-mailservers?",
		dBColumnName:     "use_mailservers",
		fieldName:        "UseMailservers",
		valueHandler:     BoolHandler,
		pairingBootstrap: true,
	}
	Usernames = SettingField{
		reactFieldName:   "usernames",
		dBColumnName:     "usernames",
		fieldName:        "Usernames",
		valueHandler:     JSONBlobHandler,
		pairingBootstrap: true,
	}
	WakuBloomFilterMode = SettingField{
		reactFieldName: "waku-bloom-filter-mode",
		dBColumnName:   "waku_bloom_filter_mode",
		fieldName:      "WakuBloomFilterMode",
		valueHandler:   BoolHandler,
	}
	WalletSetUpPassed = SettingField{
		reactFieldName:   "wallet-set-up-passed?",
		dBColumnName:     "wallet_set_up_passed",
		fieldName:        "WalletSetUpPassed",
		valueHandler:     BoolHandler,
		pairingBootstrap: true,
	}
	WalletVisibleTokens = SettingField{
		reactFieldName:   "wallet/visible-tokens",
		dBColumnName:     "wallet_visible_tokens",
		fieldName:        "WalletVisibleTokens",
		valueHandler:     JSONBlobHandler,
		pairingBootstrap: true,
	}
	WebviewAllowPermissionRequests = SettingField{
		reactFieldName: "webview-allow-permission-requests?",
		dBColumnName:   "webview_allow_permission_requests",
		fieldName:      "WebviewAllowPermissionRequests",
		valueHandler:   BoolHandler,
	}
	WalletRootAddress = SettingField{
		reactFieldName:   "wallet-root-address",
		dBColumnName:     "wallet_root_address",
		fieldName:        "WalletRootAddress",
		valueHandler:     AddressHandler,
		pairingBootstrap: true,
	}

	SettingFieldRegister = []SettingField{
		AnonMetricsShouldSend,
		Appearance,
		AutoMessageEnabled,
		BackupEnabled,
		BackupFetched,
		Bio,
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
		InstallationID,
		KeycardInstanceUID,
		KeycardPairedOn,
		KeycardPairing,
		LastBackup,
		LastUpdated,
		LatestDerivedPath,
		LinkPreviewRequestEnabled,
		LinkPreviewsEnabledSites,
		LogLevel,
		MessagesFromContactsOnly,
		Mnemonic,
		MutualContactEnabled,
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
		TestNetworksEnabled,
		UseMailservers,
		Usernames,
		WakuBloomFilterMode,
		WalletRootAddress,
		WalletSetUpPassed,
		WalletVisibleTokens,
		WebviewAllowPermissionRequests,
	}
)

func GetFieldFromProtobufType(pbt protobuf.SyncSetting_Type) (SettingField, error) {
	if pbt == protobuf.SyncSetting_UNKNOWN {
		return SettingField{}, errors.ErrUnrecognisedSyncSettingProtobufType
	}

	for _, s := range SettingFieldRegister {
		if s.SyncProtobufFactory() == nil {
			continue
		}
		if s.SyncProtobufFactory().SyncSettingProtobufType() == pbt {
			return s, nil
		}
	}

	return SettingField{}, errors.ErrUnrecognisedSyncSettingProtobufType
}

func GetFieldFromFieldName(fn string) (SettingField, error) {
	for _, s := range SettingFieldRegister {
		if s.fieldName == fn {
			return s, nil
		}
	}

	return SettingField{}, errors.ErrUnrecognisedSettingFieldName
}
