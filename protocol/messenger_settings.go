package protocol

import (
	"github.com/status-im/status-go/nodecfg"
	"github.com/status-im/status-go/protocol/requests"
)

func (m *Messenger) SetLightClient(request *requests.SetLightClient) error {
	return nodecfg.SetLightClient(m.database, request.Enabled)
}

func (m *Messenger) SetLogLevel(request *requests.SetLogLevel) error {
	if err := request.Validate(); err != nil {
		return err
	}

	return nodecfg.SetLogLevel(m.database, request.LogLevel)
}

func (m *Messenger) SetCustomNodes(request *requests.SetCustomNodes) error {
	return nodecfg.SetWakuV2CustomNodes(m.database, request.CustomNodes)
}
