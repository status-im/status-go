package protocol

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"net"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/storenodes"
	"github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/signal"
)

const defaultBackoff = 10 * time.Second
const graylistBackoff = 3 * time.Minute
const backoffByUserAction = 0
const isAndroidEmulator = runtime.GOOS == "android" && runtime.GOARCH == "amd64"
const findNearestStorenode = !isAndroidEmulator
const overrideDNS = runtime.GOOS == "android" || runtime.GOOS == "ios"
const bootstrapDNS = "8.8.8.8:53"

func (m *Messenger) storenodesByFleet(fleet string) []mailservers.Mailserver {
	return mailservers.DefaultStorenodesByFleet(fleet)
}

type byRTTMsAndCanConnectBefore []SortedStorenodes

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

func (m *Messenger) StartStorenodeCycle(storenodes []mailservers.Mailserver) error {
	m.storenodeCycle.allStorenodes = storenodes

	if len(storenodes) == 0 {
		m.logger.Warn("not starting storenode cycle: empty storenode list")
		return nil
	}
	for _, storenode := range storenodes {

		peerInfo, err := storenode.PeerInfo()
		if err != nil {
			return err
		}

		for _, addr := range peerInfo.Addrs {
			_, err := m.transport.AddStorePeer(addr)
			if err != nil {
				return err
			}
		}
	}
	go m.verifyStorenodeStatus()

	m.logger.Debug("starting storenode cycle",
		zap.Uint("WakuVersion", m.transport.WakuVersion()),
		zap.Any("storenode", storenodes),
	)

	return nil
}

func (m *Messenger) DisconnectActiveStorenode() {
	m.storenodeCycle.Lock()
	defer m.storenodeCycle.Unlock()
	m.disconnectActiveStorenode(graylistBackoff)
}

func (m *Messenger) disconnecStorenode(backoffDuration time.Duration) error {
	if m.storenodeCycle.activeStorenode == nil {
		m.logger.Info("no active storenode")
		return nil
	}
	m.logger.Info("disconnecting active storenode", zap.String("nodeID", m.storenodeCycle.activeStorenode.ID))
	m.mailPeersMutex.Lock()
	pInfo, ok := m.storenodeCycle.peers[m.storenodeCycle.activeStorenode.ID]
	if ok {
		pInfo.status = disconnected

		pInfo.canConnectAfter = time.Now().Add(backoffDuration)
		m.storenodeCycle.peers[m.storenodeCycle.activeStorenode.ID] = pInfo
	} else {
		m.storenodeCycle.peers[m.storenodeCycle.activeStorenode.ID] = peerStatus{
			status:          disconnected,
			storenode:       *m.storenodeCycle.activeStorenode,
			canConnectAfter: time.Now().Add(backoffDuration),
		}
	}
	m.mailPeersMutex.Unlock()

	m.storenodeCycle.activeStorenode = nil
	return nil
}

func (m *Messenger) disconnectActiveStorenode(backoffDuration time.Duration) {
	err := m.disconnecStorenode(backoffDuration)
	if err != nil {
		m.logger.Error("failed to disconnect storenode", zap.Error(err))
	}
	signal.SendMailserverChanged(nil)
}

func (m *Messenger) cycleMailservers() {
	m.logger.Info("Automatically switching storenode")

	if m.storenodeCycle.activeStorenode != nil {
		m.disconnectActiveStorenode(graylistBackoff)
	}

	useMailserver, err := m.settings.CanUseMailservers()
	if err != nil {
		m.logger.Error("failed to get use mailservers", zap.Error(err))
		return
	}

	if !useMailserver {
		m.logger.Info("Skipping storenode search due to useMailserver being false")
		return
	}

	err = m.findNewStorenode()
	if err != nil {
		m.logger.Error("Error getting new storenode", zap.Error(err))
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

func (m *Messenger) allStorenodes() ([]mailservers.Mailserver, error) {
	// Get configured fleet
	fleet, err := m.getFleet()
	if err != nil {
		return nil, err
	}

	// Get default storenode for given fleet
	allStorenodes := m.storenodesByFleet(fleet)

	// Add custom configured storenode
	if m.mailserversDatabase != nil {
		customMailservers, err := m.mailserversDatabase.Mailservers()
		if err != nil {
			return nil, err
		}

		for _, c := range customMailservers {
			if c.Fleet == fleet {
				allStorenodes = append(allStorenodes, c)
			}
		}
	}

	return allStorenodes, nil
}

type SortedStorenodes struct {
	Mailserver      mailservers.Mailserver
	RTTMs           int
	CanConnectAfter time.Time
}

func (m *Messenger) findNewStorenode() error {

	// we have to override DNS manually because of https://github.com/status-im/status-mobile/issues/19581
	if overrideDNS {
		var dialer net.Dialer
		net.DefaultResolver = &net.Resolver{
			PreferGo: false,
			Dial: func(context context.Context, _, _ string) (net.Conn, error) {
				conn, err := dialer.DialContext(context, "udp", bootstrapDNS)
				if err != nil {
					return nil, err
				}
				return conn, nil
			},
		}
	}

	pinnedMailserver, err := m.getPinnedMailserver()
	if err != nil {
		m.logger.Error("Could not obtain the pinned mailserver", zap.Error(err))
		return err
	}
	if pinnedMailserver != nil {
		return m.connectToMailserver(*pinnedMailserver)
	}

	allStorenodes := m.storenodeCycle.allStorenodes

	//	TODO: remove this check once sockets are stable on x86_64 emulators
	if findNearestStorenode {
		m.logger.Info("Finding a new storenode...")

		if len(allStorenodes) == 0 {
			m.logger.Warn("no storenodes available") // Do nothing...
			return nil

		}

		pingResult, err := m.transport.PingPeer(m.ctx, allStorenodes, 500)
		if err != nil {
			// pinging storenodes might fail, but we don't care
			m.logger.Warn("ping failed with", zap.Error(err))
		}

		var availableStorenodes []*mailservers.PingResult
		for _, result := range pingResult {
			if result.Err != nil {
				m.logger.Info("connecting error", zap.String("err", *result.Err))
				continue // The results with error are ignored
			}
			availableStorenodes = append(availableStorenodes, result)
		}

		if len(availableStorenodes) == 0 {
			m.logger.Warn("No storenodes available") // Do nothing...
			return nil
		}

		mailserversByID := make(map[string]mailservers.Mailserver)
		for idx := range allStorenodes {
			mailserversByID[allStorenodes[idx].ID] = allStorenodes[idx]
		}
		var sortedMailservers []SortedStorenodes
		for _, ping := range availableStorenodes {
			ms := mailserversByID[ping.ID]
			sortedMailserver := SortedStorenodes{
				Mailserver: ms,
				RTTMs:      *ping.RTTMs,
			}
			m.mailPeersMutex.Lock()
			pInfo, ok := m.storenodeCycle.peers[ms.ID]
			m.mailPeersMutex.Unlock()
			if ok {
				if time.Now().Before(pInfo.canConnectAfter) {
					continue // We can't connect to this node yet
				}
			}

			sortedMailservers = append(sortedMailservers, sortedMailserver)

		}
		sort.Sort(byRTTMsAndCanConnectBefore(sortedMailservers))

		// Picks a random mailserver amongs the ones with the lowest latency
		// The pool size is 1/4 of the mailservers were pinged successfully
		pSize := poolSize(len(sortedMailservers) - 1)
		if pSize <= 0 {
			pSize = len(sortedMailservers)
			if pSize <= 0 {
				m.logger.Warn("No mailservers available") // Do nothing...
				return nil
			}
		}

		r, err := rand.Int(rand.Reader, big.NewInt(int64(pSize)))
		if err != nil {
			return err
		}

		msPing := sortedMailservers[r.Int64()]
		ms := mailserversByID[msPing.Mailserver.ID]
		m.logger.Info("connecting to mailserver", zap.String("address", ms.ID))
		return m.connectToMailserver(ms)
	}

	mailserversByID := make(map[string]mailservers.Mailserver)
	for idx := range allStorenodes {
		mailserversByID[allStorenodes[idx].ID] = allStorenodes[idx]
	}

	pSize := poolSize(len(allStorenodes) - 1)
	if pSize <= 0 {
		pSize = len(allStorenodes)
		if pSize <= 0 {
			m.logger.Warn("No mailservers available") // Do nothing...
			return nil
		}
	}

	r, err := rand.Int(rand.Reader, big.NewInt(int64(pSize)))
	if err != nil {
		return err
	}

	msPing := allStorenodes[r.Int64()]
	ms := mailserversByID[msPing.ID]
	m.logger.Info("connecting to mailserver", zap.String("address", ms.ID))

	return m.connectToMailserver(ms)
}

func (m *Messenger) mailserverStatus(mailserverID string) connStatus {
	m.mailPeersMutex.RLock()
	defer m.mailPeersMutex.RUnlock()
	peer, ok := m.storenodeCycle.peers[mailserverID]
	if !ok {
		return disconnected
	}
	return peer.status
}

func (m *Messenger) connectToMailserver(ms mailservers.Mailserver) error {

	m.logger.Info("connecting to mailserver", zap.Any("peer", ms.ID))

	m.storenodeCycle.activeStorenode = &ms
	signal.SendMailserverChanged(m.storenodeCycle.activeStorenode)

	activeMailserverStatus := m.mailserverStatus(ms.ID)
	if activeMailserverStatus != connected {
		m.mailPeersMutex.Lock()
		m.storenodeCycle.peers[ms.ID] = peerStatus{
			status:                connected,
			lastConnectionAttempt: time.Now(),
			canConnectAfter:       time.Now().Add(defaultBackoff),
			storenode:             ms,
		}
		m.mailPeersMutex.Unlock()

		m.storenodeCycle.activeStorenode.FailedRequests = 0
		peerID, err := m.storenodeCycle.activeStorenode.PeerID()
		if err != nil {
			m.logger.Error("could not decode the peer id of storenode", zap.Error(err))
			return err
		}

		m.logger.Info("storenode available", zap.String("storenodeID", m.storenodeCycle.activeStorenode.ID))
		m.EmitMailserverAvailable()
		signal.SendMailserverAvailable(m.storenodeCycle.activeStorenode)

		m.transport.SetStorePeerID(peerID)

		// Query mailserver
		m.asyncRequestAllHistoricMessages()
	}
	return nil
}

// getActiveMailserver returns the active mailserver if a communityID is present then it'll return the mailserver
// for that community if it has a mailserver setup otherwise it'll return the global mailserver
func (m *Messenger) getActiveMailserver(communityID ...string) *mailservers.Mailserver {
	if len(communityID) == 0 || communityID[0] == "" {
		return m.storenodeCycle.activeStorenode
	}
	ms, err := m.communityStorenodes.GetStorenodeByCommunnityID(communityID[0])
	if err != nil {
		if !errors.Is(err, storenodes.ErrNotFound) {
			m.logger.Error("getting storenode for community, using global", zap.String("communityID", communityID[0]), zap.Error(err))
		}
		// if we don't find a specific mailserver for the community, we just use the regular mailserverCycle's one
		return m.storenodeCycle.activeStorenode
	}
	return &ms
}

func (m *Messenger) getActiveMailserverID(communityID ...string) string {
	ms := m.getActiveMailserver(communityID...)
	if ms == nil {
		return ""
	}
	return ms.ID
}

func (m *Messenger) isMailserverAvailable(mailserverID string) bool {
	return m.mailserverStatus(mailserverID) == connected
}

func mailserverAddressToID(uniqueID string, allStorenodes []mailservers.Mailserver) (string, error) {
	for _, ms := range allStorenodes {
		if uniqueID == ms.ID {
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
	pInfo, ok := m.storenodeCycle.peers[id]
	if !ok {
		pInfo.status = disconnected
	}

	pInfo.canConnectAfter = time.Now().Add(graylistBackoff)
	m.storenodeCycle.peers[id] = pInfo
}

func (m *Messenger) asyncRequestAllHistoricMessages() {
	if !m.config.codeControlFlags.AutoRequestHistoricMessages {
		return
	}

	m.logger.Debug("asyncRequestAllHistoricMessages")

	go func() {
		_, err := m.RequestAllHistoricMessages(false, true)
		if err != nil {
			m.logger.Error("failed to request historic messages", zap.Error(err))
		}
	}()
}

func (m *Messenger) verifyStorenodeStatus() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := m.disconnectStorenodeIfRequired()
			if err != nil {
				m.logger.Error("failed to handle mailserver cycle event", zap.Error(err))
				continue
			}

		case <-m.quit:
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

	fleetMailservers := mailservers.DefaultStorenodes()

	for _, c := range fleetMailservers {
		if c.Fleet == fleet && c.ID == pinnedMailserver {
			return &c, nil
		}
	}

	if m.mailserversDatabase != nil {
		customMailservers, err := m.mailserversDatabase.Mailservers()
		if err != nil {
			return nil, err
		}

		for _, c := range customMailservers {
			if c.Fleet == fleet && c.ID == pinnedMailserver {
				return &c, nil
			}
		}
	}

	return nil, nil
}

func (m *Messenger) EmitMailserverAvailable() {
	for _, s := range m.storenodeCycle.availabilitySubscriptions {
		s <- struct{}{}
		close(s)
		l := len(m.storenodeCycle.availabilitySubscriptions)
		m.storenodeCycle.availabilitySubscriptions = m.storenodeCycle.availabilitySubscriptions[:l-1]
	}
}

func (m *Messenger) SubscribeMailserverAvailable() chan struct{} {
	c := make(chan struct{})
	m.storenodeCycle.availabilitySubscriptions = append(m.storenodeCycle.availabilitySubscriptions, c)
	return c
}

func (m *Messenger) disconnectStorenodeIfRequired() error {
	m.logger.Debug("wakuV2 storenode status verification")

	if m.storenodeCycle.activeStorenode == nil {
		// No active storenode, find a new one
		m.cycleMailservers()
		return nil
	}

	// Check whether we want to disconnect the active storenode
	if m.storenodeCycle.activeStorenode.FailedRequests >= mailserverMaxFailedRequests {
		m.penalizeMailserver(m.storenodeCycle.activeStorenode.ID)
		signal.SendMailserverNotWorking()
		m.logger.Info("too many failed requests", zap.String("storenode", m.storenodeCycle.activeStorenode.ID))
		m.storenodeCycle.activeStorenode.FailedRequests = 0
		return m.connectToNewMailserverAndWait()
	}

	return nil
}

func (m *Messenger) waitForAvailableStoreNode(timeout time.Duration) bool {
	// Add 1 second to timeout, because the storenode cycle has 1 second ticker, which doesn't tick on start.
	// This can be improved after merging https://github.com/status-im/status-go/pull/4380.
	// NOTE: https://stackoverflow.com/questions/32705582/how-to-get-time-tick-to-tick-immediately
	timeout += time.Second

	finish := make(chan struct{})
	cancel := make(chan struct{})

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer func() {
			wg.Done()
		}()
		for !m.isMailserverAvailable(m.getActiveMailserverID()) {
			select {
			case <-m.SubscribeMailserverAvailable():
			case <-cancel:
				return
			}
		}
	}()

	go func() {
		defer func() {
			close(finish)
		}()
		wg.Wait()
	}()

	select {
	case <-finish:
	case <-time.After(timeout):
		close(cancel)
	case <-m.ctx.Done():
		close(cancel)
	}

	return m.isMailserverAvailable(m.getActiveMailserverID())
}
