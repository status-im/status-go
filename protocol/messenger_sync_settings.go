package protocol

import (
	"context"

	"go.uber.org/zap"

	"github.com/status-im/status-go/protocol/protobuf"
)

func (m *Messenger) handleSyncSettings(messageState *ReceivedMessageState, settings protobuf.SyncSettings) error {

	return nil
}

func (m *Messenger) startSyncSettingsLoop() {
	go func() {
		logger := m.logger.Named("SyncSettingsLoop")

		for {
			select {
			case s := <-m.settings.SyncQueue:
				if s.Field.SyncProtobufFactory() != nil {
					logger.Debug("setting for sync received")

					clock, chat := m.getLastClockWithRelatedChat()
					rm, err := s.Field.SyncProtobufFactory()(chat.ID, s.Value, clock)
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
