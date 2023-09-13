package peermanager

import (
	"context"
	"errors"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	wenr "github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"

	"go.uber.org/zap"
)

// NodeTopicDetails stores pubSubTopic related data like topicHandle for the node.
type NodeTopicDetails struct {
	topic *pubsub.Topic
}

// PeerManager applies various controls and manage connections towards peers.
type PeerManager struct {
	peerConnector       *PeerConnectionStrategy
	maxPeers            int
	maxRelayPeers       int
	logger              *zap.Logger
	InRelayPeersTarget  int
	OutRelayPeersTarget int
	host                host.Host
	serviceSlots        *ServiceSlots
	ctx                 context.Context
	sub                 event.Subscription
	topicMutex          sync.RWMutex
	subRelayTopics      map[string]*NodeTopicDetails
}

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

// NewPeerManager creates a new peerManager instance.
func NewPeerManager(maxConnections int, maxPeers int, logger *zap.Logger) *PeerManager {

	maxRelayPeers, _ := relayAndServicePeers(maxConnections)
	inRelayPeersTarget, outRelayPeersTarget := inAndOutRelayPeers(maxRelayPeers)

	if maxPeers == 0 || maxConnections > maxPeers {
		maxPeers = maxConnsToPeerRatio * maxConnections
	}

	pm := &PeerManager{
		logger:              logger.Named("peer-manager"),
		maxRelayPeers:       maxRelayPeers,
		InRelayPeersTarget:  inRelayPeersTarget,
		OutRelayPeersTarget: outRelayPeersTarget,
		serviceSlots:        NewServiceSlot(),
		subRelayTopics:      make(map[string]*NodeTopicDetails),
		maxPeers:            maxPeers,
	}
	logger.Info("PeerManager init values", zap.Int("maxConnections", maxConnections),
		zap.Int("maxRelayPeers", maxRelayPeers),
		zap.Int("outRelayPeersTarget", outRelayPeersTarget),
		zap.Int("inRelayPeersTarget", pm.InRelayPeersTarget),
		zap.Int("maxPeers", maxPeers))

	return pm
}

// SetHost sets the host to be used in order to access the peerStore.
func (pm *PeerManager) SetHost(host host.Host) {
	pm.host = host
}

// SetPeerConnector sets the peer connector to be used for establishing relay connections.
func (pm *PeerManager) SetPeerConnector(pc *PeerConnectionStrategy) {
	pm.peerConnector = pc
}

// Start starts the processing to be done by peer manager.
func (pm *PeerManager) Start(ctx context.Context) {
	pm.ctx = ctx
	if pm.sub != nil {
		go pm.peerEventLoop(ctx)
	}
	go pm.connectivityLoop(ctx)
}

// This is a connectivity loop, which currently checks and prunes inbound connections.
func (pm *PeerManager) connectivityLoop(ctx context.Context) {
	pm.connectToRelayPeers()
	t := time.NewTicker(peerConnectivityLoopSecs * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			pm.connectToRelayPeers()
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
			pm.logger.Error("Failed to retrieve peer direction",
				logging.HostID("peerID", p), zap.Error(err))
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
	pm.logger.Debug("Number of peers connected", zap.Int("inPeers", inPeers.Len()),
		zap.Int("outPeers", outPeers.Len()))

	//Need to filter peers to check if they support relay
	if inPeers.Len() != 0 {
		inRelayPeers, _ = utils.FilterPeersByProto(pm.host, inPeers, relay.WakuRelayID_v200)
	}
	if outPeers.Len() != 0 {
		outRelayPeers, _ = utils.FilterPeersByProto(pm.host, outPeers, relay.WakuRelayID_v200)
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
		curPeers := topicInst.topic.ListPeers()
		curPeerLen := len(curPeers)
		if curPeerLen < waku_proto.GossipSubOptimalFullMeshSize {
			pm.logger.Info("Subscribed topic is unhealthy, initiating more connections to maintain health",
				zap.String("pubSubTopic", topicStr), zap.Int("connectedPeerCount", curPeerLen),
				zap.Int("optimumPeers", waku_proto.GossipSubOptimalFullMeshSize))
			//Find not connected peers.
			notConnectedPeers := pm.getNotConnectedPers(topicStr)
			if notConnectedPeers.Len() == 0 {
				//TODO: Trigger on-demand discovery for this topic.
				continue
			}
			//Connect to eligible peers.
			numPeersToConnect := waku_proto.GossipSubOptimalFullMeshSize - curPeerLen

			if numPeersToConnect > notConnectedPeers.Len() {
				numPeersToConnect = notConnectedPeers.Len()
			}
			pm.connectToPeers(notConnectedPeers[0:numPeersToConnect])
		}
	}
}

// connectToRelayPeers ensures minimum D connections are there for each pubSubTopic.
// If not, initiates connections to additional peers.
// It also checks for incoming relay connections and prunes once they cross inRelayTarget
func (pm *PeerManager) connectToRelayPeers() {
	//Check for out peer connections and connect to more peers.
	pm.ensureMinRelayConnsPerTopic()

	inRelayPeers, outRelayPeers := pm.getRelayPeers()
	pm.logger.Info("number of relay peers connected",
		zap.Int("in", inRelayPeers.Len()),
		zap.Int("out", outRelayPeers.Len()))
	if inRelayPeers.Len() > 0 &&
		inRelayPeers.Len() > pm.InRelayPeersTarget {
		pm.pruneInRelayConns(inRelayPeers)
	}
}

// addrInfoToPeerData returns addressinfo for a peer
// If addresses are expired, it removes the peer from host peerStore and returns nil.
func addrInfoToPeerData(origin wps.Origin, peerID peer.ID, host host.Host) *PeerData {
	addrs := host.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		//Addresses expired, remove peer from peerStore
		host.Peerstore().RemovePeer(peerID)
		return nil
	}
	return &PeerData{
		Origin: origin,
		AddrInfo: peer.AddrInfo{
			ID:    peerID,
			Addrs: addrs,
		},
	}
}

// connectToPeers connects to peers provided in the list if the addresses have not expired.
func (pm *PeerManager) connectToPeers(peers peer.IDSlice) {
	for _, peerID := range peers {
		peerData := addrInfoToPeerData(wps.PeerManager, peerID, pm.host)
		if peerData == nil {
			continue
		}
		pm.peerConnector.PushToChan(*peerData)
	}
}

// getNotConnectedPers returns peers for a pubSubTopic that are not connected.
func (pm *PeerManager) getNotConnectedPers(pubsubTopic string) (notConnectedPeers peer.IDSlice) {
	var peerList peer.IDSlice
	if pubsubTopic == "" {
		peerList = pm.host.Peerstore().Peers()
	} else {
		peerList = pm.host.Peerstore().(*wps.WakuPeerstoreImpl).PeersByPubSubTopic(pubsubTopic)
	}
	for _, peerID := range peerList {
		if pm.host.Network().Connectedness(peerID) != network.Connected {
			notConnectedPeers = append(notConnectedPeers, peerID)
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
		zap.Int("cnt", inRelayPeers.Len()), zap.Int("target", pm.InRelayPeersTarget))
	for pruningStartIndex := pm.InRelayPeersTarget; pruningStartIndex < inRelayPeers.Len(); pruningStartIndex++ {
		p := inRelayPeers[pruningStartIndex]
		err := pm.host.Network().ClosePeer(p)
		if err != nil {
			pm.logger.Warn("Failed to disconnect connection towards peer",
				logging.HostID("peerID", p))
		}
		pm.logger.Debug("Successfully disconnected connection towards peer",
			logging.HostID("peerID", p))
	}
}

// AddDiscoveredPeer to add dynamically discovered peers.
// Note that these peers will not be set in service-slots.
// TODO: It maybe good to set in service-slots based on services supported in the ENR
func (pm *PeerManager) AddDiscoveredPeer(p PeerData, connectNow bool) {
	//Doing this check again inside addPeer, in order to avoid additional complexity of rollingBack other changes.
	if pm.maxPeers <= pm.host.Peerstore().Peers().Len() {
		return
	}
	//Check if the peer is already present, if so skip adding
	_, err := pm.host.Peerstore().(wps.WakuPeerstore).Origin(p.AddrInfo.ID)
	if err == nil {
		pm.logger.Debug("Found discovered peer already in peerStore", logging.HostID("peer", p.AddrInfo.ID))
		return
	}
	// Try to fetch shard info from ENR to arrive at pubSub topics.
	if len(p.PubSubTopics) == 0 && p.ENR != nil {
		shards, err := wenr.RelaySharding(p.ENR.Record())
		if err != nil {
			pm.logger.Error("Could not derive relayShards from ENR", zap.Error(err),
				logging.HostID("peer", p.AddrInfo.ID), zap.String("enr", p.ENR.String()))
		} else {
			if shards != nil {
				p.PubSubTopics = make([]string, 0)
				topics := shards.Topics()
				for _, topic := range topics {
					topicStr := topic.String()
					p.PubSubTopics = append(p.PubSubTopics, topicStr)
				}
			} else {
				pm.logger.Debug("ENR doesn't have relay shards", logging.HostID("peer", p.AddrInfo.ID))
			}
		}
	}

	_ = pm.addPeer(p.AddrInfo.ID, p.AddrInfo.Addrs, p.Origin, p.PubSubTopics)

	if p.ENR != nil {
		err := pm.host.Peerstore().(wps.WakuPeerstore).SetENR(p.AddrInfo.ID, p.ENR)
		if err != nil {
			pm.logger.Error("could not store enr", zap.Error(err),
				logging.HostID("peer", p.AddrInfo.ID), zap.String("enr", p.ENR.String()))
		}
	}
	if connectNow {
		pm.peerConnector.PushToChan(p)
	}
}

// addPeer adds peer to only the peerStore.
// It also sets additional metadata such as origin, ENR and supported protocols
func (pm *PeerManager) addPeer(ID peer.ID, addrs []ma.Multiaddr, origin wps.Origin, pubSubTopics []string, protocols ...protocol.ID) error {
	if pm.maxPeers <= pm.host.Peerstore().Peers().Len() {
		return errors.New("peer store capacity reached")
	}
	pm.logger.Info("adding peer to peerstore", logging.HostID("peer", ID))
	if origin == wps.Static {
		pm.host.Peerstore().AddAddrs(ID, addrs, peerstore.PermanentAddrTTL)
	} else {
		//Need to re-evaluate the address expiry
		// For now expiring them with default addressTTL which is an hour.
		pm.host.Peerstore().AddAddrs(ID, addrs, peerstore.AddressTTL)
	}
	err := pm.host.Peerstore().(wps.WakuPeerstore).SetOrigin(ID, origin)
	if err != nil {
		pm.logger.Error("could not set origin", zap.Error(err), logging.HostID("peer", ID))
		return err
	}

	if len(protocols) > 0 {
		err = pm.host.Peerstore().AddProtocols(ID, protocols...)
		if err != nil {
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
			logging.HostID("peer", ID), zap.Strings("topics", pubSubTopics))
	}
	return nil
}

// AddPeer adds peer to the peerStore and also to service slots
func (pm *PeerManager) AddPeer(address ma.Multiaddr, origin wps.Origin, pubSubTopics []string, protocols ...protocol.ID) (peer.ID, error) {
	//Assuming all addresses have peerId
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return "", err
	}

	//Add Service peers to serviceSlots.
	for _, proto := range protocols {
		pm.addPeerToServiceSlot(proto, info.ID)
	}

	//Add to the peer-store
	err = pm.addPeer(info.ID, info.Addrs, origin, pubSubTopics, protocols...)
	if err != nil {
		return "", err
	}

	return info.ID, nil
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
		pm.logger.Warn("Cannot add Relay peer to service peer slots")
		return
	}

	//For now adding the peer to serviceSlot which means the latest added peer would be given priority.
	//TODO: Ideally we should sort the peers per service and return best peer based on peer score or RTT etc.
	pm.logger.Info("Adding peer to service slots", logging.HostID("peer", peerID),
		zap.String("service", string(proto)))
	// getPeers returns nil for WakuRelayIDv200 protocol, but we don't run this ServiceSlot code for WakuRelayIDv200 protocol
	pm.serviceSlots.getPeers(proto).add(peerID)
}

// SelectPeerByContentTopic is used to return a random peer that supports a given protocol for given contentTopic.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol and contentTopic, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
func (pm *PeerManager) SelectPeerByContentTopic(proto protocol.ID, contentTopic string, specificPeers ...peer.ID) (peer.ID, error) {
	pubsubTopic, err := waku_proto.GetPubSubTopicFromContentTopic(contentTopic)
	if err != nil {
		return "", err
	}
	return pm.SelectPeer(proto, pubsubTopic, specificPeers...)
}

// SelectPeer is used to return a random peer that supports a given protocol.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
// if pubSubTopic is specified, peer is selected from list that support the pubSubTopic
func (pm *PeerManager) SelectPeer(proto protocol.ID, pubSubTopic string, specificPeers ...peer.ID) (peer.ID, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?

	if peerID := pm.selectServicePeer(proto, pubSubTopic, specificPeers...); peerID != nil {
		return *peerID, nil
	}

	// if not found in serviceSlots or proto == WakuRelayIDv200
	filteredPeers, err := utils.FilterPeersByProto(pm.host, specificPeers, proto)
	if err != nil {
		return "", err
	}
	if pubSubTopic != "" {
		filteredPeers = pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopic(pubSubTopic, filteredPeers...)
	}
	return utils.SelectRandomPeer(filteredPeers, pm.logger)
}

func (pm *PeerManager) selectServicePeer(proto protocol.ID, pubSubTopic string, specificPeers ...peer.ID) (peerIDPtr *peer.ID) {
	peerIDPtr = nil

	//Try to fetch from serviceSlot
	if slot := pm.serviceSlots.getPeers(proto); slot != nil {
		if pubSubTopic == "" {
			if peerID, err := slot.getRandom(); err == nil {
				peerIDPtr = &peerID
			} else {
				pm.logger.Debug("could not retrieve random peer from slot", zap.Error(err))
			}
		} else { //PubsubTopic based selection
			keys := make([]peer.ID, 0, len(slot.m))
			for i := range slot.m {
				keys = append(keys, i)
			}
			selectedPeers := pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopic(pubSubTopic, keys...)
			peerID, err := utils.SelectRandomPeer(selectedPeers, pm.logger)
			if err == nil {
				peerIDPtr = &peerID
			} else {
				pm.logger.Debug("could not select random peer", zap.Error(err))
			}
		}
	}
	return
}
