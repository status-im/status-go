package protocol

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

func (m *Messenger) handleSyncSettings(messageState *ReceivedMessageState, settings protobuf.SyncSettings) error {

	return nil
}

func (m *Messenger) handleSyncSettingCurrency(messageState *ReceivedMessageState, settings protobuf.SyncSettingCurrency) error {
	var field = "currency"

	err := m.settings.SaveSetting(field, settings.Value)
	if err != nil {
		return err
	}

	return nil
}

func (m *Messenger) handleSyncSettingGifFavorites(messageState *ReceivedMessageState, settings protobuf.SyncSettingGifFavorites) error {

	return nil
}

func (m *Messenger) handleSyncSettingGifRecents(messageState *ReceivedMessageState, settings protobuf.SyncSettingGifRecents) error {

	return nil
}

func (m *Messenger) handleSyncSettingMessagesFromContactsOnly(messageState *ReceivedMessageState, settings protobuf.SyncSettingMessagesFromContactsOnly) error {

	return nil
}

func (m *Messenger) handleSyncSettingPreferredName(messageState *ReceivedMessageState, settings protobuf.SyncSettingPreferredName) error {

	return nil
}

func (m *Messenger) handleSyncSettingPreviewPrivacy(messageState *ReceivedMessageState, settings protobuf.SyncSettingPreviewPrivacy) error {

	return nil
}

func (m *Messenger) handleSyncSettingProfilePicturesShowTo(messageState *ReceivedMessageState, settings protobuf.SyncSettingProfilePicturesShowTo) error {

	return nil
}

func (m *Messenger) handleSyncSettingProfilePicturesVisibility(messageState *ReceivedMessageState, settings protobuf.SyncSettingProfilePicturesVisibility) error {

	return nil
}

func (m *Messenger) handleSyncSettingSendStatusUpdates(messageState *ReceivedMessageState, settings protobuf.SyncSettingSendStatusUpdates) error {

	return nil
}

func (m *Messenger) handleSyncSettingStickerPacksInstalled(messageState *ReceivedMessageState, settings protobuf.SyncSettingStickerPacksInstalled) error {

	return nil
}

func (m *Messenger) handleSyncSettingStickerPacksPending(messageState *ReceivedMessageState, settings protobuf.SyncSettingStickerPacksPending) error {

	return nil
}

func (m *Messenger) handleSyncSettingStickersRecentStickers(messageState *ReceivedMessageState, settings protobuf.SyncSettingStickersRecentStickers) error {

	return nil
}

func (m *Messenger) handleSyncSettingTelemetryServerURL(messageState *ReceivedMessageState, settings protobuf.SyncSettingTelemetryServerURL) error {

	return nil
}
