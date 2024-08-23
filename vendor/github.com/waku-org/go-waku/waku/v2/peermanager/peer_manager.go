package peermanager

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enr"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/discv5"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/metadata"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/service"

	"go.uber.org/zap"
)

type TopicHealth int

const (
	UnHealthy           = iota
	MinimallyHealthy    = 1
	SufficientlyHealthy = 2
)

func (t TopicHealth) String() string {
	switch t {
	case UnHealthy:
		return "UnHealthy"
	case MinimallyHealthy:
		return "MinimallyHealthy"
	case SufficientlyHealthy:
		return "SufficientlyHealthy"
	default:
		return ""
	}
}

type TopicHealthStatus struct {
	Topic  string
	Health TopicHealth
}

// NodeTopicDetails stores pubSubTopic related data like topicHandle for the node.
type NodeTopicDetails struct {
	topic        *pubsub.Topic
	healthStatus TopicHealth
}

// WakuProtoInfo holds protocol specific info
// To be used at a later stage to set various config such as criteria for peer management specific to each Waku protocols
// This should make peer-manager agnostic to protocol
type WakuProtoInfo struct {
	waku2ENRBitField uint8
}

// PeerManager applies various controls and manage connections towards peers.
type PeerManager struct {
	peerConnector          *PeerConnectionStrategy
	metadata               *metadata.WakuMetadata
	relay                  *relay.WakuRelay
	maxPeers               int
	maxRelayPeers          int
	logger                 *zap.Logger
	InPeersTarget          int
	OutPeersTarget         int
	host                   host.Host
	serviceSlots           *ServiceSlots
	ctx                    context.Context
	sub                    event.Subscription
	topicMutex             sync.RWMutex
	subRelayTopics         map[string]*NodeTopicDetails
	discoveryService       *discv5.DiscoveryV5
	wakuprotoToENRFieldMap map[protocol.ID]WakuProtoInfo
	TopicHealthNotifCh     chan<- TopicHealthStatus
	rttCache               *FastestPeerSelector
	RelayEnabled           bool
}

// PeerSelection provides various options based on which Peer is selected from a list of peers.
type PeerSelection int

const (
	Automatic PeerSelection = iota
	LowestRTT
)

// ErrNoPeersAvailable is emitted when no suitable peers are found for
// some protocol
var ErrNoPeersAvailable = errors.New("no suitable peers found")

const maxFailedAttempts = 5
const prunePeerStoreInterval = 10 * time.Minute
const peerConnectivityLoopSecs = 15
const maxConnsToPeerRatio = 5

// 80% relay peers 20% service peers
func relayAndServicePeers(maxConnections int) (int, int) {
	return maxConnections - maxConnections/5, maxConnections / 5
}

// 66% inRelayPeers 33% outRelayPeers
func inAndOutRelayPeers(relayPeers int) (int, int) {
	outRelayPeers := relayPeers / 3
	//
	const minOutRelayConns = 10
	if outRelayPeers < minOutRelayConns {
		outRelayPeers = minOutRelayConns
	}
	return relayPeers - outRelayPeers, outRelayPeers
}

// checkAndUpdateTopicHealth finds health of specified topic and updates and notifies of the same.
// Also returns the healthyPeerCount
func (pm *PeerManager) checkAndUpdateTopicHealth(topic *NodeTopicDetails) int {
	if topic == nil {
		return 0
	}

	healthyPeerCount := 0

	for _, p := range pm.relay.PubSub().MeshPeers(topic.topic.String()) {
		if pm.host.Network().Connectedness(p) == network.Connected {
			pThreshold, err := pm.host.Peerstore().(wps.WakuPeerstore).Score(p)
			if err == nil {
				if pThreshold < relay.PeerPublishThreshold {
					pm.logger.Debug("peer score below publish threshold", zap.Stringer("peer", p), zap.Float64("score", pThreshold))
				} else {
					healthyPeerCount++
				}
			} else {
				if errors.Is(err, peerstore.ErrNotFound) {
					// For now considering peer as healthy if we can't fetch score.
					healthyPeerCount++
					pm.logger.Debug("peer score is not available yet", zap.Stringer("peer", p))
				} else {
					pm.logger.Warn("failed to fetch peer score ", zap.Error(err), zap.Stringer("peer", p))
				}
			}
		}
	}

	//Update topic's health
	oldHealth := topic.healthStatus
	if healthyPeerCount < 1 { //Ideally this check should be done with minPeersForRelay, but leaving it as is for now.
		topic.healthStatus = UnHealthy
	} else if healthyPeerCount < waku_proto.GossipSubDMin {
		topic.healthStatus = MinimallyHealthy
	} else {
		topic.healthStatus = SufficientlyHealthy
	}

	if oldHealth != topic.healthStatus {
		//Check old health, and if there is a change notify of the same.
		pm.logger.Debug("topic health has changed", zap.String("pubsubtopic", topic.topic.String()), zap.Stringer("health", topic.healthStatus))
		pm.TopicHealthNotifCh <- TopicHealthStatus{topic.topic.String(), topic.healthStatus}
	}
	return healthyPeerCount
}

// TopicHealth can be used to fetch health of a specific pubsubTopic.
// Returns error if topic is not found.
func (pm *PeerManager) TopicHealth(pubsubTopic string) (TopicHealth, error) {
	pm.topicMutex.RLock()
	defer pm.topicMutex.RUnlock()

	topicDetails, ok := pm.subRelayTopics[pubsubTopic]
	if !ok {
		return UnHealthy, errors.New("topic not found")
	}
	return topicDetails.healthStatus, nil
}

// NewPeerManager creates a new peerManager instance.
func NewPeerManager(maxConnections int, maxPeers int, metadata *metadata.WakuMetadata, relay *relay.WakuRelay, relayEnabled bool, logger *zap.Logger) *PeerManager {
	var inPeersTarget, outPeersTarget, maxRelayPeers int
	if relayEnabled {
		maxRelayPeers, _ := relayAndServicePeers(maxConnections)
		inPeersTarget, outPeersTarget = inAndOutRelayPeers(maxRelayPeers)

		if maxPeers == 0 || maxConnections > maxPeers {
			maxPeers = maxConnsToPeerRatio * maxConnections
		}
	} else {
		maxRelayPeers = 0
		inPeersTarget = 0
		//TODO: ideally this should be 2 filter peers per topic, 2 lightpush peers per topic and 2-4 store nodes per topic
		outPeersTarget = 10
	}
	pm := &PeerManager{
		logger:                 logger.Named("peer-manager"),
		metadata:               metadata,
		relay:                  relay,
		maxRelayPeers:          maxRelayPeers,
		InPeersTarget:          inPeersTarget,
		OutPeersTarget:         outPeersTarget,
		serviceSlots:           NewServiceSlot(),
		subRelayTopics:         make(map[string]*NodeTopicDetails),
		maxPeers:               maxPeers,
		wakuprotoToENRFieldMap: map[protocol.ID]WakuProtoInfo{},
		rttCache:               NewFastestPeerSelector(logger),
		RelayEnabled:           relayEnabled,
	}
	logger.Info("PeerManager init values", zap.Int("maxConnections", maxConnections),
		zap.Int("maxRelayPeers", maxRelayPeers),
		zap.Int("outPeersTarget", outPeersTarget),
		zap.Int("inPeersTarget", pm.InPeersTarget),
		zap.Int("maxPeers", maxPeers))

	return pm
}

// SetDiscv5 sets the discoveryv5 service to be used for peer discovery.
func (pm *PeerManager) SetDiscv5(discv5 *discv5.DiscoveryV5) {
	pm.discoveryService = discv5
}

// SetHost sets the host to be used in order to access the peerStore.
func (pm *PeerManager) SetHost(host host.Host) {
	pm.host = host
	pm.rttCache.SetHost(host)
}

// SetPeerConnector sets the peer connector to be used for establishing relay connections.
func (pm *PeerManager) SetPeerConnector(pc *PeerConnectionStrategy) {
	pm.peerConnector = pc
}

// Start starts the processing to be done by peer manager.
func (pm *PeerManager) Start(ctx context.Context) {
	pm.ctx = ctx
	if pm.RelayEnabled {
		pm.RegisterWakuProtocol(relay.WakuRelayID_v200, relay.WakuRelayENRField)
		if pm.sub != nil {
			go pm.peerEventLoop(ctx)
		}
		go pm.connectivityLoop(ctx)
	}
	go pm.peerStoreLoop(ctx)
}

func (pm *PeerManager) peerStoreLoop(ctx context.Context) {
	t := time.NewTicker(prunePeerStoreInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pm.prunePeerStore()
		}
	}
}

func (pm *PeerManager) prunePeerStore() {
	peers := pm.host.Peerstore().Peers()
	numPeers := len(peers)
	if numPeers < pm.maxPeers {
		pm.logger.Debug("peerstore size within capacity, not pruning", zap.Int("capacity", pm.maxPeers), zap.Int("numPeers", numPeers))
		return
	}
	peerCntBeforePruning := numPeers
	pm.logger.Debug("peerstore capacity exceeded, hence pruning", zap.Int("capacity", pm.maxPeers), zap.Int("numPeers", peerCntBeforePruning))

	for _, peerID := range peers {
		connFailues := pm.host.Peerstore().(wps.WakuPeerstore).ConnFailures(peerID)
		if connFailues > maxFailedAttempts {
			// safety check so that we don't end up disconnecting connected peers.
			if pm.host.Network().Connectedness(peerID) == network.Connected {
				pm.host.Peerstore().(wps.WakuPeerstore).ResetConnFailures(peerID)
				continue
			}
			pm.host.Peerstore().RemovePeer(peerID)
			numPeers--
		}
		if numPeers < pm.maxPeers {
			pm.logger.Debug("finished pruning peer store", zap.Int("capacity", pm.maxPeers), zap.Int("beforeNumPeers", peerCntBeforePruning), zap.Int("afterNumPeers", numPeers))
			return
		}
	}

	notConnectedPeers := pm.getPeersBasedOnconnectionStatus("", network.NotConnected)
	peersByTopic := make(map[string]peer.IDSlice)
	var prunedPeers peer.IDSlice

	//prune not connected peers without shard
	for _, peerID := range notConnectedPeers {
		topics, err := pm.host.Peerstore().(wps.WakuPeerstore).PubSubTopics(peerID)
		//Prune peers without pubsubtopics.
		if err != nil || len(topics) == 0 {
			if err != nil {
				pm.logger.Error("pruning:failed to fetch pubsub topics", zap.Error(err), zap.Stringer("peer", peerID))
			}
			prunedPeers = append(prunedPeers, peerID)
			pm.host.Peerstore().RemovePeer(peerID)
			numPeers--
		} else {
			prunedPeers = append(prunedPeers, peerID)
			for topic := range topics {
				peersByTopic[topic] = append(peersByTopic[topic], peerID)
			}
		}
		if numPeers < pm.maxPeers {
			pm.logger.Debug("finished pruning peer store", zap.Int("capacity", pm.maxPeers), zap.Int("beforeNumPeers", peerCntBeforePruning), zap.Int("afterNumPeers", numPeers), zap.Stringers("prunedPeers", prunedPeers))
			return
		}
	}
	pm.logger.Debug("pruned notconnected peers", zap.Stringers("prunedPeers", prunedPeers))

	// calculate the avg peers per shard
	total, maxPeerCnt := 0, 0
	for _, peersInTopic := range peersByTopic {
		peerLen := len(peersInTopic)
		total += peerLen
		if peerLen > maxPeerCnt {
			maxPeerCnt = peerLen
		}
	}
	avgPerTopic := min(1, total/maxPeerCnt)
	// prune peers from shard with higher than avg count

	for topic, peers := range peersByTopic {
		count := max(len(peers)-avgPerTopic, 0)
		var prunedPeers peer.IDSlice
		for i, pID := range peers {
			if i > count {
				break
			}
			prunedPeers = append(prunedPeers, pID)
			pm.host.Peerstore().RemovePeer(pID)
			numPeers--
			if numPeers < pm.maxPeers {
				pm.logger.Debug("finished pruning peer store", zap.Int("capacity", pm.maxPeers), zap.Int("beforeNumPeers", peerCntBeforePruning), zap.Int("afterNumPeers", numPeers), zap.Stringers("prunedPeers", prunedPeers))
				return
			}
		}
		pm.logger.Debug("pruned peers higher than average", zap.Stringers("prunedPeers", prunedPeers), zap.String("topic", topic))
	}
	pm.logger.Debug("finished pruning peer store", zap.Int("capacity", pm.maxPeers), zap.Int("beforeNumPeers", peerCntBeforePruning), zap.Int("afterNumPeers", numPeers))
}

// This is a connectivity loop, which currently checks and prunes inbound connections.
func (pm *PeerManager) connectivityLoop(ctx context.Context) {
	pm.connectToPeers()
	t := time.NewTicker(peerConnectivityLoopSecs * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pm.connectToPeers()
		}
	}
}

// GroupPeersByDirection returns all the connected peers in peer store grouped by Inbound or outBound direction
func (pm *PeerManager) GroupPeersByDirection(specificPeers ...peer.ID) (inPeers peer.IDSlice, outPeers peer.IDSlice, err error) {
	if len(specificPeers) == 0 {
		specificPeers = pm.host.Network().Peers()
	}

	for _, p := range specificPeers {
		direction, err := pm.host.Peerstore().(wps.WakuPeerstore).Direction(p)
		if err == nil {
			if direction == network.DirInbound {
				inPeers = append(inPeers, p)
			} else if direction == network.DirOutbound {
				outPeers = append(outPeers, p)
			}
		} else {
			pm.logger.Error("failed to retrieve peer direction",
				zap.Stringer("peerID", p), zap.Error(err))
		}
	}
	return inPeers, outPeers, nil
}

// getRelayPeers - Returns list of in and out peers supporting WakuRelayProtocol within specifiedPeers.
// If specifiedPeers is empty, it checks within all peers in peerStore.
func (pm *PeerManager) getRelayPeers(specificPeers ...peer.ID) (inRelayPeers peer.IDSlice, outRelayPeers peer.IDSlice) {
	//Group peers by their connected direction inbound or outbound.
	inPeers, outPeers, err := pm.GroupPeersByDirection(specificPeers...)
	if err != nil {
		return
	}
	pm.logger.Debug("number of peers connected", zap.Int("inPeers", inPeers.Len()),
		zap.Int("outPeers", outPeers.Len()))

	//Need to filter peers to check if they support relay
	if inPeers.Len() != 0 {
		inRelayPeers, _ = pm.FilterPeersByProto(inPeers, nil, relay.WakuRelayID_v200)
	}
	if outPeers.Len() != 0 {
		outRelayPeers, _ = pm.FilterPeersByProto(outPeers, nil, relay.WakuRelayID_v200)
	}
	return
}

// ensureMinRelayConnsPerTopic makes sure there are min of D conns per pubsubTopic.
// If not it will look into peerStore to initiate more connections.
// If peerStore doesn't have enough peers, will wait for discv5 to find more and try in next cycle
func (pm *PeerManager) ensureMinRelayConnsPerTopic() {
	pm.topicMutex.RLock()
	defer pm.topicMutex.RUnlock()
	for topicStr, topicInst := range pm.subRelayTopics {

		meshPeerLen := pm.checkAndUpdateTopicHealth(topicInst)
		curConnectedPeerLen := pm.getPeersBasedOnconnectionStatus(topicStr, network.Connected).Len()

		if meshPeerLen < waku_proto.GossipSubDMin || curConnectedPeerLen < pm.OutPeersTarget {
			pm.logger.Debug("subscribed topic has not reached target peers, initiating more connections to maintain healthy mesh",
				zap.String("pubSubTopic", topicStr), zap.Int("connectedPeerCount", curConnectedPeerLen),
				zap.Int("targetPeers", pm.OutPeersTarget))
			//Find not connected peers.
			notConnectedPeers := pm.getPeersBasedOnconnectionStatus(topicStr, network.NotConnected)
			if notConnectedPeers.Len() == 0 {
				pm.logger.Debug("could not find any peers in peerstore to connect to, discovering more", zap.String("pubSubTopic", topicStr))
				go pm.discoverPeersByPubsubTopics([]string{topicStr}, relay.WakuRelayID_v200, pm.ctx, 2)
				continue
			}
			pm.logger.Debug("connecting to eligible peers in peerstore", zap.String("pubSubTopic", topicStr))
			//Connect to eligible peers.
			numPeersToConnect := pm.OutPeersTarget - curConnectedPeerLen
			if numPeersToConnect > 0 {
				if numPeersToConnect > notConnectedPeers.Len() {
					numPeersToConnect = notConnectedPeers.Len()
				}
				pm.connectToSpecifiedPeers(notConnectedPeers[0:numPeersToConnect])
			}
		}
	}
}

// connectToPeers ensures minimum D connections are there for each pubSubTopic.
// If not, initiates connections to additional peers.
// It also checks for incoming relay connections and prunes once they cross inRelayTarget
func (pm *PeerManager) connectToPeers() {
	if pm.RelayEnabled {
		//Check for out peer connections and connect to more peers.
		pm.ensureMinRelayConnsPerTopic()

		inRelayPeers, outRelayPeers := pm.getRelayPeers()
		pm.logger.Debug("number of relay peers connected",
			zap.Int("in", inRelayPeers.Len()),
			zap.Int("out", outRelayPeers.Len()))
		if inRelayPeers.Len() > 0 &&
			inRelayPeers.Len() > pm.InPeersTarget {
			pm.pruneInRelayConns(inRelayPeers)
		}
	} else {
		//TODO: Connect to filter peers per topic as of now.
		//Fetch filter peers from peerStore, TODO: topics for lightNode not available here?
		//Filter subscribe to notify peerManager whenever a new topic/shard is subscribed to.
		pm.logger.Debug("light mode..not doing anything")
	}
}

// connectToSpecifiedPeers connects to peers provided in the list if the addresses have not expired.
func (pm *PeerManager) connectToSpecifiedPeers(peers peer.IDSlice) {
	for _, peerID := range peers {
		peerData := AddrInfoToPeerData(wps.PeerManager, peerID, pm.host)
		if peerData == nil {
			continue
		}
		pm.peerConnector.PushToChan(*peerData)
	}
}

// getPeersBasedOnconnectionStatus returns peers for a pubSubTopic that are either connected/not-connected based on status passed.
func (pm *PeerManager) getPeersBasedOnconnectionStatus(pubsubTopic string, connected network.Connectedness) (filteredPeers peer.IDSlice) {
	var peerList peer.IDSlice
	if pubsubTopic == "" {
		peerList = pm.host.Peerstore().Peers()
	} else {
		peerList = pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubsubTopic)
	}
	for _, peerID := range peerList {
		if pm.host.Network().Connectedness(peerID) == connected {
			filteredPeers = append(filteredPeers, peerID)
		}
	}
	return
}

// pruneInRelayConns prune any incoming relay connections crossing derived inrelayPeerTarget
func (pm *PeerManager) pruneInRelayConns(inRelayPeers peer.IDSlice) {

	//Start disconnecting peers, based on what?
	//For now no preference is used
	//TODO: Need to have more intelligent way of doing this, maybe peer scores.
	//TODO: Keep optimalPeersRequired for a pubSubTopic in mind while pruning connections to peers.
	pm.logger.Info("peer connections exceed target relay peers, hence pruning",
		zap.Int("cnt", inRelayPeers.Len()), zap.Int("target", pm.InPeersTarget))
	for pruningStartIndex := pm.InPeersTarget; pruningStartIndex < inRelayPeers.Len(); pruningStartIndex++ {
		p := inRelayPeers[pruningStartIndex]
		err := pm.host.Network().ClosePeer(p)
		if err != nil {
			pm.logger.Warn("failed to disconnect connection towards peer",
				zap.Stringer("peerID", p))
		}
		pm.logger.Debug("successfully disconnected connection towards peer",
			zap.Stringer("peerID", p))
	}
}

func (pm *PeerManager) processPeerENR(p *service.PeerData) []protocol.ID {
	shards, err := wenr.RelaySharding(p.ENR.Record())
	if err != nil {
		pm.logger.Error("could not derive relayShards from ENR", zap.Error(err),
			zap.Stringer("peer", p.AddrInfo.ID), zap.String("enr", p.ENR.String()))
	} else {
		if shards != nil {
			p.PubsubTopics = make([]string, 0)
			topics := shards.Topics()
			for _, topic := range topics {
				topicStr := topic.String()
				p.PubsubTopics = append(p.PubsubTopics, topicStr)
			}
		} else {
			pm.logger.Debug("ENR doesn't have relay shards", zap.Stringer("peer", p.AddrInfo.ID))
		}
	}
	supportedProtos := []protocol.ID{}
	//Identify and specify protocols supported by the peer based on the discovered peer's ENR
	var enrField wenr.WakuEnrBitfield
	if err := p.ENR.Record().Load(enr.WithEntry(wenr.WakuENRField, &enrField)); err == nil {
		for proto, protoENR := range pm.wakuprotoToENRFieldMap {
			protoENRField := protoENR.waku2ENRBitField
			if protoENRField&enrField != 0 {
				supportedProtos = append(supportedProtos, proto)
				//Add Service peers to serviceSlots.
				pm.addPeerToServiceSlot(proto, p.AddrInfo.ID)
			}
		}
	}
	return supportedProtos
}

// AddDiscoveredPeer to add dynamically discovered peers.
// Note that these peers will not be set in service-slots.
func (pm *PeerManager) AddDiscoveredPeer(p service.PeerData, connectNow bool) {
	//Check if the peer is already present, if so skip adding
	_, err := pm.host.Peerstore().(wps.WakuPeerstore).Origin(p.AddrInfo.ID)
	if err == nil {
		//Add addresses if existing addresses have expired
		existingAddrs := pm.host.Peerstore().Addrs(p.AddrInfo.ID)
		if len(existingAddrs) == 0 {
			pm.host.Peerstore().AddAddrs(p.AddrInfo.ID, p.AddrInfo.Addrs, peerstore.AddressTTL)
		}
		enr, err := pm.host.Peerstore().(wps.WakuPeerstore).ENR(p.AddrInfo.ID)
		// Verifying if the enr record is more recent (DiscV5 and peer exchange can return peers already seen)
		if err == nil {
			if p.ENR != nil {
				if enr.Record().Seq() >= p.ENR.Seq() {
					return
				}
				//Peer is already in peer-store but stored ENR is older than discovered one.
				pm.logger.Info("peer already found in peerstore, but re-adding it as ENR sequence is higher than locally stored",
					zap.Stringer("peer", p.AddrInfo.ID), logging.Uint64("newENRSeq", p.ENR.Record().Seq()), logging.Uint64("storedENRSeq", enr.Record().Seq()))
			} else {
				pm.logger.Info("peer already found in peerstore, but no new ENR", zap.Stringer("peer", p.AddrInfo.ID))
			}
		} else {
			//Peer is in peer-store but it doesn't have an enr
			pm.logger.Info("peer already found in peerstore, but doesn't have an ENR record, re-adding",
				zap.Stringer("peer", p.AddrInfo.ID))
		}
	}
	pm.logger.Debug("adding discovered peer", zap.Stringer("peerID", p.AddrInfo.ID))

	supportedProtos := []protocol.ID{}
	if len(p.PubsubTopics) == 0 && p.ENR != nil {
		// Try to fetch shard info and supported protocols from ENR to arrive at pubSub topics.
		supportedProtos = pm.processPeerENR(&p)
	}

	_ = pm.addPeer(p.AddrInfo.ID, p.AddrInfo.Addrs, p.Origin, p.PubsubTopics, supportedProtos...)

	if p.ENR != nil {
		pm.logger.Debug("setting ENR for peer", zap.Stringer("peerID", p.AddrInfo.ID), zap.Stringer("enr", p.ENR))
		err := pm.host.Peerstore().(wps.WakuPeerstore).SetENR(p.AddrInfo.ID, p.ENR)
		if err != nil {
			pm.logger.Error("could not store enr", zap.Error(err),
				zap.Stringer("peer", p.AddrInfo.ID), zap.String("enr", p.ENR.String()))
		}
	}
	if connectNow {
		pm.logger.Debug("connecting now to discovered peer", zap.Stringer("peer", p.AddrInfo.ID))
		go pm.peerConnector.PushToChan(p)
	}
}

// addPeer adds peer to the peerStore.
// It also sets additional metadata such as origin and supported protocols
func (pm *PeerManager) addPeer(ID peer.ID, addrs []ma.Multiaddr, origin wps.Origin, pubSubTopics []string, protocols ...protocol.ID) error {

	pm.logger.Info("adding peer to peerstore", zap.Stringer("peer", ID))
	if origin == wps.Static {
		pm.host.Peerstore().AddAddrs(ID, addrs, peerstore.PermanentAddrTTL)
	} else {
		//Need to re-evaluate the address expiry
		// For now expiring them with default addressTTL which is an hour.
		pm.host.Peerstore().AddAddrs(ID, addrs, peerstore.AddressTTL)
	}
	err := pm.host.Peerstore().(wps.WakuPeerstore).SetOrigin(ID, origin)
	if err != nil {
		pm.logger.Error("could not set origin", zap.Error(err), zap.Stringer("peer", ID))
		return err
	}

	if len(protocols) > 0 {
		err = pm.host.Peerstore().AddProtocols(ID, protocols...)
		if err != nil {
			pm.logger.Error("could not set protocols", zap.Error(err), zap.Stringer("peer", ID))
			return err
		}
	}
	if len(pubSubTopics) == 0 {
		// Probably the peer is discovered via DNSDiscovery (for which we don't have pubSubTopic info)
		//If pubSubTopic and enr is empty or no shard info in ENR,then set to defaultPubSubTopic
		pubSubTopics = []string{relay.DefaultWakuTopic}
	}
	err = pm.host.Peerstore().(wps.WakuPeerstore).SetPubSubTopics(ID, pubSubTopics)
	if err != nil {
		pm.logger.Error("could not store pubSubTopic", zap.Error(err),
			zap.Stringer("peer", ID), zap.Strings("topics", pubSubTopics))
	}
	return nil
}

func AddrInfoToPeerData(origin wps.Origin, peerID peer.ID, host host.Host, pubsubTopics ...string) *service.PeerData {
	addrs := host.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		//Addresses expired, remove peer from peerStore
		host.Peerstore().RemovePeer(peerID)
		return nil
	}
	return &service.PeerData{
		Origin: origin,
		AddrInfo: peer.AddrInfo{
			ID:    peerID,
			Addrs: addrs,
		},
		PubsubTopics: pubsubTopics,
	}
}

// AddPeer adds peer to the peerStore and also to service slots
func (pm *PeerManager) AddPeer(address ma.Multiaddr, origin wps.Origin, pubsubTopics []string, protocols ...protocol.ID) (*service.PeerData, error) {
	//Assuming all addresses have peerId
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return nil, err
	}

	//Add Service peers to serviceSlots.
	for _, proto := range protocols {
		pm.addPeerToServiceSlot(proto, info.ID)
	}

	//Add to the peer-store
	err = pm.addPeer(info.ID, info.Addrs, origin, pubsubTopics, protocols...)
	if err != nil {
		return nil, err
	}

	pData := &service.PeerData{
		Origin: origin,
		AddrInfo: peer.AddrInfo{
			ID:    info.ID,
			Addrs: info.Addrs,
		},
		PubsubTopics: pubsubTopics,
	}

	return pData, nil
}

// Connect establishes a connection to a
func (pm *PeerManager) Connect(pData *service.PeerData) {
	go pm.peerConnector.PushToChan(*pData)
}

// RemovePeer deletes peer from the peerStore after disconnecting it.
// It also removes the peer from serviceSlot.
func (pm *PeerManager) RemovePeer(peerID peer.ID) {
	pm.host.Peerstore().RemovePeer(peerID)
	//Search if this peer is in serviceSlot and if so, remove it from there
	// TODO:Add another peer which is statically configured to the serviceSlot.
	pm.serviceSlots.removePeer(peerID)
}

// addPeerToServiceSlot adds a peerID to serviceSlot.
// Adding to peerStore is expected to be already done by caller.
// If relay proto is passed, it is not added to serviceSlot.
func (pm *PeerManager) addPeerToServiceSlot(proto protocol.ID, peerID peer.ID) {
	if proto == relay.WakuRelayID_v200 {
		pm.logger.Debug("cannot add Relay peer to service peer slots")
		return
	}

	//For now adding the peer to serviceSlot which means the latest added peer would be given priority.
	//TODO: Ideally we should sort the peers per service and return best peer based on peer score or RTT etc.
	pm.logger.Info("adding peer to service slots", zap.Stringer("peer", peerID),
		zap.String("service", string(proto)))
	// getPeers returns nil for WakuRelayIDv200 protocol, but we don't run this ServiceSlot code for WakuRelayIDv200 protocol
	pm.serviceSlots.getPeers(proto).add(peerID)
}
