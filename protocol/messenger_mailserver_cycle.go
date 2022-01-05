package protocol

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/go-waku/waku/v2/dnsdisc"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/signal"
)

const defaultBackoff = 30 * time.Second

type byRTTMs []*mailservers.PingResult

func (s byRTTMs) Len() int {
	return len(s)
}

func (s byRTTMs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byRTTMs) Less(i, j int) bool {
	return *s[i].RTTMs < *s[j].RTTMs
}

func (m *Messenger) StartMailserverCycle() error {
	canUseMailservers, err := m.settings.CanUseMailservers()
	if err != nil {
		return err
	}
	if !canUseMailservers {
		return errors.New("mailserver use is not allowed")
	}

	m.logger.Debug("started mailserver cycle")

	m.mailserverCycle.events = make(chan *p2p.PeerEvent, 20)
	m.mailserverCycle.subscription = m.server.SubscribeEvents(m.mailserverCycle.events)

	go m.checkMailserverConnection()
	go m.updateWakuV1PeerStatus()
	go m.updateWakuV2PeerStatus()
	return nil
}

func (m *Messenger) DisconnectActiveMailserver() {
	m.mailserverCycle.Lock()
	defer m.mailserverCycle.Unlock()
	m.disconnectActiveMailserver()
}

func (m *Messenger) disconnectV1Mailserver() {
	// TODO: remove this function once WakuV1 is deprecated
	if m.mailserverCycle.activeMailserver == nil {
		return
	}
	m.logger.Info("Disconnecting active mailserver", zap.Any("nodeID", m.mailserverCycle.activeMailserver.ID()))
	pInfo, ok := m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID().String()]
	if ok {
		pInfo.status = disconnected
		pInfo.canConnectAfter = time.Now().Add(defaultBackoff)
		m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID().String()] = pInfo
	} else {
		m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID().String()] = peerStatus{
			status:          disconnected,
			canConnectAfter: time.Now().Add(defaultBackoff),
		}
	}

	m.server.RemovePeer(m.mailserverCycle.activeMailserver)
	m.mailserverCycle.activeMailserver = nil
}

func (m *Messenger) disconnectStoreNode() {
	if m.mailserverCycle.activeStoreNode == nil {
		return
	}
	m.logger.Info("Disconnecting active storeNode", zap.Any("nodeID", m.mailserverCycle.activeStoreNode.Pretty()))
	pInfo, ok := m.mailserverCycle.peers[string(*m.mailserverCycle.activeStoreNode)]
	if ok {
		pInfo.status = disconnected
		pInfo.canConnectAfter = time.Now().Add(defaultBackoff)
		m.mailserverCycle.peers[string(*m.mailserverCycle.activeStoreNode)] = pInfo
	} else {
		m.mailserverCycle.peers[string(*m.mailserverCycle.activeStoreNode)] = peerStatus{
			status:          disconnected,
			canConnectAfter: time.Now().Add(defaultBackoff),
		}
	}

	err := m.transport.DropPeer(string(*m.mailserverCycle.activeStoreNode))
	if err != nil {
		m.logger.Warn("Could not drop peer")
	}

	m.mailserverCycle.activeStoreNode = nil
}

func (m *Messenger) disconnectActiveMailserver() {
	switch m.transport.WakuVersion() {
	case 1:
		m.disconnectV1Mailserver()
	case 2:
		m.disconnectStoreNode()
	}
	signal.SendMailserverChanged("")
}

func (m *Messenger) cycleMailservers() {
	m.mailserverCycle.Lock()
	defer m.mailserverCycle.Unlock()

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

func (m *Messenger) findNewMailserver() error {
	switch m.transport.WakuVersion() {
	case 1:
		return m.findNewMailserverV1()
	case 2:
		return m.findStoreNode()
	default:
		return errors.New("waku version is not supported")
	}
}

func (m *Messenger) findStoreNode() error {
	allMailservers := parseStoreNodeConfig(m.config.clusterConfig.StoreNodes)

	// TODO: append user mailservers once that functionality is available for waku2

	var mailserverList []multiaddr.Multiaddr
	now := time.Now()
	for _, node := range allMailservers {
		pID, err := getPeerID(node)
		if err != nil {
			continue
		}

		pInfo, ok := m.mailserverCycle.peers[string(pID)]
		if !ok || pInfo.canConnectAfter.Before(now) {
			mailserverList = append(mailserverList, node)
		}
	}

	m.logger.Info("Finding a new store node...")

	var mailserverStr []string
	for _, m := range mailserverList {
		mailserverStr = append(mailserverStr, m.String())
	}

	pingResult, err := mailservers.DoPing(context.Background(), mailserverStr, 500, mailservers.MultiAddressToAddress)
	if err != nil {
		return err
	}

	var availableMailservers []*mailservers.PingResult
	for _, result := range pingResult {
		if result.Err != nil {
			continue // The results with error are ignored
		}
		availableMailservers = append(availableMailservers, result)
	}
	sort.Sort(byRTTMs(availableMailservers))

	if len(availableMailservers) == 0 {
		m.logger.Warn("No store nodes available") // Do nothing...
		return nil
	}

	// Picks a random mailserver amongs the ones with the lowest latency
	// The pool size is 1/4 of the mailservers were pinged successfully
	pSize := poolSize(len(availableMailservers) - 1)
	if pSize <= 0 {
		m.logger.Warn("No store nodes available") // Do nothing...
		return nil
	}

	r, err := rand.Int(rand.Reader, big.NewInt(int64(pSize)))
	if err != nil {
		return err
	}

	return m.connectToStoreNode(parseMultiaddresses([]string{availableMailservers[r.Int64()].Address})[0])
}

func (m *Messenger) findNewMailserverV1() error {
	// TODO: remove this function once WakuV1 is deprecated

	allMailservers := parseNodes(m.config.clusterConfig.TrustedMailServers)

	// Append user mailservers
	var fleet string
	dbFleet, err := m.settings.GetFleet()
	if err != nil {
		return err
	}
	if dbFleet != "" {
		fleet = dbFleet
	} else if m.config.clusterConfig.Fleet != "" {
		fleet = m.config.clusterConfig.Fleet
	} else {
		fleet = params.FleetProd
	}

	customMailservers, err := m.mailservers.Mailservers()
	if err != nil {
		return err
	}
	for _, c := range customMailservers {
		if c.Fleet == fleet {
			mNode, err := enode.ParseV4(c.Address)
			if err != nil {
				allMailservers = append(allMailservers, mNode)
			}
		}
	}

	var mailserverList []*enode.Node
	now := time.Now()
	for _, node := range allMailservers {
		pInfo, ok := m.mailserverCycle.peers[node.ID().String()]
		if !ok || pInfo.canConnectAfter.Before(now) {
			mailserverList = append(mailserverList, node)
		}
	}

	m.logger.Info("Finding a new mailserver...")

	var mailserverStr []string
	for _, m := range mailserverList {
		mailserverStr = append(mailserverStr, m.String())
	}

	pingResult, err := mailservers.DoPing(context.Background(), mailserverStr, 500, mailservers.EnodeStringToAddr)
	if err != nil {
		return err
	}

	var availableMailservers []*mailservers.PingResult
	for _, result := range pingResult {
		if result.Err != nil {
			continue // The results with error are ignored
		}
		availableMailservers = append(availableMailservers, result)
	}
	sort.Sort(byRTTMs(availableMailservers))

	if len(availableMailservers) == 0 {
		m.logger.Warn("No mailservers available") // Do nothing...
		return nil
	}

	// Picks a random mailserver amongs the ones with the lowest latency
	// The pool size is 1/4 of the mailservers were pinged successfully
	pSize := poolSize(len(availableMailservers) - 1)
	r, err := rand.Int(rand.Reader, big.NewInt(int64(pSize)))
	if err != nil {
		return err
	}

	return m.connectToMailserver(parseNodes([]string{availableMailservers[r.Int64()].Address})[0])
}

func (m *Messenger) activeMailserverStatus() (connStatus, error) {
	var mailserverID string
	switch m.transport.WakuVersion() {
	case 1:
		if m.mailserverCycle.activeMailserver == nil {
			return disconnected, errors.New("Active mailserver is not set")
		}
		mailserverID = m.mailserverCycle.activeMailserver.ID().String()
	case 2:
		if m.mailserverCycle.activeStoreNode == nil {
			return disconnected, errors.New("Active storenode is not set")
		}
		mailserverID = string(*m.mailserverCycle.activeStoreNode)
	default:
		return disconnected, errors.New("waku version is not supported")
	}

	return m.mailserverCycle.peers[mailserverID].status, nil
}

func (m *Messenger) connectToMailserver(node *enode.Node) error {
	// TODO: remove this function once WakuV1 is deprecated

	if m.transport.WakuVersion() != 1 {
		return nil // This can only be used with wakuV1
	}

	m.logger.Info("Connecting to mailserver", zap.Any("peer", node.ID()))
	nodeConnected := false

	m.mailserverCycle.activeMailserver = node
	signal.SendMailserverChanged(m.mailserverCycle.activeMailserver.String())

	// Adding a peer and marking it as connected can't be executed sync in WakuV1, because
	// There's a delay between requesting a peer being added, and a signal being
	// received after the peer was added. So we first set the peer status as
	// Connecting and once a peerConnected signal is received, we mark it as
	// Connected
	activeMailserverStatus, err := m.activeMailserverStatus()
	if err != nil {
		return err
	}

	if activeMailserverStatus == connected {
		nodeConnected = true
	} else {
		// Attempt to connect to mailserver by adding it as a peer
		m.SetMailserver(node.ID().Bytes())
		m.server.AddPeer(node)
		if err := m.peerStore.Update([]*enode.Node{node}); err != nil {
			return err
		}

		pInfo, ok := m.mailserverCycle.peers[node.ID().String()]
		if ok {
			pInfo.status = connecting
			pInfo.lastConnectionAttempt = time.Now()
			m.mailserverCycle.peers[node.ID().String()] = pInfo
		} else {
			m.mailserverCycle.peers[node.ID().String()] = peerStatus{
				status:                connecting,
				lastConnectionAttempt: time.Now(),
			}
		}
	}

	if nodeConnected {
		m.logger.Info("Mailserver available")
		signal.SendMailserverAvailable(m.mailserverCycle.activeMailserver.String())
	}

	return nil
}

func (m *Messenger) connectToStoreNode(node multiaddr.Multiaddr) error {
	if m.transport.WakuVersion() != 2 {
		return nil // This can only be used with wakuV2
	}

	m.logger.Info("Connecting to storenode", zap.Any("peer", node))

	nodeConnected := false

	peerID, err := getPeerID(node)
	if err != nil {
		return err
	}

	m.mailserverCycle.activeStoreNode = &peerID
	signal.SendMailserverChanged(m.mailserverCycle.activeStoreNode.Pretty())

	// Adding a peer and marking it as connected can't be executed sync in WakuV1, because
	// There's a delay between requesting a peer being added, and a signal being
	// received after the peer was added. So we first set the peer status as
	// Connecting and once a peerConnected signal is received, we mark it as
	// Connected
	activeMailserverStatus, err := m.activeMailserverStatus()
	if err != nil {
		return err
	}

	if activeMailserverStatus == connected {
		nodeConnected = true
	} else {
		// Attempt to connect to mailserver by adding it as a peer
		m.SetMailserver([]byte(peerID.Pretty()))
		if err := m.transport.DialPeer(node.String()); err != nil {
			return err
		}

		pInfo, ok := m.mailserverCycle.peers[string(peerID)]
		if ok {
			pInfo.status = connected
			pInfo.lastConnectionAttempt = time.Now()
		} else {
			m.mailserverCycle.peers[string(peerID)] = peerStatus{
				status:                connected,
				lastConnectionAttempt: time.Now(),
			}
		}

		nodeConnected = true
	}

	if nodeConnected {
		m.logger.Info("Storenode available")
		signal.SendMailserverAvailable(m.mailserverCycle.activeStoreNode.Pretty())
	}

	return nil
}

func (m *Messenger) isActiveMailserverAvailable() bool {
	m.mailserverCycle.RLock()
	defer m.mailserverCycle.RUnlock()

	mailserverStatus, err := m.activeMailserverStatus()
	if err != nil {
		return false
	}

	return mailserverStatus == connected
}

func (m *Messenger) updateWakuV2PeerStatus() {
	if m.transport.WakuVersion() != 2 {
		return // This can only be used with wakuV2
	}

	connSubscription, err := m.transport.SubscribeToConnStatusChanges()
	if err != nil {
		m.logger.Error("Could not subscribe to connection status changes", zap.Error(err))
	}

	for {
		select {
		case status := <-connSubscription.C:
			m.mailserverCycle.Lock()

			for pID, pInfo := range m.mailserverCycle.peers {
				if pInfo.status == disconnected {
					continue
				}

				// Removing disconnected

				found := false
				for connectedPeer := range status.Peers {
					peerID, err := peer.Decode(connectedPeer)
					if err != nil {
						continue
					}

					if string(peerID) == pID {
						found = true
						break
					}
				}
				if !found && pInfo.status == connected {
					m.logger.Info("Peer disconnected", zap.String("peer", peer.ID(pID).Pretty()))
					pInfo.status = disconnected
					pInfo.canConnectAfter = time.Now().Add(defaultBackoff)
				}

				m.mailserverCycle.peers[pID] = pInfo
			}

			for connectedPeer := range status.Peers {
				peerID, err := peer.Decode(connectedPeer)
				if err != nil {
					continue
				}

				pInfo, ok := m.mailserverCycle.peers[string(peerID)]
				if !ok || pInfo.status != connected {
					m.logger.Info("Peer connected", zap.String("peer", connectedPeer))
					pInfo.status = connected
					pInfo.canConnectAfter = time.Now().Add(defaultBackoff)
					m.mailserverCycle.peers[string(peerID)] = pInfo
				}
			}
			m.mailserverCycle.Unlock()

		case <-m.quit:
			connSubscription.Unsubscribe()
			return
		}
	}
}

func (m *Messenger) updateWakuV1PeerStatus() {
	// TODO: remove this function once WakuV1 is deprecated

	if m.transport.WakuVersion() != 1 {
		return // This can only be used with wakuV1
	}

	for {
		select {
		case <-m.mailserverCycle.events:
			connectedPeers := m.server.PeersInfo()
			m.mailserverCycle.Lock()

			for pID, pInfo := range m.mailserverCycle.peers {
				if pInfo.status == disconnected {
					continue
				}

				// Removing disconnected

				found := false
				for _, connectedPeer := range connectedPeers {
					if enode.HexID(connectedPeer.ID) == enode.HexID(pID) {
						found = true
						break
					}
				}
				if !found && (pInfo.status == connected || (pInfo.status == connecting && pInfo.lastConnectionAttempt.Add(8*time.Second).Before(time.Now()))) {
					m.logger.Info("Peer disconnected", zap.String("peer", enode.HexID(pID).String()))
					pInfo.status = disconnected
					pInfo.canConnectAfter = time.Now().Add(defaultBackoff)
				}

				m.mailserverCycle.peers[pID] = pInfo
			}

			for _, connectedPeer := range connectedPeers {
				hexID := enode.HexID(connectedPeer.ID).String()
				pInfo, ok := m.mailserverCycle.peers[hexID]
				if !ok || pInfo.status != connected {
					m.logger.Info("Peer connected", zap.String("peer", hexID))
					pInfo.status = connected
					pInfo.canConnectAfter = time.Now().Add(defaultBackoff)
					if m.mailserverCycle.activeMailserver != nil && hexID == m.mailserverCycle.activeMailserver.ID().String() {
						m.logger.Info("Mailserver available")
						signal.SendMailserverAvailable(m.mailserverCycle.activeMailserver.String())
					}
					m.mailserverCycle.peers[hexID] = pInfo
				}
			}
			m.mailserverCycle.Unlock()
		case <-m.quit:
			m.mailserverCycle.Lock()
			defer m.mailserverCycle.Unlock()
			close(m.mailserverCycle.events)
			m.mailserverCycle.subscription.Unsubscribe()
			return
		}
	}
}

func (m *Messenger) checkMailserverConnection() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		m.logger.Info("Verifying mailserver connection state...")
		//	m.settings.GetPinnedMailserver
		//if pinnedMailserver != "" && self.activeMailserver != pinnedMailserver {
		// connect to current mailserver from the settings
		// self.mailservers = pinnedMailserver
		// self.connect(pinnedMailserver)
		//} else {
		// or setup a random mailserver:
		if !m.isActiveMailserverAvailable() {
			m.cycleMailservers()
		}
		// }

		select {
		case <-m.quit:
			return
		case <-ticker.C:
			continue
		}
	}
}

func parseNodes(enodes []string) []*enode.Node {
	var nodes []*enode.Node
	for _, item := range enodes {
		parsedPeer, err := enode.ParseV4(item)
		if err == nil {
			nodes = append(nodes, parsedPeer)
		}
	}
	return nodes
}

func parseMultiaddresses(addresses []string) []multiaddr.Multiaddr {
	var result []multiaddr.Multiaddr
	for _, item := range addresses {
		ma, err := multiaddr.NewMultiaddr(item)
		if err == nil {
			result = append(result, ma)
		}
	}
	return result
}

func parseStoreNodeConfig(addresses []string) []multiaddr.Multiaddr {
	// TODO: once a scoring/reputation mechanism is added to waku,
	// this function can be modified to retrieve the storenodes
	// from waku peerstore.
	// We don't do that now because we can't trust any random storenode
	// So we use only those specified in the cluster config
	var result []multiaddr.Multiaddr
	var dnsDiscWg sync.WaitGroup

	maChan := make(chan multiaddr.Multiaddr, 1000)

	for _, addrString := range addresses {
		if strings.HasPrefix(addrString, "enrtree://") {
			// Use DNS Discovery
			dnsDiscWg.Add(1)
			go func(addr string) {
				defer dnsDiscWg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				multiaddresses, err := dnsdisc.RetrieveNodes(ctx, addr)
				if err == nil {
					for _, ma := range multiaddresses {
						maChan <- ma
					}
				}
			}(addrString)

		} else {
			// It's a normal multiaddress
			ma, err := multiaddr.NewMultiaddr(addrString)
			if err == nil {
				maChan <- ma
			}
		}
	}
	dnsDiscWg.Wait()
	close(maChan)
	for ma := range maChan {
		result = append(result, ma)
	}

	return result
}

func getPeerID(addr multiaddr.Multiaddr) (peer.ID, error) {
	idStr, err := addr.ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		return "", err
	}
	return peer.Decode(idStr)
}
