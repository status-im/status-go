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

// Currency

func buildRawCurrencySyncMessage(pb *protobuf.SyncSettingCurrency, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_CURRENCY, chatID)
}

func currencyProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingCurrency)
	v, ok := value.(string)
	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected 'string', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawCurrencySyncMessage(pb, chatID)
}

func currencyProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSettingCurrency{
		Clock: clock,
		Value: s.Currency,
	}
	return buildRawCurrencySyncMessage(pb, chatID)
}

// GifAPIKey

func buildRawGifAPIKeySyncMessage(pb *protobuf.SyncSettingGifAPIKey, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_API_KEY, chatID)
}

func gifAPIKeyProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingGifAPIKey)
	v, ok := value.(string)
	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected 'string', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawGifAPIKeySyncMessage(pb, chatID)
}

func gifAPIKeyProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSettingGifAPIKey{
		Clock: clock,
		Value: s.GifAPIKey,
	}
	return buildRawGifAPIKeySyncMessage(pb, chatID)
}

// GifFavorites

func buildRawGifFavoritesSyncMessage(pb *protobuf.SyncSettingGifFavorites, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_FAVOURITES, chatID)
}

func gifFavouritesProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingGifFavorites)
	v, ok := value.([]byte)
	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawGifFavoritesSyncMessage(pb, chatID)
}

func gifFavouritesProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	gf, _ := s.GifFavorites.MarshalJSON() // Don't need to parse error because it is always nil
	pb := &protobuf.SyncSettingGifFavorites{
		Clock: clock,
		Value: gf,
	}
	return buildRawGifFavoritesSyncMessage(pb, chatID)
}

// GifFavorites

func buildRawGifRecentsSyncMessage(pb *protobuf.SyncSettingGifRecents, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_GIF_RECENTS, chatID)
}

func gifRecentsProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingGifRecents)
	v, ok := value.([]byte)
	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawGifRecentsSyncMessage(pb, chatID)
}

func gifRecentsProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	gr, _ := s.GifRecents.MarshalJSON() // Don't need to parse error because it is always nil
	pb := &protobuf.SyncSettingGifRecents{
		Clock: clock,
		Value: gr,
	}
	return buildRawGifRecentsSyncMessage(pb, chatID)
}

// MessagesFromContactsOnly

func buildRawMessagesFromContactsOnlySyncMessage(pb *protobuf.SyncSettingMessagesFromContactsOnly, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_MESSAGES_FROM_CONTACTS_ONLY, chatID)
}

func messagesFromContactsOnlyProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingMessagesFromContactsOnly)
	v, ok := value.(bool)
	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected 'bool', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawMessagesFromContactsOnlySyncMessage(pb, chatID)
}

func messagesFromContactsOnlyProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSettingMessagesFromContactsOnly{
		Clock: clock,
		Value: s.MessagesFromContactsOnly,
	}
	return buildRawMessagesFromContactsOnlySyncMessage(pb, chatID)
}

// PreferredName

func buildRawPreferredNameSyncMessage(pb *protobuf.SyncSettingPreferredName, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PREFERRED_NAME, chatID)
}

func preferredNameProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingPreferredName)
	v, ok := value.(string)
	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected 'string', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawPreferredNameSyncMessage(pb, chatID)
}

func preferredNameProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	var pn string
	if s.PreferredName != nil {
		pn = *s.PreferredName
	}

	pb := &protobuf.SyncSettingPreferredName{
		Clock: clock,
		Value: pn,
	}

	return buildRawPreferredNameSyncMessage(pb, chatID)
}

// PreviewPrivacy

func buildRawPreviewPrivacySyncMessage(pb *protobuf.SyncSettingPreviewPrivacy, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PREVIEW_PRIVACY, chatID)
}

func previewPrivacyProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingPreviewPrivacy)
	v, ok := value.(bool)
	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawPreviewPrivacySyncMessage(pb, chatID)
}

func previewPrivacyProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSettingPreviewPrivacy{
		Clock: clock,
		Value: s.PreviewPrivacy,
	}
	return buildRawPreviewPrivacySyncMessage(pb, chatID)
}

// ProfilePicturesShowTo

func buildRawProfilePicturesShowToSyncMessage(pb *protobuf.SyncSettingProfilePicturesShowTo, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_SHOW_TO, chatID)
}

func profilePicturesShowToProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingProfilePicturesShowTo)
	val, err := parseNumberToInt64(value)
	if err != nil {
		switch v := value.(type) {
		case ProfilePicturesShowToType:
			pb.Value = int64(v)
		default:
			return nil, errors.Wrapf(err, "expected a numeric type, received %T", value)
		}
	} else {
		pb.Value = val
	}

	pb.Clock = clock

	return buildRawProfilePicturesShowToSyncMessage(pb, chatID)
}

func profilePicturesShowToProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSettingProfilePicturesShowTo{
		Clock: clock,
		Value: int64(s.ProfilePicturesShowTo),
	}
	return buildRawProfilePicturesShowToSyncMessage(pb, chatID)
}

// ProfilePicturesVisibility

func buildRawProfilePicturesVisibilitySyncMessage(pb *protobuf.SyncSettingProfilePicturesVisibility, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_PROFILE_PICTURES_VISIBILITY, chatID)
}

// profilePicturesVisibilityProtobufFactory
func profilePicturesVisibilityProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingProfilePicturesVisibility)
	val, err := parseNumberToInt64(value)
	if err != nil {
		switch v := value.(type) {
		case ProfilePicturesVisibilityType:
			pb.Value = int64(v)
		default:
			return nil, errors.Wrapf(err, "expected a numeric type, received %T", value)
		}
	} else {
		pb.Value = val
	}

	pb.Clock = clock

	return buildRawProfilePicturesVisibilitySyncMessage(pb, chatID)
}

func profilePicturesVisibilityProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSettingProfilePicturesVisibility{
		Clock: clock,
		Value: int64(s.ProfilePicturesVisibility),
	}
	return buildRawProfilePicturesVisibilitySyncMessage(pb, chatID)
}

// SendStatusUpdates

func buildRawSendStatusUpdatesSyncMessage(pb *protobuf.SyncSettingSendStatusUpdates, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_SEND_STATUS_UPDATES, chatID)
}

func sendStatusUpdatesProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingSendStatusUpdates)
	v, ok := value.(bool)

	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected 'bool', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawSendStatusUpdatesSyncMessage(pb, chatID)
}

func sendStatusUpdatesProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSettingSendStatusUpdates{
		Clock: clock,
		Value: s.SendStatusUpdates,
	}
	return buildRawSendStatusUpdatesSyncMessage(pb, chatID)
}

// StickerPacksInstalled

func buildRawStickerPacksInstalledSyncMessage(pb *protobuf.SyncSettingStickerPacksInstalled, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_PACKS_INSTALLED, chatID)
}

func stickersPacksInstalledProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingStickerPacksInstalled)
	v, ok := value.([]byte)

	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawStickerPacksInstalledSyncMessage(pb, chatID)
}

func stickersPacksInstalledProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	spi, _ := s.StickerPacksInstalled.MarshalJSON() // Don't need to parse error because it is always nil
	pb := &protobuf.SyncSettingStickerPacksInstalled{
		Clock: clock,
		Value: spi,
	}
	return buildRawStickerPacksInstalledSyncMessage(pb, chatID)
}

// StickerPacksPending

func buildRawStickerPacksPendingSyncMessage(pb *protobuf.SyncSettingStickerPacksPending, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_PACKS_PENDING, chatID)
}

func stickersPacksPendingProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingStickerPacksPending)
	v, ok := value.([]byte)

	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawStickerPacksPendingSyncMessage(pb, chatID)
}

func stickersPacksPendingProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	spp, _ := s.StickerPacksPending.MarshalJSON() // Don't need to parse error because it is always nil
	pb := &protobuf.SyncSettingStickerPacksPending{
		Clock: clock,
		Value: spp,
	}
	return buildRawStickerPacksPendingSyncMessage(pb, chatID)
}

// StickerPacksPending

func buildRawStickersRecentStickersSyncMessage(pb *protobuf.SyncSettingStickersRecentStickers, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_STICKERS_RECENT_STICKERS, chatID)
}

func stickersRecentStickersProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingStickersRecentStickers)
	v, ok := value.([]byte)

	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected '[]byte', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawStickersRecentStickersSyncMessage(pb, chatID)
}

func stickersRecentStickersProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	srs, _ := s.StickersRecentStickers.MarshalJSON() // Don't need to parse error because it is always nil
	pb := &protobuf.SyncSettingStickersRecentStickers{
		Clock: clock,
		Value: srs,
	}
	return buildRawStickersRecentStickersSyncMessage(pb, chatID)
}

// TelemetryServerURL

func buildRawTelemetryServerURLSyncMessage(pb *protobuf.SyncSettingTelemetryServerURL, chatID string) (*common.RawMessage, error) {
	return buildRawSyncSettingMessage(pb, protobuf.ApplicationMetadataMessage_SYNC_SETTING_TELEMETRY_SERVER_URL, chatID)
}

func telemetryServerURLProtobufFactory(value interface{}, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := new(protobuf.SyncSettingTelemetryServerURL)
	v, ok := value.(string)

	if !ok {
		return nil, errors.Wrapf(ErrTypeAssertionFailed, "expected 'string', received %T", value)
	}

	pb.Value = v
	pb.Clock = clock

	return buildRawTelemetryServerURLSyncMessage(pb, chatID)
}

func telemetryServerURLProtobufFactoryStruct(s Settings, clock uint64, chatID string) (*common.RawMessage, error) {
	pb := &protobuf.SyncSettingTelemetryServerURL{
		Clock: clock,
		Value: s.TelemetryServerURL,
	}
	return buildRawTelemetryServerURLSyncMessage(pb, chatID)
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
