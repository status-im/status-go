package protocol

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	gocommon "github.com/status-im/status-go/common"
	"github.com/status-im/status-go/params"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
	"github.com/status-im/status-go/signal"
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

// requestHistoricMessagesAfterResume triggers a historic message sync when the
// app returns to the foreground (SetPaused(false)).
//
// While backgrounded, the history sync is deferred (see handleConnectionChange
// and checkForStorenodeCycleSignals, both gated on !isPaused()). The connection
// and storenode signals are edge-triggered, so when the device wakes from sleep
// nothing necessarily "changes" from the node's point of view and those signals
// may not fire again. Firing a one-shot request on resume can then silently
// no-op in shouldSync() because the storenode/connection isn't ready yet at that
// exact instant, and there is nothing left to retry it.
//
// To make resume reliable, we wait (bounded) for an available storenode before
// requesting. WaitForAvailableStoreNode returns immediately if a storenode is
// already available (covering the "nothing changed after wake" case) and
// otherwise blocks until the cycle re-establishes one. The underlying request is
// throttled/deduplicated via withHistoricSyncInFlight, so this is safe even if a
// storenode-available signal also fires.
func (m *Messenger) requestHistoricMessagesAfterResume() {
	if !m.config.codeControlFlags.AutoRequestHistoricMessages {
		return
	}

	m.shutdownWaitGroup.Add(1)
	go func() {
		defer gocommon.LogOnPanic()
		defer m.shutdownWaitGroup.Done()

		ctx, cancel := context.WithTimeout(m.ctx, resumeStorenodeWaitTimeout)
		defer cancel()

		if !m.messaging.WaitForAvailableStoreNode(ctx) {
			m.logger.Debug("no storenode available after resume, skipping historic sync")
			return
		}

		if _, err := m.requestAllHistoricMessages(true, false); err != nil {
			m.logger.Error("failed to request historic messages after resume", zap.Error(err))
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
	defer m.shutdownWaitGroup.Done()

	changed := m.messaging.OnStorenodeChanged()
	notWorking := m.messaging.OnStorenodeNotWorking()
	available := m.messaging.OnStorenodeAvailable()

	allMailservers, err := m.AllMailservers()
	if err != nil {
		m.logger.Error("Could not retrieve mailserver list", zap.Error(err))
		return
	}

	mailserverMap := make(map[peer.ID]messagingtypes.StoreNode)
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
		case <-m.quit:
			return
		case <-m.ctx.Done():
			return
		case <-notWorking:
			signal.SendStoreNodeNotWorking()

		case activeMailserver := <-changed:
			if activeMailserver != "" {
				ms, ok := mailserverMap[activeMailserver]
				if ok {
					signal.SendStoreNodeChanged(&ms)
				}
			} else {
				signal.SendStoreNodeChanged(nil)
			}
		case activeMailserver := <-available:
			if activeMailserver != "" {
				ms, ok := mailserverMap[activeMailserver]
				if ok {
					signal.SendStoreNodeAvailable(&ms)
				}
				// Skip history sync when backgrounded; SetPaused(false)
				// will trigger it when the app returns to foreground.
				fmt.Println("--------Messenger.checkForStorenodeCycleSignals", activeMailserver, m.isPaused())
				if !m.isPaused() {
					m.asyncRequestAllHistoricMessages()
				}
			}
		}
	}
}
