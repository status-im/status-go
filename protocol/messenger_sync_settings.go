package protocol

import (
	"context"
	errorsLib "errors"

	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/internal/db/multiaccounts/settings"
	"github.com/status-im/status-go/protocol/common"
	"github.com/status-im/status-go/protocol/protobuf"
)

// syncSettings syncs all settings that are syncable
func (m *Messenger) prepareSyncSettingsMessages(currentClock uint64, prepareForBackup bool) (resultRaw []*common.RawMessage, resultSync []*protobuf.SyncSetting, errorResult error) {
	var errors []error
	s, err := m.settings.GetSettings()
	if err != nil {
		errorResult = err
		return
	}

	logger := m.logger.Named("prepareSyncSettings")
	// Do not use the network clock, use the db value
	_, chat := m.getLastClockWithRelatedChat()

	for _, sf := range settings.SettingFieldRegister {
		if sf.CanSync(settings.FromStruct) {
			// DisplayName is backed up via `protobuf.BackedUpProfile` message.
			if prepareForBackup && sf.SyncProtobufFactory().SyncSettingProtobufType() == protobuf.SyncSetting_DISPLAY_NAME {
				continue
			}

			// Pull clock from the db
			clock, err := m.settings.GetSettingLastSynced(sf)
			if err != nil {
				logger.Error("m.settings.GetSettingLastSynced", zap.Error(err), zap.String("SettingField", sf.GetDBName()))
				errorResult = err
				return
			}
			if clock == 0 {
				clock = currentClock
			}

			// Build protobuf
			rm, sm, err := sf.SyncProtobufFactory().FromStruct()(s, clock, chat.ID)
			if err != nil {
				// Collect errors to give other sync messages a chance to send
				logger.Error("SyncProtobufFactory.Struct", zap.Error(err))
				errors = append(errors, err)
			}

			resultRaw = append(resultRaw, rm)
			resultSync = append(resultSync, sm)
		}
	}
	errorResult = errorsLib.Join(errors...)
	return
}

func (m *Messenger) syncSettings(rawMessageHandler RawMessageHandler) error {
	logger := m.logger.Named("syncSettings")

	clock, _ := m.getLastClockWithRelatedChat()
	rawMessages, _, err := m.prepareSyncSettingsMessages(clock, false)

	if err != nil {
		return err
	}

	for _, rm := range rawMessages {
		_, err := rawMessageHandler(context.Background(), *rm)
		if err != nil {
			logger.Error("dispatchMessage", zap.Error(err))
			return err
		}
		logger.Debug("dispatchMessage success", zap.Any("rm", rm))
	}

	return nil
}

// startSyncSettingsLoop watches the m.settings.SyncQueue and sends a sync message in response to a settings update
func (m *Messenger) startSyncSettingsLoop() {
	go func() {
		defer gocommon.LogOnPanic()
		logger := m.logger.Named("SyncSettingsLoop")

		for {
			select {
			case s := <-m.settings.GetSyncQueue():
				if s.CanSync(settings.FromInterface) {
					logger.Debug("setting for sync received from settings.SyncQueue")

					clock, chat := m.getLastClockWithRelatedChat()

					// Only the messenger has access to the clock, so set the settings sync clock here.
					err := m.settings.SetSettingLastSynced(s.SettingField, clock)
					if err != nil {
						logger.Error("m.settings.SetSettingLastSynced", zap.Error(err))
						break
					}
					rm, _, err := s.SyncProtobufFactory().FromInterface()(s.Value, clock, chat.ID)
					if err != nil {
						logger.Error("SyncProtobufFactory().FromInterface", zap.Error(err), zap.String("SyncSettingField", s.GetDBName()))
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

func (m *Messenger) startSettingsChangesLoop() {
	channel := m.settings.SubscribeToChanges()
	go func() {
		defer gocommon.LogOnPanic()
		for {
			select {
			case s := <-channel:
				switch s.GetReactName() {
				case settings.DisplayName.GetReactName():
					m.selfContact.DisplayName = s.Value.(string)
					m.publishSelfContactSubscriptions(&SelfContactChangeEvent{DisplayNameChanged: true})
				case settings.PreferredName.GetReactName():
					m.selfContact.EnsName = s.Value.(string)
					m.publishSelfContactSubscriptions(&SelfContactChangeEvent{PreferredNameChanged: true})
				case settings.Bio.GetReactName():
					m.selfContact.Bio = s.Value.(string)
					m.publishSelfContactSubscriptions(&SelfContactChangeEvent{BioChanged: true})
				}
			case <-m.quit:
				return
			}
		}
	}()
}
