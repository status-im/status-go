package settings

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"

	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

var (
	ErrTypeAssertionFailed = errors.New("type assertion of interface value failed")
)

func buildRawSyncSettingMessage(msg *protobuf.SyncSetting, chatID string) (*common.RawMessage, error) {
	encodedMessage, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	return &common.RawMessage{
		LocalChatID:         chatID,
		Payload:             encodedMessage,
		MessageType:         protobuf.ApplicationMetadataMessage_SYNC_SETTING,
		ResendAutomatically: true,
	}, nil
}

// Currency

func buildRawCurrencySyncMessage(v string, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_CURRENCY,
		Value: &protobuf.SyncSetting_ValueString{ValueString: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func currencyProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertString(value)
	if err != nil {
		return nil, err
	}

	return buildRawCurrencySyncMessage(v, clock, chatID)
}

func currencyProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	return buildRawCurrencySyncMessage(s.Currency, clock, chatID)
}

// GifFavorites

func buildRawGifFavoritesSyncMessage(v []byte, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_GIF_FAVOURITES,
		Value: &protobuf.SyncSetting_ValueBytes{ValueBytes: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func gifFavouritesProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertBytes(value)
	if err != nil {
		return nil, err
	}

	return buildRawGifFavoritesSyncMessage(v, clock, chatID)
}

func gifFavouritesProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	gf, _ := s.GifFavorites.MarshalJSON() // Don't need to parse error because it is always nil
	return buildRawGifFavoritesSyncMessage(gf, clock, chatID)
}

// GifFavorites

func buildRawGifRecentsSyncMessage(v []byte, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_GIF_RECENTS,
		Value: &protobuf.SyncSetting_ValueBytes{ValueBytes: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func gifRecentsProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertBytes(value)
	if err != nil {
		return nil, err
	}

	return buildRawGifRecentsSyncMessage(v, clock, chatID)
}

func gifRecentsProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	gr, _ := s.GifRecents.MarshalJSON() // Don't need to parse error because it is always nil
	return buildRawGifRecentsSyncMessage(gr, clock, chatID)
}

// MessagesFromContactsOnly

func buildRawMessagesFromContactsOnlySyncMessage(v bool, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_MESSAGES_FROM_CONTACTS_ONLY,
		Value: &protobuf.SyncSetting_ValueBool{ValueBool: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func messagesFromContactsOnlyProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertBool(value)
	if err != nil {
		return nil, err
	}

	return buildRawMessagesFromContactsOnlySyncMessage(v, clock, chatID)
}

func messagesFromContactsOnlyProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	return buildRawMessagesFromContactsOnlySyncMessage(s.MessagesFromContactsOnly, clock, chatID)
}

// PreferredName

func buildRawPreferredNameSyncMessage(v string, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_PREFERRED_NAME,
		Value: &protobuf.SyncSetting_ValueString{ValueString: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func preferredNameProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertString(value)
	if err != nil {
		return nil, err
	}

	return buildRawPreferredNameSyncMessage(v, clock, chatID)
}

func preferredNameProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	var pn string
	if s.PreferredName != nil {
		pn = *s.PreferredName
	}

	return buildRawPreferredNameSyncMessage(pn, clock, chatID)
}

// PreviewPrivacy

func buildRawPreviewPrivacySyncMessage(v bool, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_PREVIEW_PRIVACY,
		Value: &protobuf.SyncSetting_ValueBool{ValueBool: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func previewPrivacyProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertBool(value)
	if err != nil {
		return nil, err
	}

	return buildRawPreviewPrivacySyncMessage(v, clock, chatID)
}

func previewPrivacyProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	return buildRawPreviewPrivacySyncMessage(s.PreviewPrivacy, clock, chatID)
}

// ProfilePicturesShowTo

func buildRawProfilePicturesShowToSyncMessage(v int64, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_PROFILE_PICTURES_SHOW_TO,
		Value: &protobuf.SyncSetting_ValueInt64{ValueInt64: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func profilePicturesShowToProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := parseNumberToInt64(value)
	if err != nil {
		return nil, err
	}

	return buildRawProfilePicturesShowToSyncMessage(v, clock, chatID)
}

func profilePicturesShowToProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	return buildRawProfilePicturesShowToSyncMessage(int64(s.ProfilePicturesShowTo), clock, chatID)
}

// ProfilePicturesVisibility

func buildRawProfilePicturesVisibilitySyncMessage(v int64, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_PROFILE_PICTURES_VISIBILITY,
		Value: &protobuf.SyncSetting_ValueInt64{ValueInt64: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func profilePicturesVisibilityProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := parseNumberToInt64(value)
	if err != nil {
		return nil, err
	}

	return buildRawProfilePicturesVisibilitySyncMessage(v, clock, chatID)
}

func profilePicturesVisibilityProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	return buildRawProfilePicturesVisibilitySyncMessage(int64(s.ProfilePicturesVisibility), clock, chatID)
}

// SendStatusUpdates

func buildRawSendStatusUpdatesSyncMessage(v bool, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_SEND_STATUS_UPDATES,
		Value: &protobuf.SyncSetting_ValueBool{ValueBool: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func sendStatusUpdatesProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertBool(value)
	if err != nil {
		return nil, err
	}

	return buildRawSendStatusUpdatesSyncMessage(v, clock, chatID)
}

func sendStatusUpdatesProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	return buildRawSendStatusUpdatesSyncMessage(s.SendStatusUpdates, clock, chatID)
}

// StickerPacksInstalled

func buildRawStickerPacksInstalledSyncMessage(v []byte, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_STICKERS_PACKS_INSTALLED,
		Value: &protobuf.SyncSetting_ValueBytes{ValueBytes: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func stickersPacksInstalledProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertBytes(value)
	if err != nil {
		return nil, err
	}

	return buildRawStickerPacksInstalledSyncMessage(v, clock, chatID)
}

func stickersPacksInstalledProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	spi, _ := s.StickerPacksInstalled.MarshalJSON() // Don't need to parse error because it is always nil
	return buildRawStickerPacksInstalledSyncMessage(spi, clock, chatID)
}

// StickerPacksPending

func buildRawStickerPacksPendingSyncMessage(v []byte, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_STICKERS_PACKS_PENDING,
		Value: &protobuf.SyncSetting_ValueBytes{ValueBytes: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func stickersPacksPendingProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertBytes(value)
	if err != nil {
		return nil, err
	}

	return buildRawStickerPacksPendingSyncMessage(v, clock, chatID)
}

func stickersPacksPendingProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	spp, _ := s.StickerPacksPending.MarshalJSON() // Don't need to parse error because it is always nil
	return buildRawStickerPacksPendingSyncMessage(spp, clock, chatID)
}

// StickerPacksPending

func buildRawStickersRecentStickersSyncMessage(v []byte, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSetting {
		Type: protobuf.SyncSetting_STICKERS_RECENT_STICKERS,
		Value: &protobuf.SyncSetting_ValueBytes{ValueBytes: v},
		Clock: clock,
	}
	return buildRawSyncSettingMessage(pb, chatID)
}

func stickersRecentStickersProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	v, err := assertBytes(value)
	if err != nil {
		return nil, err
	}

	return buildRawStickersRecentStickersSyncMessage(v, clock, chatID)
}

func stickersRecentStickersProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	srs, _ := s.StickersRecentStickers.MarshalJSON() // Don't need to parse error because it is always nil
	return buildRawStickersRecentStickersSyncMessage(srs, clock, chatID)
}

func assertBytes(value interface{}) ([]byte, error) {
	v, ok := value.([]byte)
	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}
	return v, nil
}

func assertBool(value interface{}) (bool, error) {
	v, ok := value.(bool)
	if !ok {
		return false, errors.Wrapf(ErrTypeAssertionFailed, "expected 'bool', received %T", value)
	}
	return v, nil
}

func assertString(value interface{}) (string, error) {
	v, ok := value.(string)
	if !ok {
		return "", errors.Wrapf(ErrTypeAssertionFailed, "expected 'string', received %T", value)
	}
	return v, nil
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
	case ProfilePicturesShowToType:
		return int64(v), nil
	case ProfilePicturesVisibilityType:
		return int64(v), nil
	default:
		return 0, errors.Wrapf(ErrTypeAssertionFailed, "expected a numeric type, received %T", value)
	}
}
