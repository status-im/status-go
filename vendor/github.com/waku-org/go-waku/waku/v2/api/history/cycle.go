package history

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"net/http"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/api/common"
	"github.com/waku-org/go-waku/waku/v2/protocol/store"
	"go.uber.org/zap"
)

const defaultBackoff = 10 * time.Second
const graylistBackoff = 3 * time.Minute
const storenodeVerificationInterval = time.Second
const storenodeMaxFailedRequests uint = 2
const minStorenodesToChooseFrom = 3
const isAndroidEmulator = runtime.GOOS == "android" && runtime.GOARCH == "amd64"
const findNearestMailServer = !isAndroidEmulator
const overrideDNS = runtime.GOOS == "android" || runtime.GOOS == "ios"
const bootstrapDNS = "8.8.8.8:53"

type connStatus int

const (
	disconnected connStatus = iota + 1
	connected
)

type peerStatus struct {
	status                connStatus
	canConnectAfter       time.Time
	lastConnectionAttempt time.Time
}

type StorenodeConfigProvider interface {
	UseStorenodes() (bool, error)
	GetPinnedStorenode() (peer.AddrInfo, error)
	Storenodes() ([]peer.AddrInfo, error)
}

type StorenodeCycle struct {
	sync.RWMutex

	logger *zap.Logger

	storenodeConfigProvider StorenodeConfigProvider
	pinger                  common.Pinger

	StorenodeAvailableOneshotEmitter *OneShotEmitter[struct{}]
	StorenodeChangedEmitter          *Emitter[peer.ID]
	StorenodeNotWorkingEmitter       *Emitter[struct{}]
	StorenodeAvailableEmitter        *Emitter[peer.ID]

	failedRequests map[peer.ID]uint

	peersMutex      sync.RWMutex
	activeStorenode peer.ID
	peers           map[peer.ID]peerStatus
}

func NewStorenodeCycle(logger *zap.Logger, pinger common.Pinger) *StorenodeCycle {
	return &StorenodeCycle{
		StorenodeAvailableOneshotEmitter: NewOneshotEmitter[struct{}](),
		StorenodeChangedEmitter:          NewEmitter[peer.ID](),
		StorenodeNotWorkingEmitter:       NewEmitter[struct{}](),
		StorenodeAvailableEmitter:        NewEmitter[peer.ID](),
		pinger:                           pinger,
		logger:                           logger.Named("storenode-cycle"),
	}
}

func (m *StorenodeCycle) Start(ctx context.Context) {
	m.logger.Debug("starting storenode cycle")
	m.failedRequests = make(map[peer.ID]uint)
	m.peers = make(map[peer.ID]peerStatus)

	go m.verifyStorenodeStatus(ctx)
}

func (m *StorenodeCycle) DisconnectActiveStorenode(backoff time.Duration) {
	m.Lock()
	defer m.Unlock()

	m.disconnectActiveStorenode(backoff)
}

func (m *StorenodeCycle) connectToNewStorenodeAndWait(ctx context.Context) error {
	// Handle pinned storenodes
	m.logger.Info("disconnecting storenode")
	pinnedStorenode, err := m.storenodeConfigProvider.GetPinnedStorenode()
	if err != nil {
		m.logger.Error("could not obtain the pinned storenode", zap.Error(err))
		return err
	}

	// If no pinned storenode, no need to disconnect and wait for it to be available
	if pinnedStorenode.ID == "" {
		m.disconnectActiveStorenode(graylistBackoff)
	}

	return m.findNewStorenode(ctx)
}

func (m *StorenodeCycle) disconnectStorenode(backoffDuration time.Duration) error {
	if m.activeStorenode == "" {
		m.logger.Info("no active storenode")
		return nil
	}

	m.logger.Info("disconnecting active storenode", zap.Stringer("peerID", m.activeStorenode))

	m.peersMutex.Lock()
	pInfo, ok := m.peers[m.activeStorenode]
	if ok {
		pInfo.status = disconnected
		pInfo.canConnectAfter = time.Now().Add(backoffDuration)
		m.peers[m.activeStorenode] = pInfo
	} else {
		m.peers[m.activeStorenode] = peerStatus{
			status:          disconnected,
			canConnectAfter: time.Now().Add(backoffDuration),
		}
	}
	m.peersMutex.Unlock()

	m.activeStorenode = ""

	return nil
}

func (m *StorenodeCycle) disconnectActiveStorenode(backoffDuration time.Duration) {
	err := m.disconnectStorenode(backoffDuration)
	if err != nil {
		m.logger.Error("failed to disconnect storenode", zap.Error(err))
	}

	m.StorenodeChangedEmitter.Emit("")
}

func (m *StorenodeCycle) Cycle(ctx context.Context) {
	if m.storenodeConfigProvider == nil {
		m.logger.Debug("storenodeConfigProvider not yet setup")
		return
	}

	m.logger.Info("Automatically switching storenode")

	if m.activeStorenode != "" {
		m.disconnectActiveStorenode(graylistBackoff)
	}

	useStorenode, err := m.storenodeConfigProvider.UseStorenodes()
	if err != nil {
		m.logger.Error("failed to get use storenodes", zap.Error(err))
		return
	}

	if !useStorenode {
		m.logger.Info("Skipping storenode search due to useStorenode being false")
		return
	}

	err = m.findNewStorenode(ctx)
	if err != nil {
		m.logger.Error("Error getting new storenode", zap.Error(err))
	}
}

func poolSize(fleetSize int) int {
	return int(math.Ceil(float64(fleetSize) / 4))
}

func (m *StorenodeCycle) getAvailableStorenodesSortedByRTT(ctx context.Context, allStorenodes []peer.AddrInfo) []peer.AddrInfo {
	peerIDToInfo := make(map[peer.ID]peer.AddrInfo)
	for _, p := range allStorenodes {
		peerIDToInfo[p.ID] = p
	}

	availableStorenodes := make(map[peer.ID]time.Duration)
	availableStorenodesMutex := sync.Mutex{}
	availableStorenodesWg := sync.WaitGroup{}
	for _, storenode := range allStorenodes {
		availableStorenodesWg.Add(1)
		go func(peerInfo peer.AddrInfo) {
			defer availableStorenodesWg.Done()
			ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()

			rtt, err := m.pinger.PingPeer(ctx, peerInfo)
			if err == nil { // pinging storenodes might fail, but we don't care
				availableStorenodesMutex.Lock()
				availableStorenodes[peerInfo.ID] = rtt
				availableStorenodesMutex.Unlock()
			}
		}(storenode)
	}
	availableStorenodesWg.Wait()

	if len(availableStorenodes) == 0 {
		m.logger.Warn("No storenodes available") // Do nothing..
		return nil
	}

	var sortedStorenodes []SortedStorenode
	for storenodeID, rtt := range availableStorenodes {
		sortedStorenode := SortedStorenode{
			Storenode: peerIDToInfo[storenodeID],
			RTT:       rtt,
		}
		m.peersMutex.Lock()
		pInfo, ok := m.peers[storenodeID]
		m.peersMutex.Unlock()
		if ok && time.Now().Before(pInfo.canConnectAfter) {
			continue // We can't connect to this node yet
		}
		sortedStorenodes = append(sortedStorenodes, sortedStorenode)
	}
	sort.Sort(byRTTMsAndCanConnectBefore(sortedStorenodes))

	result := make([]peer.AddrInfo, len(sortedStorenodes))
	for i, s := range sortedStorenodes {
		result[i] = s.Storenode
	}

	return result
}

func (m *StorenodeCycle) findNewStorenode(ctx context.Context) error {
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

	pinnedStorenode, err := m.storenodeConfigProvider.GetPinnedStorenode()
	if err != nil {
		m.logger.Error("Could not obtain the pinned storenode", zap.Error(err))
		return err
	}

	if pinnedStorenode.ID != "" {
		return m.setActiveStorenode(pinnedStorenode.ID)
	}

	m.logger.Info("Finding a new storenode..")

	allStorenodes, err := m.storenodeConfigProvider.Storenodes()
	if err != nil {
		return err
	}

	//	TODO: remove this check once sockets are stable on x86_64 emulators
	if findNearestMailServer {
		allStorenodes = m.getAvailableStorenodesSortedByRTT(ctx, allStorenodes)
	}

	// Picks a random storenode amongs the ones with the lowest latency
	// The pool size is 1/4 of the storenodes were pinged successfully
	// If the pool size is less than `minStorenodesToChooseFrom`, it will
	// pick a storenode fromm all the available storenodes
	pSize := poolSize(len(allStorenodes) - 1)
	if pSize <= minStorenodesToChooseFrom {
		pSize = len(allStorenodes)
		if pSize <= 0 {
			m.logger.Warn("No storenodes available") // Do nothing..
			return nil
		}
	}

	r, err := rand.Int(rand.Reader, big.NewInt(int64(pSize)))
	if err != nil {
		return err
	}

	ms := allStorenodes[r.Int64()]
	return m.setActiveStorenode(ms.ID)
}

func (m *StorenodeCycle) storenodeStatus(peerID peer.ID) connStatus {
	m.peersMutex.RLock()
	defer m.peersMutex.RUnlock()

	peer, ok := m.peers[peerID]
	if !ok {
		return disconnected
	}
	return peer.status
}

func (m *StorenodeCycle) setActiveStorenode(peerID peer.ID) error {
	m.activeStorenode = peerID

	m.StorenodeChangedEmitter.Emit(m.activeStorenode)

	storenodeStatus := m.storenodeStatus(peerID)
	if storenodeStatus != connected {
		m.peersMutex.Lock()
		m.peers[peerID] = peerStatus{
			status:                connected,
			lastConnectionAttempt: time.Now(),
			canConnectAfter:       time.Now().Add(defaultBackoff),
		}
		m.peersMutex.Unlock()

		m.failedRequests[peerID] = 0
		m.logger.Info("storenode available", zap.Stringer("peerID", m.activeStorenode))

		m.StorenodeAvailableOneshotEmitter.Emit(struct{}{}) // Maybe can be refactored away?
		m.StorenodeAvailableEmitter.Emit(m.activeStorenode)
	}
	return nil
}

func (m *StorenodeCycle) GetActiveStorenode() peer.ID {
	m.RLock()
	defer m.RUnlock()

	return m.activeStorenode
}

func (m *StorenodeCycle) IsStorenodeAvailable(peerID peer.ID) bool {
	return m.storenodeStatus(peerID) == connected
}

func (m *StorenodeCycle) penalizeStorenode(id peer.ID) {
	m.peersMutex.Lock()
	defer m.peersMutex.Unlock()
	pInfo, ok := m.peers[id]
	if !ok {
		pInfo.status = disconnected
	}

	pInfo.canConnectAfter = time.Now().Add(graylistBackoff)
	m.peers[id] = pInfo
}

func (m *StorenodeCycle) verifyStorenodeStatus(ctx context.Context) {
	ticker := time.NewTicker(storenodeVerificationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := m.disconnectStorenodeIfRequired(ctx)
			if err != nil {
				m.logger.Error("failed to handle storenode cycle event", zap.Error(err))
				continue
			}

		case <-ctx.Done():
			return
		}
	}
}

func (m *StorenodeCycle) disconnectStorenodeIfRequired(ctx context.Context) error {
	m.logger.Debug("wakuV2 storenode status verification")

	if m.activeStorenode == "" {
		// No active storenode, find a new one
		m.Cycle(ctx)
		return nil
	}

	// Check whether we want to disconnect the active storenode
	if m.failedRequests[m.activeStorenode] >= storenodeMaxFailedRequests {
		m.penalizeStorenode(m.activeStorenode)
		m.StorenodeNotWorkingEmitter.Emit(struct{}{})

		m.logger.Info("too many failed requests", zap.Stringer("storenode", m.activeStorenode))
		m.failedRequests[m.activeStorenode] = 0
		return m.connectToNewStorenodeAndWait(ctx)
	}

	return nil
}

func (m *StorenodeCycle) SetStorenodeConfigProvider(provider StorenodeConfigProvider) {
	m.storenodeConfigProvider = provider
}

func (m *StorenodeCycle) WaitForAvailableStoreNode(ctx context.Context) bool {
	// Note: Add 1 second to timeout, because the storenode cycle has 1 second ticker, which doesn't tick on start.
	// This can be improved after merging https://github.com/status-im/status-go/pull/4380.
	// NOTE: https://stackoverflow.com/questions/32705582/how-to-get-time-tick-to-tick-immediately

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !m.IsStorenodeAvailable(m.activeStorenode) {
			select {
			case <-m.StorenodeAvailableOneshotEmitter.Subscribe():
			case <-ctx.Done():
				if errors.Is(ctx.Err(), context.Canceled) {
					return
				}

				// Wait for an additional second, but handle cancellation
				select {
				case <-time.After(1 * time.Second):
				case <-ctx.Done(): // context was cancelled
				}

				return

			}
		}
	}()

	select {
	case <-waitForWaitGroup(&wg):
	case <-ctx.Done():
		// Wait for an additional second, but handle cancellation
		select {
		case <-time.After(1 * time.Second):
		case <-ctx.Done(): // context was cancelled o
		}
	}

	return m.IsStorenodeAvailable(m.activeStorenode)
}

func waitForWaitGroup(wg *sync.WaitGroup) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()
	return ch
}

type storenodeTaskParameters struct {
	customPeerID peer.ID
}

type StorenodeTaskOption func(*storenodeTaskParameters)

func WithPeerID(peerID peer.ID) StorenodeTaskOption {
	return func(stp *storenodeTaskParameters) {
		stp.customPeerID = peerID
	}
}

func (m *StorenodeCycle) PerformStorenodeTask(fn func() error, options ...StorenodeTaskOption) error {
	params := storenodeTaskParameters{}
	for _, opt := range options {
		opt(&params)
	}

	peerID := params.customPeerID
	if peerID == "" {
		peerID = m.GetActiveStorenode()
	}

	if peerID == "" {
		return errors.New("storenode not available")
	}

	m.RLock()
	defer m.RUnlock()

	var tries uint = 0
	for tries < storenodeMaxFailedRequests {
		if params.customPeerID == "" && m.storenodeStatus(peerID) != connected {
			return errors.New("storenode not available")
		}
		m.logger.Info("trying performing history requests", zap.Uint("try", tries), zap.Stringer("peerID", peerID))

		// Peform request
		err := fn()
		if err == nil {
			// Reset failed requests
			m.logger.Debug("history request performed successfully", zap.Stringer("peerID", peerID))
			m.failedRequests[peerID] = 0
			return nil
		}

		m.logger.Error("failed to perform history request",
			zap.Stringer("peerID", peerID),
			zap.Uint("tries", tries),
			zap.Error(err),
		)

		tries++

		if storeErr, ok := err.(*store.StoreError); ok {
			if storeErr.Code == http.StatusTooManyRequests {
				m.disconnectActiveStorenode(defaultBackoff)
				return fmt.Errorf("ratelimited at storenode %s: %w", peerID, err)
			}
		}

		// Increment failed requests
		m.failedRequests[peerID]++

		// Change storenode
		if m.failedRequests[peerID] >= storenodeMaxFailedRequests {
			return errors.New("too many failed requests")
		}
		// Wait a couple of second not to spam
		time.Sleep(2 * time.Second)

	}
	return errors.New("failed to perform history request")
}
