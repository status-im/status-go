package protocol

import (
	"context"
	"crypto/rand"
	"math"
	"math/big"
	"net"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/waku-org/go-waku/waku/v2/utils"

	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/storenodes"
	"github.com/status-im/status-go/services/mailservers"
	"github.com/status-im/status-go/signal"
)

const defaultBackoff = 10 * time.Second
const graylistBackoff = 3 * time.Minute
const backoffByUserAction = 0
const isAndroidEmulator = runtime.GOOS == "android" && runtime.GOARCH == "amd64"
const findNearestMailServer = !isAndroidEmulator
const overrideDNS = runtime.GOOS == "android" || runtime.GOOS == "ios"
const bootstrapDNS = "8.8.8.8:53"

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
		return s[i].RTT < s[j].RTT
	}
	return s[i].CanConnectAfter.Before(s[j].CanConnectAfter)
}

func (m *Messenger) StartMailserverCycle(mailservers []mailservers.Mailserver) error {
	if m.transport.WakuVersion() != 2 {
		m.logger.Warn("not starting mailserver cycle: requires wakuv2")
		return nil
	}

	m.mailserverCycle.allMailservers = mailservers

	if len(mailservers) == 0 {
		m.logger.Warn("not starting mailserver cycle: empty mailservers list")
		return nil
	}

	for _, storenode := range mailservers {

		peerInfo, err := storenode.PeerInfo()
		if err != nil {
			return err
		}

		for _, addr := range utils.EncapsulatePeerID(peerInfo.ID, peerInfo.Addrs...) {
			_, err := m.transport.AddStorePeer(addr)
			if err != nil {
				return err
			}
		}
	}
	go m.verifyStorenodeStatus()

	m.logger.Debug("starting mailserver cycle",
		zap.Uint("WakuVersion", m.transport.WakuVersion()),
		zap.Any("mailservers", mailservers),
	)

	return nil
}

func (m *Messenger) DisconnectActiveMailserver() {
	m.mailserverCycle.Lock()
	defer m.mailserverCycle.Unlock()
	m.disconnectActiveMailserver(graylistBackoff)
}

func (m *Messenger) disconnectMailserver(backoffDuration time.Duration) error {
	if m.mailserverCycle.activeMailserver == nil {
		m.logger.Info("no active mailserver")
		return nil
	}
	m.logger.Info("disconnecting active mailserver", zap.String("nodeID", m.mailserverCycle.activeMailserver.ID))
	m.mailPeersMutex.Lock()
	pInfo, ok := m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID]
	if ok {
		pInfo.status = disconnected

		pInfo.canConnectAfter = time.Now().Add(backoffDuration)
		m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID] = pInfo
	} else {
		m.mailserverCycle.peers[m.mailserverCycle.activeMailserver.ID] = peerStatus{
			status:          disconnected,
			mailserver:      *m.mailserverCycle.activeMailserver,
			canConnectAfter: time.Now().Add(backoffDuration),
		}
	}
	m.mailPeersMutex.Unlock()

	m.mailserverCycle.activeMailserver = nil
	return nil
}

func (m *Messenger) disconnectActiveMailserver(backoffDuration time.Duration) {
	err := m.disconnectMailserver(backoffDuration)
	if err != nil {
		m.logger.Error("failed to disconnect mailserver", zap.Error(err))
	}
	signal.SendMailserverChanged(nil)
}

func (m *Messenger) cycleMailservers() {
	m.logger.Info("Automatically switching mailserver")

	if m.mailserverCycle.activeMailserver != nil {
		m.disconnectActiveMailserver(graylistBackoff)
	}

	useMailserver, err := m.settings.CanUseMailservers()
	if err != nil {
		m.logger.Error("failed to get use mailservers", zap.Error(err))
		return
	}

	if !useMailserver {
		m.logger.Info("Skipping mailserver search due to useMailserver being false")
		return
	}

	err = m.findNewMailserver()
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
		fleet = params.FleetStatusProd
	}
	return fleet, nil
}

func (m *Messenger) allMailservers() ([]mailservers.Mailserver, error) {
	// Get configured fleet
	fleet, err := m.getFleet()
	if err != nil {
		return nil, err
	}

	// Get default mailservers for given fleet
	allMailservers := mailservers.DefaultMailserversByFleet(fleet)

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

type SortedMailserver struct {
	Mailserver      mailservers.Mailserver
	RTT             time.Duration
	CanConnectAfter time.Time
}

func (m *Messenger) getAvailableMailserversSortedByRTT(allMailservers []mailservers.Mailserver) []mailservers.Mailserver {
	// TODO: this can be replaced by peer selector once code is moved to go-waku api
	availableMailservers := make(map[string]time.Duration)
	availableMailserversMutex := sync.Mutex{}
	availableMailserversWg := sync.WaitGroup{}
	for _, mailserver := range allMailservers {
		availableMailserversWg.Add(1)
		go func(mailserver mailservers.Mailserver) {
			defer availableMailserversWg.Done()

			peerID, err := mailserver.PeerID()
			if err != nil {
				return
			}

			ctx, cancel := context.WithTimeout(m.ctx, 4*time.Second)
			defer cancel()

			rtt, err := m.transport.PingPeer(ctx, peerID)
			if err == nil { // pinging mailservers might fail, but we don't care
				availableMailserversMutex.Lock()
				availableMailservers[mailserver.ID] = rtt
				availableMailserversMutex.Unlock()
			}
		}(mailserver)
	}
	availableMailserversWg.Wait()

	if len(availableMailservers) == 0 {
		m.logger.Warn("No mailservers available") // Do nothing...
		return nil
	}

	mailserversByID := make(map[string]mailservers.Mailserver)
	for idx := range allMailservers {
		mailserversByID[allMailservers[idx].ID] = allMailservers[idx]
	}
	var sortedMailservers []SortedMailserver
	for mailserverID, rtt := range availableMailservers {
		ms := mailserversByID[mailserverID]
		sortedMailserver := SortedMailserver{
			Mailserver: ms,
			RTT:        rtt,
		}
		m.mailPeersMutex.Lock()
		pInfo, ok := m.mailserverCycle.peers[ms.ID]
		m.mailPeersMutex.Unlock()
		if ok {
			if time.Now().Before(pInfo.canConnectAfter) {
				continue // We can't connect to this node yet
			}
		}
		sortedMailservers = append(sortedMailservers, sortedMailserver)
	}
	sort.Sort(byRTTMsAndCanConnectBefore(sortedMailservers))

	result := make([]mailservers.Mailserver, len(sortedMailservers))
	for i, s := range sortedMailservers {
		result[i] = s.Mailserver
	}

	return result
}

func (m *Messenger) findNewMailserver() error {
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

	m.logger.Info("Finding a new mailserver...")

	allMailservers := m.mailserverCycle.allMailservers

	//	TODO: remove this check once sockets are stable on x86_64 emulators
	if findNearestMailServer {
		allMailservers = m.getAvailableMailserversSortedByRTT(allMailservers)
	}

	// Picks a random mailserver amongs the ones with the lowest latency
	// The pool size is 1/4 of the mailservers were pinged successfully
	pSize := poolSize(len(allMailservers) - 1)
	if pSize <= 0 {
		pSize = len(allMailservers)
		if pSize <= 0 {
			m.logger.Warn("No storenodes available") // Do nothing...
			return nil
		}
	}

	r, err := rand.Int(rand.Reader, big.NewInt(int64(pSize)))
	if err != nil {
		return err
	}

	ms := allMailservers[r.Int64()]
	return m.connectToMailserver(ms)
}

func (m *Messenger) mailserverStatus(mailserverID string) connStatus {
	m.mailPeersMutex.RLock()
	defer m.mailPeersMutex.RUnlock()
	peer, ok := m.mailserverCycle.peers[mailserverID]
	if !ok {
		return disconnected
	}
	return peer.status
}

func (m *Messenger) connectToMailserver(ms mailservers.Mailserver) error {

	m.logger.Info("connecting to mailserver", zap.String("mailserverID", ms.ID))

	m.mailserverCycle.activeMailserver = &ms
	signal.SendMailserverChanged(m.mailserverCycle.activeMailserver)

	mailserverStatus := m.mailserverStatus(ms.ID)
	if mailserverStatus != connected {
		m.mailPeersMutex.Lock()
		m.mailserverCycle.peers[ms.ID] = peerStatus{
			status:                connected,
			lastConnectionAttempt: time.Now(),
			canConnectAfter:       time.Now().Add(defaultBackoff),
			mailserver:            ms,
		}
		m.mailPeersMutex.Unlock()

		m.mailserverCycle.activeMailserver.FailedRequests = 0
		peerID, err := m.mailserverCycle.activeMailserver.PeerID()
		if err != nil {
			m.logger.Error("could not decode the peer id of mailserver", zap.Error(err))
			return err
		}

		m.logger.Info("mailserver available", zap.String("mailserverID", m.mailserverCycle.activeMailserver.ID))
		m.mailserverCycle.availabilitySubscriptions.EmitMailserverAvailable()
		signal.SendMailserverAvailable(m.mailserverCycle.activeMailserver)

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
		return m.mailserverCycle.activeMailserver
	}
	ms, err := m.communityStorenodes.GetStorenodeByCommunityID(communityID[0])
	if err != nil {
		if !errors.Is(err, storenodes.ErrNotFound) {
			m.logger.Error("getting storenode for community, using global", zap.String("communityID", communityID[0]), zap.Error(err))
		}
		// if we don't find a specific mailserver for the community, we just use the regular mailserverCycle's one
		return m.mailserverCycle.activeMailserver
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

	fleetMailservers := mailservers.DefaultMailservers()

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

func (m *Messenger) disconnectStorenodeIfRequired() error {
	m.logger.Debug("wakuV2 storenode status verification")

	if m.mailserverCycle.activeMailserver == nil {
		// No active storenode, find a new one
		m.cycleMailservers()
		return nil
	}

	// Check whether we want to disconnect the active storenode
	if m.mailserverCycle.activeMailserver.FailedRequests >= mailserverMaxFailedRequests {
		m.penalizeMailserver(m.mailserverCycle.activeMailserver.ID)
		signal.SendMailserverNotWorking()
		m.logger.Info("too many failed requests", zap.String("storenode", m.mailserverCycle.activeMailserver.ID))
		m.mailserverCycle.activeMailserver.FailedRequests = 0
		return m.connectToNewMailserverAndWait()
	}

	return nil
}

func (m *Messenger) waitForAvailableStoreNode(timeout time.Duration) bool {
	// Add 1 second to timeout, because the mailserver cycle has 1 second ticker, which doesn't tick on start.
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
			case <-m.mailserverCycle.availabilitySubscriptions.Subscribe():
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
