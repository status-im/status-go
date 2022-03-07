package protocol

import (
	"context"

	"go.uber.org/zap"

	"github.com/status-im/status-go/multiaccounts/errors"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/protobuf"
)

// syncSettings syncs all settings that are syncable
func (m *Messenger) syncSettings() error {
	logger := m.logger.Named("syncSettings")

	s, err := m.settings.GetSettings()
	if err != nil {
		return err
	}

	// Do not use the network clock, use the db value
	_, chat := m.getLastClockWithRelatedChat()

	var errs []error
	for _, sf := range settings.SettingFieldRegister {
		if sf.SyncProtobufFactory() != nil && sf.SyncProtobufFactory().Struct != nil {
			// Pull clock from the db
			clock, err := m.settings.GetSettingLastSynced(sf)
			if err != nil {
				logger.Error("m.settings.GetSettingLastSynced", zap.Error(err), zap.Any("SettingField", sf))
				return err
			}

			rm, err := sf.SyncProtobufFactory().Struct(s, clock, chat.ID)
			if err != nil {
				// Collect errors to give other sync messages a chance to send
				logger.Error("SyncProtobufFactory.Struct", zap.Error(err))
				errs = append(errs, err)
			}

			_, err = m.dispatchMessage(context.Background(), *rm)
			if err != nil {
				logger.Error("dispatchMessage", zap.Error(err))
				return err
			}
		}
	}

	if len(errs) != 0 {
		// return just the first error, the others have been logged
		return errs[0]
	}
	return nil
}

func (m *Messenger) handleSyncSetting(response *MessengerResponse, field settings.SettingField, value interface{}, clock uint64) error {
	err := m.settings.SaveSyncSetting(field, value, clock)
	if err == errors.ErrNewClockOlderThanCurrent {
		m.logger.Info("handleSyncSetting - SaveSyncSetting :", zap.Error(err))
		return nil
	}
	if err != nil {
		return err
	}

	response.Settings = append(response.Settings, &settings.SyncSettingField{SettingField:field, Value: value})
	return nil
}

// handleSyncSettings Handler for inbound protobuf.SyncSettings
// TODO remove this func
func (m *Messenger) handleSyncSettings(syncSettings protobuf.SyncSettings) error {

	if err := m.settings.SaveSyncSetting(
		settings.Currency,
		syncSettings.Currency.GetValue(),
		syncSettings.Currency.GetClock(),
	); err != nil {
		return err
	}

	if err := m.settings.SaveSyncSetting(
		settings.GifAPIKey,
		syncSettings.GifApiKey.GetValue(),
		syncSettings.GifApiKey.GetClock(),
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
				if s.SyncProtobufFactory() != nil && s.SyncProtobufFactory().Interface != nil {
					logger.Debug("setting for sync received")

					clock, chat := m.getLastClockWithRelatedChat()

					// Only the messenger has access to the clock, so set the settings sync clock here.
					err :=  m.settings.SetSettingLastSynced(s.SettingField, clock)
					if err != nil {
						logger.Error("m.settings.SetSettingLastSynced", zap.Error(err))
						break
					}

					rm, err := s.SyncProtobufFactory().Interface(s.Value, clock, chat.ID)
					if err != nil {
						logger.Error("SyncProtobufFactory", zap.Error(err), zap.Any("SyncSettingField", s))
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
