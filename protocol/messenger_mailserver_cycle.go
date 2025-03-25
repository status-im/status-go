package protocol

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
	wakutypes "github.com/status-im/status-go/waku/types"
)

func (m *Messenger) AllMailservers() ([]wakutypes.Mailserver, error) {
	// Get configured fleet
	fleet, err := m.getFleet()
	if err != nil {
		return nil, err
	}

	return m.allMailserversByFleet(fleet)
}

func (m *Messenger) allMailserversByFleet(fleet string) ([]wakutypes.Mailserver, error) {
	// Get default mailservers for given fleet
	allMailservers := params.DefaultStoreNodes(fleet)

	// Add custom configured mailservers
	if m.mailserversDatabase != nil {
		customMailservers, err := m.mailserversDatabase.Mailservers()
		if err != nil {
			return nil, err
		}

		for _, c := range customMailservers {
			if c.Fleet == fleet {
				allMailservers = append(allMailservers, c)
			}
		}
	}

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
	if !m.config.codeControlFlags.AutoRequestHistoricMessages || m.transport.WakuVersion() == 1 {
		return
	}

	m.logger.Debug("asyncRequestAllHistoricMessages")

	go func() {
		defer gocommon.LogOnPanic()
		_, err := m.RequestAllHistoricMessages(false, true)
		if err != nil {
			m.logger.Error("failed to request historic messages", zap.Error(err))
		}
	}()
}

func (m *Messenger) GetPinnedStorenode() (peer.AddrInfo, error) {
	fleet, err := m.getFleet()
	if err != nil {
		return peer.AddrInfo{}, err
	}

	pinnedMailservers, err := m.settings.GetPinnedMailservers()
	if err != nil {
		return peer.AddrInfo{}, err
	}

	pinnedMailserver, ok := pinnedMailservers[fleet]
	if !ok {
		return peer.AddrInfo{}, nil
	}

	allMailservers, err := m.allMailserversByFleet(fleet)
	if err != nil {
		return peer.AddrInfo{}, err
	}

	for _, c := range allMailservers {
		if c.ID == pinnedMailserver {
			return c.PeerInfo()
		}
	}

	return peer.AddrInfo{}, nil
}

func (m *Messenger) UseStorenodes() (bool, error) {
	return m.settings.CanUseMailservers()
}

func (m *Messenger) Storenodes() ([]peer.AddrInfo, error) {
	mailservers, err := m.AllMailservers()
	if err != nil {
		return nil, err
	}

	var result []peer.AddrInfo
	for _, m := range mailservers {

		peerInfo, err := m.PeerInfo()
		if err != nil {
			return nil, err
		}
		result = append(result, peerInfo)
	}

	return result, nil
}

func (m *Messenger) checkForStorenodeCycleSignals() {
	defer gocommon.LogOnPanic()

	if m.transport.WakuVersion() != 2 {
		return
	}

	changed := m.transport.OnStorenodeChanged()
	notWorking := m.transport.OnStorenodeNotWorking()
	available := m.transport.OnStorenodeAvailable()

	allMailservers, err := m.AllMailservers()
	if err != nil {
		m.logger.Error("Could not retrieve mailserver list", zap.Error(err))
		return
	}

	mailserverMap := make(map[peer.ID]wakutypes.Mailserver)
	for _, ms := range allMailservers {
		peerID, err := ms.PeerID()
		if err != nil {
			m.logger.Error("could not retrieve peerID", zap.Error(err))
			return
		}
		mailserverMap[peerID] = ms
	}

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-notWorking:
			signal.SendMailserverNotWorking()

		case activeMailserver := <-changed:
			if activeMailserver != "" {
				ms, ok := mailserverMap[activeMailserver]
				if ok {
					signal.SendMailserverChanged(&ms)
				}
			} else {
				signal.SendMailserverChanged(nil)
			}
		case activeMailserver := <-available:
			if activeMailserver != "" {
				ms, ok := mailserverMap[activeMailserver]
				if ok {
					signal.SendMailserverAvailable(&ms)
				}
				m.asyncRequestAllHistoricMessages()
			}
		}
	}
}
