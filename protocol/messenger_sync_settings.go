package protocol

import (
	"github.com/status-im/status-go/protocol/protobuf"
)

func (m *Messenger) handleSyncSettings(messageState *ReceivedMessageState, settings protobuf.SyncSettings) error {

	return nil
}

func (m *Messenger) startSyncSettingsLoop() {
	go func() {
		for {
			select {
			case s := <-m.settings.SyncQueue:
				if s.Field.ShouldSync {

				}
			case <-m.quit:
				return
			}
		}
	}()
}
