package pairing

import (
	"github.com/status-im/status-go/internal/protocol"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
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
