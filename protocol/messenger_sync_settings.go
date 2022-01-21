package protocol

import (
	"context"

	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"

	"github.com/status-im/status-go/multiaccounts/settings"
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

// syncSettings syncs all settings that are syncable
func (m *Messenger) syncSettings() error {
	logger := m.logger.Named("syncSettings")

	s, err := m.settings.GetSettings()
	if err != nil {
		return err
	}

	clock, chat := m.getLastClockWithRelatedChat()

	// TODO implement value handler from Settings field type to protobuf type
	//  we need to make sure that the data format is always the same in the protobuf!

	gr, err := s.GifRecents.MarshalJSON()
	if err != nil {
		return err
	}
	gf, err := s.GifFavorites.MarshalJSON()
	if err != nil {
		return err
	}
	spi, err := s.StickerPacksInstalled.MarshalJSON()
	if err != nil {
		return err
	}
	spp, err := s.StickerPacksPending.MarshalJSON()
	if err != nil {
		return err
	}
	srs, err := s.StickersRecentStickers.MarshalJSON()
	if err != nil {
		return err
	}

	ps := &protobuf.SyncSettings{
		Currency: &protobuf.SyncSettingCurrency{
			Value: s.Currency,
			Clock: clock,
		},
		GifFavorites: &protobuf.SyncSettingGifFavorites{
			Value: gf,
			Clock: clock,
		},
		GifRecents: &protobuf.SyncSettingGifRecents{
			Value: gr,
			Clock: clock,
		},
		MessagesFromContactsOnly: &protobuf.SyncSettingMessagesFromContactsOnly{
			Value: s.MessagesFromContactsOnly,
			Clock: clock,
		},
		PreferredName: &protobuf.SyncSettingPreferredName{
			Value: *s.PreferredName,
			Clock: clock,
		},
		PreviewPrivacy: &protobuf.SyncSettingPreviewPrivacy{
			Value: s.PreviewPrivacy,
			Clock: clock,
		},
		ProfilePicturesShowTo: &protobuf.SyncSettingProfilePicturesShowTo{
			Value: int64(s.ProfilePicturesShowTo),
			Clock: clock,
		},
		ProfilePicturesVisibility: &protobuf.SyncSettingProfilePicturesVisibility{
			Value: int64(s.ProfilePicturesVisibility),
			Clock: clock,
		},
		SendStatusUpdates: &protobuf.SyncSettingSendStatusUpdates{
			Value: s.SendStatusUpdates,
			Clock: clock,
		},
		StickerPacksInstalled: &protobuf.SyncSettingStickerPacksInstalled{
			Value: spi,
			Clock: clock,
		},
		StickerPacksPending: &protobuf.SyncSettingStickerPacksPending{
			Value: spp,
			Clock: clock,
		},
		StickersRecentStickers: &protobuf.SyncSettingStickersRecentStickers{
			Value: srs,
			Clock: clock,
		},
		TelemetryServer_URL: &protobuf.SyncSettingTelemetryServerURL{
			Value: s.TelemetryServerURL,
			Clock: clock,
		},
	}

	rm, err := buildRawSyncSettingMessage(ps, protobuf.ApplicationMetadataMessage_SYNC_SETTINGS, chat.ID)
	if err != nil {
		return err
	}

	_, err = m.dispatchMessage(context.Background(), *rm)
	if err != nil {
		logger.Error("dispatchMessage", zap.Error(err))
	}

	return nil
}

// handleSyncSettings Handler for inbound protobuf.SyncSettings
func (m *Messenger) handleSyncSettings(syncSettings protobuf.SyncSettings) error {

	if err := m.settings.SaveSyncSetting(
		settings.Currency,
		syncSettings.Currency.GetValue(),
		syncSettings.Currency.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.GifFavourites,
		syncSettings.GifFavorites.GetValue(),
		syncSettings.GifFavorites.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.GifRecents,
		syncSettings.GifRecents.GetValue(),
		syncSettings.GifRecents.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.MessagesFromContactsOnly,
		syncSettings.MessagesFromContactsOnly.GetValue(),
		syncSettings.MessagesFromContactsOnly.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.PreferredName,
		syncSettings.PreferredName.GetValue(),
		syncSettings.PreferredName.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.PreviewPrivacy,
		syncSettings.PreviewPrivacy.GetValue(),
		syncSettings.PreviewPrivacy.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.ProfilePicturesShowTo,
		syncSettings.ProfilePicturesShowTo.GetValue(),
		syncSettings.ProfilePicturesShowTo.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.ProfilePicturesVisibility,
		syncSettings.ProfilePicturesVisibility.GetValue(),
		syncSettings.ProfilePicturesVisibility.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.SendStatusUpdates,
		syncSettings.SendStatusUpdates.GetValue(),
		syncSettings.SendStatusUpdates.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.StickersPacksInstalled,
		syncSettings.StickerPacksInstalled.GetValue(),
		syncSettings.StickerPacksInstalled.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.StickersPacksPending,
		syncSettings.StickerPacksPending.GetValue(),
		syncSettings.StickerPacksPending.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.StickersRecentStickers,
		syncSettings.StickersRecentStickers.GetValue(),
		syncSettings.StickersRecentStickers.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.TelemetryServerURL,
		syncSettings.TelemetryServer_URL.GetValue(),
		syncSettings.TelemetryServer_URL.GetClock(),
	); err != nil {
		return err
	}

	return nil
}

// startSyncSettingsLoop watches the m.settings.SyncQueue and sends a sync message in response to a settings update
func (m *Messenger) startSyncSettingsLoop() {
	go func() {
		logger := m.logger.Named("SyncSettingsLoop")

		for {
			select {
			case s := <-m.settings.SyncQueue:
				if s.Field.SyncProtobufFactory() != nil {
					logger.Debug("setting for sync received")

					clock, chat := m.getLastClockWithRelatedChat()
					pb, amt := s.Field.SyncProtobufFactory()(s.Value, clock)

					rm, err := buildRawSyncSettingMessage(pb, amt, chat.ID)
					if err != nil {
						logger.Error("syncProtobufFactory", zap.Error(err), zap.Any("SyncSettingField", s))
						break
					}

					_, err = m.dispatchMessage(context.Background(), *rm)
					if err != nil {
						logger.Error("dispatchMessage", zap.Error(err))
						break
					}

					logger.Debug("message dispatched")
				}
			case <-m.quit:
				return
			}
		}
	}()
}
