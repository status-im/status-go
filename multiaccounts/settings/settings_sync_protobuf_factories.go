package settings

import (
	"github.com/golang/protobuf/proto"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

func buildRawSyncSettingMessage(msg proto.Message, messageType protobuf.ApplicationMetadataMessage_Type, chatID string) (*common.RawMessage, error) {
	encodedMessage, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return &common.RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         messageType,
		ResendAutomatically: true,
	}, nil
}

func currencyProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingCurrency)
	pb.Value = value.(string)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_CURRENCY, chatID)
}

func gifRecentsProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingGifRecents)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_RECENTS, chatID)
}

func gifFavouritesProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingGifFavorites)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_FAVOURITES, chatID)
}

func messagesFromContactsOnlyProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingMessagesFromContactsOnly)
	pb.Value = value.(bool)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_MESSAGES_FROM_CONTACTS_ONLY, chatID)
}

func preferredNameProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingPreferredName)
	pb.Value = value.(string)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PREFERRED_NAME, chatID)
}

func previewPrivacyProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingPreviewPrivacy)
	pb.Value = value.(bool)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PREVIEW_PRIVACY, chatID)
}

func profilePicturesShowToProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingProfilePicturesShowTo)
	pb.Value = value.(int64)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_SHOW_TO, chatID)
}

func profilePicturesVisibilityProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingProfilePicturesVisibility)
	pb.Value = value.(int64)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_VISIBILITY, chatID)
}

func sendStatusUpdatesProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingSendStatusUpdates)
	pb.Value = value.(bool)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_FAVOURITES, chatID)
}

func stickersPacksInstalledProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingStickerPacksInstalled)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_PACKS_INSTALLED, chatID)
}

func stickersPacksPendingProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingStickerPacksPending)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_PACKS_PENDING, chatID)
}

func stickersRecentStickersProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingStickersRecentStickers)
	pb.Value = value.([]byte)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_RECENT_STICKERS, chatID)
}

func telemetryServerURLProtobufFactory(chatID string, value interface{}, clock uint64) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingTelemetryServerURL)
	pb.Value = value.(string)
	pb.Clock = clock

	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_TELEMETRY_SERVER_URL, chatID)
}
