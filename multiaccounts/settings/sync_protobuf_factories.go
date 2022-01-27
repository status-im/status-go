package settings

import (
	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/protobuf"
)

func currencyProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingCurrency)
	pb.Value = value.(string)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_CURRENCY
}

func gifAPIKeyProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingGifAPIKey)
	pb.Value = value.(string)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_API_KEY
}

func gifFavouritesProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingGifFavorites)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_FAVOURITES
}

func gifRecentsProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingGifRecents)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_RECENTS
}

func messagesFromContactsOnlyProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingMessagesFromContactsOnly)
	pb.Value = value.(bool)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_MESSAGES_FROM_CONTACTS_ONLY
}

func preferredNameProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingPreferredName)
	pb.Value = value.(string)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PREFERRED_NAME
}

func previewPrivacyProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingPreviewPrivacy)
	pb.Value = value.(bool)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PREVIEW_PRIVACY
}

func profilePicturesShowToProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingProfilePicturesShowTo)
	pb.Value = int64(value.(ProfilePicturesShowToType))
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_SHOW_TO
}

func profilePicturesVisibilityProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingProfilePicturesVisibility)
	pb.Value = int64(value.(ProfilePicturesVisibilityType))
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_VISIBILITY
}

func sendStatusUpdatesProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingSendStatusUpdates)
	pb.Value = value.(bool)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_FAVOURITES
}

func stickersPacksInstalledProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingStickerPacksInstalled)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_PACKS_INSTALLED
}

func stickersPacksPendingProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingStickerPacksPending)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_PACKS_PENDING
}

func stickersRecentStickersProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingStickersRecentStickers)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_RECENT_STICKERS
}

func telemetryServerURLProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingTelemetryServerURL)
	pb.Value = value.(string)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_TELEMETRY_SERVER_URL
}
