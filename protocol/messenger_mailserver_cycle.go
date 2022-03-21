package protocol

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/signal"
)

const defaultBackoff = 10 * time.Second
const graylistBackoff = 3 * time.Minute

func (m *Messenger) mailserversByFleet(fleet string) []mailservers.Mailserver {
	var items []mailservers.Mailserver
	for _, ms := range mailservers.DefaultMailservers() {
		if ms.Fleet == fleet {
			items = append(items, ms)
		}
	}
	return items
}

type byRTTMsAndCanConnectBefore []SortedMailserver

func (s byRTTMsAndCanConnectBefore) Len() int {
	return len(s)
}

func (s byRTTMsAndCanConnectBefore) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byRTTMsAndCanConnectBefore) Less(i, j int) bool {
	// Slightly inaccurate as time sensitive sorting, but it does not matter so much
	now := time.Now()
	if s[i].CanConnectAfter.Before(now) && s[j].CanConnectAfter.Before(now) {
		return s[i].RTTMs < s[j].RTTMs
	}
	return s[i].CanConnectAfter.Before(s[j].CanConnectAfter)
}

func (m *Messenger) activeMailserverID() ([]byte, error) {
	if m.mailserverCycle.activeMailserver == nil {
		return nil, nil
	}

	return m.mailserverCycle.activeMailserver.IDBytes()
}

func (m *Messenger) StartMailserverCycle() error {

	if m.server == nil {
		m.logger.Warn("not starting mailserver cycle")
		return nil
	}

	m.logger.Debug("started mailserver cycle")

	m.mailserverCycle.events = make(chan *p2p.PeerEvent, 20)
	m.mailserverCycle.subscription = m.server.SubscribeEvents(m.mailserverCycle.events)

	go m.updateWakuV1PeerStatus()
	go m.updateWakuV2PeerStatus()
	return nil
}

func (m *Messenger) DisconnectActiveMailserver() {
	m.mailserverCycle.Lock()
	defer m.mailserverCycle.Unlock()
	m.disconnectActiveMailserver()
}

func (m *Messenger) disconnectMailserver() error {
	if m.mailserverCycle.activeMailserver == nil {
		m.logger.Info("no active mailserver")
		return nil
	}
	m.logger.Info("disconnecting active mailserver", zap.String("nodeID", m.mailserverCycle.activeMailserver.ID))
	m.mailPeersMutex.Lock()
	pInfo, ok := m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID]
	if ok {
		pInfo.status = disconnected
		pInfo.canConnectAfter = time.Now().Add(graylistBackoff)
		m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID] = pInfo
	} else {
		m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID] = peerStatus{
			status:          disconnected,
			mailserver:      *m.mailserverCycle.activeMailserver,
			canConnectAfter: time.Now().Add(graylistBackoff),
		}
	}
	m.mailPeersMutex.Unlock()

	if m.mailserverCycle.activeMailserver.Version == 2 {
		peerID, err := m.mailserverCycle.activeMailserver.PeerID()
		if err != nil {
			return err
		}
		err = m.transport.DropPeer(string(*peerID))
		if err != nil {
			m.logger.Warn("could not drop peer")
			return err
		}

	} else {
		node, err := m.mailserverCycle.activeMailserver.Enode()
		if err != nil {
			return err
		}
		m.server.RemovePeer(node)
	}

	m.mailserverCycle.activeMailserver = nil
	return nil
}

func (m *Messenger) disconnectActiveMailserver() {
	err := m.disconnectMailserver()
	if err != nil {
		m.logger.Error("failed to disconnect mailserver", zap.Error(err))
	}
	signal.SendMailserverChanged("", "")
}

func (m *Messenger) cycleMailservers() {
	m.logger.Info("Automatically switching mailserver")

	if m.mailserverCycle.activeMailserver != nil {
		m.disconnectActiveMailserver()
	}

	err := m.findNewMailserver()
	if err != nil {
		m.logger.Error("Error getting new mailserver", zap.Error(err))
	}
}

func poolSize(fleetSize int) int {
	return int(math.Ceil(float64(fleetSize) / 4))
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
		fleet = params.FleetProd
	}
	return fleet, nil
}

func (m *Messenger) allMailservers() ([]mailservers.Mailserver, error) {
	// Append user mailservers
	fleet, err := m.getFleet()
	if err != nil {
		return nil, err
	}

	allMailservers := m.mailserversByFleet(fleet)

	customMailservers, err := m.mailservers.Mailservers()
	if err != nil {
		return nil, err
	}

	for _, c := range customMailservers {
		if c.Fleet == fleet {
			allMailservers = append(allMailservers, c)
		}
	}

	return allMailservers, nil
}

type SortedMailserver struct {
	Address         string
	RTTMs           int
	CanConnectAfter time.Time
}

func (m *Messenger) findNewMailserver() error {
	pinnedMailserver, err := m.getPinnedMailserver()
	if err != nil {
		m.logger.Error("Could not obtain the pinned mailserver", zap.Error(err))
		return err
	}
	if pinnedMailserver != nil {
		return m.connectToMailserver(*pinnedMailserver)
	}

	// Append user mailservers
	fleet, err := m.getFleet()
	if err != nil {
		return err
	}

	allMailservers := m.mailserversByFleet(fleet)

	customMailservers, err := m.mailservers.Mailservers()
	if err != nil {
		return err
	}

	for _, c := range customMailservers {
		if c.Fleet == fleet {
			allMailservers = append(allMailservers, c)
		}
	}

	m.logger.Info("Finding a new mailserver...")

	var mailserverStr []string
	for _, m := range allMailservers {
		mailserverStr = append(mailserverStr, m.Address)
	}

	if len(allMailservers) == 0 {
		m.logger.Warn("no mailservers available") // Do nothing...
		return nil

	}

	var parseFn func(string) (string, error)
	if allMailservers[0].Version == 2 {
		parseFn = mailservers.MultiAddressToAddress
	} else {
		parseFn = mailservers.EnodeStringToAddr
	}

	pingResult, err := mailservers.DoPing(context.Background(), mailserverStr, 500, parseFn)
	if err != nil {
		return err
	}

	var availableMailservers []*mailservers.PingResult
	for _, result := range pingResult {
		if result.Err != nil {
			m.logger.Info("connecting error", zap.String("eerr", *result.Err))
			continue // The results with error are ignored
		}
		availableMailservers = append(availableMailservers, result)
	}

	if len(availableMailservers) == 0 {
		m.logger.Warn("No mailservers available") // Do nothing...
		return nil
	}

	mailserversByAddress := make(map[string]mailservers.Mailserver)
	for idx := range allMailservers {
		mailserversByAddress[allMailservers[idx].Address] = allMailservers[idx]
	}
	var sortedMailservers []SortedMailserver
	for _, ping := range availableMailservers {
		address := ping.Address
		ms := mailserversByAddress[address]
		sortedMailserver := SortedMailserver{
			Address: address,
			RTTMs:   *ping.RTTMs,
		}
		m.mailPeersMutex.Lock()
		pInfo, ok := m.mailserverCycle.peers[ms.ID]
		m.mailPeersMutex.Unlock()
		if ok {
			sortedMailserver.CanConnectAfter = pInfo.canConnectAfter
		}

		sortedMailservers = append(sortedMailservers, sortedMailserver)

	}
	sort.Sort(byRTTMsAndCanConnectBefore(sortedMailservers))

	// Picks a random mailserver amongs the ones with the lowest latency
	// The pool size is 1/4 of the mailservers were pinged successfully
	pSize := poolSize(len(sortedMailservers) - 1)
	if pSize <= 0 {
		pSize = len(sortedMailservers)
	}

	r, err := rand.Int(rand.Reader, big.NewInt(int64(pSize)))
	if err != nil {
		return err
	}

	msPing := sortedMailservers[r.Int64()]
	ms := mailserversByAddress[msPing.Address]
	m.logger.Info("connecting to mailserver", zap.String("address", ms.Address))
	return m.connectToMailserver(ms)
}

func (m *Messenger) activeMailserverStatus() (connStatus, error) {
	if m.mailserverCycle.activeMailserver == nil {
		return disconnected, errors.New("Active mailserver is not set")
	}

	mailserverID := m.mailserverCycle.activeMailserver.ID

	m.mailPeersMutex.Lock()
	status := m.mailserverCycle.peers[mailserverID].status
	m.mailPeersMutex.Unlock()

	return status, nil

}

func (m *Messenger) connectToMailserver(ms mailservers.Mailserver) error {

	m.logger.Info("connecting to mailserver", zap.Any("peer", ms.ID))

	m.mailserverCycle.activeMailserver = &ms
	signal.SendMailserverChanged(m.mailserverCycle.activeMailserver.Address, m.mailserverCycle.activeMailserver.ID)

	// Adding a peer and marking it as connected can't be executed sync in WakuV1, because
	// There's a delay between requesting a peer being added, and a signal being
	// received after the peer was added. So we first set the peer status as
	// Connecting and once a peerConnected signal is received, we mark it as
	// Connected
	activeMailserverStatus, err := m.activeMailserverStatus()
	if err != nil {
		return err
	}

	if activeMailserverStatus != connected {
		// Attempt to connect to mailserver by adding it as a peer

		if ms.Version == 2 {
			if err := m.transport.DialPeer(ms.Address); err != nil {
				m.logger.Error("failed to dial", zap.Error(err))
				return err
			}
		} else {
			node, err := ms.Enode()
			if err != nil {
				return err
			}
			m.server.AddPeer(node)
			if err := m.peerStore.Update([]*enode.Node{node}); err != nil {
				return err
			}
		}

		m.mailPeersMutex.Lock()
		pInfo, ok := m.mailserverCycle.peers[ms.ID]
		if ok {
			pInfo.status = connecting
			pInfo.lastConnectionAttempt = time.Now()
			pInfo.mailserver = ms
			m.mailserverCycle.peers[ms.ID] = pInfo
		} else {
			m.mailserverCycle.peers[ms.ID] = peerStatus{
				status:                connecting,
				mailserver:            ms,
				lastConnectionAttempt: time.Now(),
			}
		}
		m.mailPeersMutex.Unlock()
	}
	return nil
}

func (m *Messenger) getActiveMailserver() *mailservers.Mailserver {
	return m.mailserverCycle.activeMailserver
}

func (m *Messenger) isActiveMailserverAvailable() bool {
	mailserverStatus, err := m.activeMailserverStatus()
	if err != nil {
		return false
	}

	return mailserverStatus == connected
}

func (m *Messenger) mailserverAddressToID(uniqueID string) (string, error) {
	allMailservers, err := m.allMailservers()
	if err != nil {
		return "", err
	}

	for _, ms := range allMailservers {
		if uniqueID == ms.UniqueID() {
			return ms.ID, nil
		}

	}

	return "", nil
}

type ConnectedPeer struct {
	UniqueID string
}

func (m *Messenger) mailserverPeersInfo() []ConnectedPeer {
	var connectedPeers []ConnectedPeer
	for _, connectedPeer := range m.server.PeersInfo() {
		connectedPeers = append(connectedPeers, ConnectedPeer{
			// This is a bit fragile, but should work
			UniqueID: strings.TrimSuffix(connectedPeer.Enode, "?discport=0"),
		})
	}

	return connectedPeers
}

func (m *Messenger) penalizeMailserver(id string) {
	m.mailPeersMutex.Lock()
	defer m.mailPeersMutex.Unlock()
	pInfo, ok := m.mailserverCycle.peers[id]
	if !ok {
		pInfo.status = disconnected
	}

	pInfo.canConnectAfter = time.Now().Add(graylistBackoff)
	m.mailserverCycle.peers[id] = pInfo
}

func (m *Messenger) handleMailserverCycleEvent(connectedPeers []ConnectedPeer) error {
	m.logger.Debug("connected peers", zap.Any("connected", connectedPeers))
	m.logger.Debug("peers info", zap.Any("peer-info", m.mailserverCycle.peers))

	m.mailPeersMutex.Lock()
	for pID, pInfo := range m.mailserverCycle.peers {
		if pInfo.status == disconnected {
			continue
		}

		// Removing disconnected

		found := false
		for _, connectedPeer := range connectedPeers {
			id, err := m.mailserverAddressToID(connectedPeer.UniqueID)
			if err != nil {
				m.logger.Error("failed to convert id to hex", zap.Error(err))
				return err
			}

			if pID == id {
				found = true
				break
			}
		}
		if !found && (pInfo.status == connected || (pInfo.status == connecting && pInfo.lastConnectionAttempt.Add(8*time.Second).Before(time.Now()))) {
			m.logger.Info("peer disconnected", zap.String("peer", pID))
			pInfo.status = disconnected
			pInfo.canConnectAfter = time.Now().Add(defaultBackoff)
		}

		m.mailserverCycle.peers[pID] = pInfo
	}

	for _, connectedPeer := range connectedPeers {
		id, err := m.mailserverAddressToID(connectedPeer.UniqueID)
		if err != nil {
			m.logger.Error("failed to convert id to hex", zap.Error(err))
			return err
		}
		if id == "" {
			continue
		}
		pInfo, ok := m.mailserverCycle.peers[id]
		if !ok || pInfo.status != connected {
			m.logger.Info("peer connected", zap.String("peer", connectedPeer.UniqueID))
			pInfo.status = connected
			if pInfo.canConnectAfter.Before(time.Now()) {
				pInfo.canConnectAfter = time.Now().Add(defaultBackoff)
			}

			if m.mailserverCycle.activeMailserver != nil && id == m.mailserverCycle.activeMailserver.ID {
				m.logger.Info("mailserver available", zap.String("address", connectedPeer.UniqueID))
				m.EmitMailserverAvailable()
				signal.SendMailserverAvailable(m.mailserverCycle.activeMailserver.Address, m.mailserverCycle.activeMailserver.ID)
			}
			// Query mailserver
			go func() {
				_, err := m.performMailserverRequest(m.RequestAllHistoricMessages)
				if err != nil {
					m.logger.Error("could not perform mailserver request", zap.Error(err))
				}
			}()

			m.mailserverCycle.peers[id] = pInfo
		}
	}
	m.mailPeersMutex.Unlock()
	// Check whether we want to disconnect the mailserver
	if m.mailserverCycle.activeMailserver != nil {
		if m.mailserverCycle.activeMailserver.FailedRequests >= mailserverMaxFailedRequests {
			m.penalizeMailserver(m.mailserverCycle.activeMailserver.ID)
			signal.SendMailserverNotWorking()
			m.logger.Info("connecting too many failed requests")
			m.mailserverCycle.activeMailserver.FailedRequests = 0

			return m.connectToNewMailserverAndWait()
		}

		m.mailPeersMutex.Lock()
		pInfo, ok := m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID]
		m.mailPeersMutex.Unlock()
		if ok {
			if pInfo.status != connected && pInfo.lastConnectionAttempt.Add(30*time.Second).Before(time.Now()) {
				m.logger.Info("penalizing mailserver & disconnecting connecting", zap.String("id", m.mailserverCycle.activeMailserver.ID))

				signal.SendMailserverNotWorking()
				m.penalizeMailserver(m.mailserverCycle.activeMailserver.ID)
				m.disconnectActiveMailserver()
			}
		}

	} else {
		m.cycleMailservers()
	}

	m.logger.Debug("updated-peers", zap.Any("peers", m.mailserverCycle.peers))

	return nil
}

func (m *Messenger) updateWakuV1PeerStatus() {

	if m.transport.WakuVersion() != 1 {
		m.logger.Debug("waku version not 1, returning")
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := m.handleMailserverCycleEvent(m.mailserverPeersInfo())
			if err != nil {
				m.logger.Error("failed to handle mailserver cycle event", zap.Error(err))
				continue
			}

			ms := m.getActiveMailserver()
			if ms != nil {
				node, err := ms.Enode()
				if err != nil {
					m.logger.Error("failed to parse enode", zap.Error(err))
					continue
				}
				m.server.AddPeer(node)
				if err := m.peerStore.Update([]*enode.Node{node}); err != nil {
					m.logger.Error("failed to update peers", zap.Error(err))
					continue
				}
			}

		case <-m.mailserverCycle.events:
			err := m.handleMailserverCycleEvent(m.mailserverPeersInfo())
			if err != nil {
				m.logger.Error("failed to handle mailserver cycle event", zap.Error(err))
				return
			}
		case <-m.quit:
			close(m.mailserverCycle.events)
			m.mailserverCycle.subscription.Unsubscribe()
			return
		}
	}
}

func (m *Messenger) updateWakuV2PeerStatus() {
	if m.transport.WakuVersion() != 2 {
		m.logger.Debug("waku version not 2, returning")
		return
	}

	connSubscription, err := m.transport.SubscribeToConnStatusChanges()
	if err != nil {
		m.logger.Error("Could not subscribe to connection status changes", zap.Error(err))
	}

	for {
		select {
		case status := <-connSubscription.C:
			var connectedPeers []ConnectedPeer
			for id := range status.Peers {
				connectedPeers = append(connectedPeers, ConnectedPeer{UniqueID: id})
			}
			err := m.handleMailserverCycleEvent(connectedPeers)
			if err != nil {
				m.logger.Error("failed to handle mailserver cycle event", zap.Error(err))
				return
			}

		case <-m.quit:
			close(m.mailserverCycle.events)
			m.mailserverCycle.subscription.Unsubscribe()
			connSubscription.Unsubscribe()
			return
		}
	}
}

func (m *Messenger) getPinnedMailserver() (*mailservers.Mailserver, error) {
	fleet, err := m.getFleet()
	if err != nil {
		return nil, err
	}

	pinnedMailservers, err := m.settings.GetPinnedMailservers()
	if err != nil {
		return nil, err
	}

	pinnedMailserver, ok := pinnedMailservers[fleet]
	if !ok {
		return nil, nil
	}

	customMailservers, err := m.mailservers.Mailservers()
	if err != nil {
		return nil, err
	}

	fleetMailservers := mailservers.DefaultMailservers()

	for _, c := range fleetMailservers {
		if c.Fleet == fleet && c.ID == pinnedMailserver {
			return &c, nil
		}
	}

	for _, c := range customMailservers {
		if c.Fleet == fleet && c.ID == pinnedMailserver {
			return &c, nil
		}
	}

	return nil, nil
}

func (m *Messenger) EmitMailserverAvailable() {
	for _, s := range m.mailserverCycle.availabilitySubscriptions {
		s <- struct{}{}
		close(s)
		l := len(m.mailserverCycle.availabilitySubscriptions)
		m.mailserverCycle.availabilitySubscriptions = m.mailserverCycle.availabilitySubscriptions[:l-1]
	}
}

func (m *Messenger) SubscribeMailserverAvailable() chan struct{} {
	c := make(chan struct{})
	m.mailserverCycle.availabilitySubscriptions = append(m.mailserverCycle.availabilitySubscriptions, c)
	return c
}
