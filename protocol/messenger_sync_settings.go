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
		if sf.SyncProtobufFactory() != nil && sf.SyncProtobufFactory().FromStruct() != nil {
			// Pull clock from the db
			clock, err := m.settings.GetSettingLastSynced(sf)
			if err != nil {
				logger.Error("m.settings.GetSettingLastSynced", zap.Error(err), zap.Any("SettingField", sf))
				return err
			}

			rm, err := sf.SyncProtobufFactory().FromStruct()(s, clock, chat.ID)
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
			logger.Debug("dispatchMessage success", zap.Any("rm", rm))
		}
	}

	if len(errs) != 0 {
		// return just the first error, the others have been logged
		return errs[0]
	}
	return nil
}

func (m *Messenger) handleSyncSetting(response *MessengerResponse, syncSetting *protobuf.SyncSetting) error {
	field, err := settings.GetFieldFromProtobufType(syncSetting.Type)
	if err != nil {
		return err
	}

	value := field.SyncProtobufFactory().ValueFrom()(syncSetting)

	err = m.settings.SaveSyncSetting(field, value, syncSetting.Clock)
	if err == errors.ErrNewClockOlderThanCurrent {
		m.logger.Info("handleSyncSetting - SaveSyncSetting :", zap.Error(err))
		return nil
	}
	if err != nil {
		return err
	}

	response.Settings = append(response.Settings, &settings.SyncSettingField{SettingField: field, Value: value})
	return nil
}

// startSyncSettingsLoop watches the m.settings.SyncQueue and sends a sync message in response to a settings update
func (m *Messenger) startSyncSettingsLoop() {
	go func() {
		logger := m.logger.Named("SyncSettingsLoop")

		for {
			select {
			case s := <-m.settings.SyncQueue:
				if s.SyncProtobufFactory() != nil && s.SyncProtobufFactory().FromInterface() != nil {
					logger.Debug("setting for sync received from settings.SyncQueue")

					clock, chat := m.getLastClockWithRelatedChat()

					// Only the messenger has access to the clock, so set the settings sync clock here.
					err := m.settings.SetSettingLastSynced(s.SettingField, clock)
					if err != nil {
						logger.Error("m.settings.SetSettingLastSynced", zap.Error(err))
						break
					}

					rm, err := s.SyncProtobufFactory().FromInterface()(s.Value, clock, chat.ID)
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
