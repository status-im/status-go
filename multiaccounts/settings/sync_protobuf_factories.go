package settings

import (
	"errors"

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
	v, ok := value.(ProfilePicturesShowToType)
	if !ok {

	}
	pb.Value = int64(v)
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_SHOW_TO
}

// profilePicturesVisibilityProtobufFactory
// TODO change the SyncSettingProtobufFactory signature to return an error if the data type can't match
//  something like `pb.Value, ok := value.(bool)`
func profilePicturesVisibilityProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type) {
	pb := new(protobuf.SyncSettingProfilePicturesVisibility)
	val, err := parseNumberToInt64(value)
	if err != nil {
		switch v := value.(type) {
		case ProfilePicturesVisibilityType:
			pb.Value = int64(v)
		default:
			// TODO throw error once SyncSettingProtobufFactory signature has changed
			pb.Value = int64(value.(int))
		}
	} else {
		pb.Value = val
	}

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

func parseNumberToInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	default:
		// TODO create Err const for type match not found
		return 0, errors.New("")
	}
}