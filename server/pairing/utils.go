package pairing

import (
	"github.com/status-im/status-go/v10/protocol"
	"github.com/status-im/status-go/v10/protocol/encryption/multidevice"
)

func GetMessengerInstallationsMap(m *protocol.Messenger) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, installation := range m.Installations() {
		ids[installation.ID] = struct{}{}
	}
	return ids
}

func FindNewInstallations(m *protocol.Messenger, prevInstallationIds map[string]struct{}) *multidevice.Installation {
	for _, installation := range m.Installations() {
		if _, ok := prevInstallationIds[installation.ID]; !ok {
			return installation
		}
	}
	return nil
}
