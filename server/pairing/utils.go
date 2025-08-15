package pairing

import (
	messagingtypes "github.com/status-im/status-go/messaging/types"
	"github.com/status-im/status-go/protocol"
)

func GetMessengerInstallationsMap(m *protocol.Messenger) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, installation := range m.Installations() {
		ids[installation.ID] = struct{}{}
	}
	return ids
}

func FindNewInstallations(m *protocol.Messenger, prevInstallationIds map[string]struct{}) *messagingtypes.Installation {
	for _, installation := range m.Installations() {
		if _, ok := prevInstallationIds[installation.ID]; !ok {
			return installation
		}
	}
	return nil
}
