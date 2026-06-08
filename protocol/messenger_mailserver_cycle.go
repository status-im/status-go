package protocol

import (
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/params"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
)

func (m *Messenger) AllMailservers() ([]messagingtypes.StoreNode, error) {
	// Get configured fleet
	fleet, err := m.getFleet()
	if err != nil {
		return nil, err
	}

	return m.allMailserversByFleet(fleet)
}

func (m *Messenger) allMailserversByFleet(fleet string) ([]messagingtypes.StoreNode, error) {
	// Get default mailservers for given fleet
	allMailservers := params.DefaultStoreNodes(fleet)
	return allMailservers, nil
}

func (m *Messenger) getFleet() (string, error) {
	var fleet string
	dbFleet, err := m.settings.GetFleet()
	if err != nil {
		return "", err
	}
	if dbFleet != "" {
		fleet = dbFleet
	} else if m.config.clusterConfig.Fleet != "" {
		fleet = m.config.clusterConfig.Fleet
	} else {
		fleet = params.FleetStatusProd
	}
	return fleet, nil
}

func (m *Messenger) asyncRequestAllHistoricMessages() {
	if !m.config.codeControlFlags.AutoRequestHistoricMessages {
		return
	}

	m.logger.Debug("asyncRequestAllHistoricMessages")

	go func() {
		defer gocommon.LogOnPanic()
		_, err := m.requestAllHistoricMessages(true, false)
		if err != nil {
			m.logger.Error("failed to request historic messages", zap.Error(err))
		}
	}()
}
