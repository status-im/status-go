package protocol

import (
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/protocol/requests"
)

func (m *Messenger) SetLightClient(request *requests.SetLightClient) error {
	return nodecfg.SetLightClient(m.database, request.Enabled)
}
