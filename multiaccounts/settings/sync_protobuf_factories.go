package settings

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	ErrTypeAssertionFailed = errors.New("type assertion of interface value failed")
)

func currencyProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingCurrency)
	v, ok := value.(string)
	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected 'string', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_CURRENCY, nil
}

func gifAPIKeyProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingGifAPIKey)
	v, ok := value.(string)
	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected 'string', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_API_KEY, nil
}

func gifFavouritesProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingGifFavorites)
	v, ok := value.([]byte)
	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_FAVOURITES, nil
}

func gifRecentsProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingGifRecents)
	v, ok := value.([]byte)
	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_RECENTS, nil
}

func messagesFromContactsOnlyProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingMessagesFromContactsOnly)
	v, ok := value.(bool)
	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected 'bool', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_MESSAGES_FROM_CONTACTS_ONLY, nil
}

func preferredNameProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingPreferredName)
	v, ok := value.(string)
	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected 'string', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PREFERRED_NAME, nil
}

func previewPrivacyProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingPreviewPrivacy)
	v, ok := value.(bool)
	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PREVIEW_PRIVACY, nil
}

func profilePicturesShowToProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingProfilePicturesShowTo)
	val, err := parseNumberToInt64(value)
	if err != nil {
		switch v := value.(type) {
		case ProfilePicturesShowToType:
			pb.Value = int64(v)
		default:
			return nil, 0, errors.Wrapf(err, "expected a numeric type, received %T", value)
		}
	} else {
		pb.Value = val
	}

	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_SHOW_TO, nil
}

// profilePicturesVisibilityProtobufFactory
func profilePicturesVisibilityProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingProfilePicturesVisibility)
	val, err := parseNumberToInt64(value)
	if err != nil {
		switch v := value.(type) {
		case ProfilePicturesVisibilityType:
			pb.Value = int64(v)
		default:
			return nil, 0, errors.Wrapf(err, "expected a numeric type, received %T", value)
		}
	} else {
		pb.Value = val
	}

	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_VISIBILITY, nil
}

func sendStatusUpdatesProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingSendStatusUpdates)
	v, ok := value.(bool)

	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected 'bool', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_FAVOURITES, nil
}

func stickersPacksInstalledProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingStickerPacksInstalled)
	v, ok := value.([]byte)

	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_PACKS_INSTALLED, nil
}

func stickersPacksPendingProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingStickerPacksPending)
	v, ok := value.([]byte)

	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_PACKS_PENDING, nil
}

func stickersRecentStickersProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingStickersRecentStickers)
	v, ok := value.([]byte)

	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_RECENT_STICKERS, nil
}

func telemetryServerURLProtobufFactory(value interface{}, clock uint64) (proto.Message, protobuf.ApplicationMetadataMessage_Type, error) {
	pb := new(protobuf.SyncSettingTelemetryServerURL)
	v, ok := value.(string)

	if !ok {
		return nil, 0, errors.Wrapf(ErrTypeAssertionFailed, "expected 'string', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_TELEMETRY_SERVER_URL, nil
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
		return 0, ErrTypeAssertionFailed
	}
}