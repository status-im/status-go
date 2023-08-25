package peermanager

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-waku/logging"
	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	"github.com/waku-org/go-waku/waku/v2/utils"

	"go.uber.org/zap"
)

// TODO: Move all the protocol IDs to a common location.
// WakuRelayIDv200 is protocol ID for Waku v2 relay protocol
const WakuRelayIDv200 = protocol.ID("/vac/waku/relay/2.0.0")

// PeerManager applies various controls and manage connections towards peers.
type PeerManager struct {
	peerConnector       *PeerConnectionStrategy
	maxConnections      int
	maxRelayPeers       int
	logger              *zap.Logger
	InRelayPeersTarget  int
	OutRelayPeersTarget int
	host                host.Host
	serviceSlots        map[protocol.ID][]peer.ID
	ctx                 context.Context
}

const maxRelayPeersShare = 5

// const defaultMaxOutRelayPeersTarget = 10
const outRelayPeersShare = 3
const peerConnectivityLoopSecs = 15
const minOutRelayConns = 10

// NewPeerManager creates a new peerManager instance.
func NewPeerManager(maxConnections int, logger *zap.Logger) *PeerManager {

	maxRelayPeersValue := maxConnections - (maxConnections / maxRelayPeersShare)
	outRelayPeersTargetValue := int(maxRelayPeersValue / outRelayPeersShare)
	if outRelayPeersTargetValue < minOutRelayConns {
		outRelayPeersTargetValue = minOutRelayConns
	}
	inRelayPeersTargetValue := maxRelayPeersValue - outRelayPeersTargetValue
	if inRelayPeersTargetValue < 0 {
		inRelayPeersTargetValue = 0
	}

	pm := &PeerManager{
		maxConnections:      maxConnections,
		logger:              logger.Named("peer-manager"),
		maxRelayPeers:       maxRelayPeersValue,
		InRelayPeersTarget:  inRelayPeersTargetValue,
		OutRelayPeersTarget: outRelayPeersTargetValue,
		serviceSlots:        make(map[protocol.ID][]peer.ID),
	}
	logger.Info("PeerManager init values", zap.Int("maxConnections", maxConnections),
		zap.Int("maxRelayPeersValue", maxRelayPeersValue),
		zap.Int("outRelayPeersTargetValue", outRelayPeersTargetValue),
		zap.Int("inRelayPeersTarget", pm.InRelayPeersTarget))

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
	go pm.connectivityLoop(ctx)
}

// This is a connectivity loop, which currently checks and prunes inbound connections.
func (pm *PeerManager) connectivityLoop(ctx context.Context) {
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
func (pm *PeerManager) GroupPeersByDirection() (inPeers peer.IDSlice, outPeers peer.IDSlice, err error) {
	peers := pm.host.Network().Peers()

	for _, p := range peers {
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

func (pm *PeerManager) getRelayPeers() (inRelayPeers peer.IDSlice, outRelayPeers peer.IDSlice) {
	//Group peers by their connected direction inbound or outbound.
	inPeers, outPeers, err := pm.GroupPeersByDirection()
	if err != nil {
		return
	}
	pm.logger.Info("Number of peers connected", zap.Int("inPeers", inPeers.Len()),
		zap.Int("outPeers", outPeers.Len()))

	//Need to filter peers to check if they support relay
	if inPeers.Len() != 0 {
		inRelayPeers, _ = utils.FilterPeersByProto(pm.host, inPeers, WakuRelayIDv200)
	}
	if outPeers.Len() != 0 {
		outRelayPeers, _ = utils.FilterPeersByProto(pm.host, outPeers, WakuRelayIDv200)
	}
	pm.logger.Info("Number of Relay peers connected", zap.Int("inRelayPeers", inRelayPeers.Len()),
		zap.Int("outRelayPeers", outRelayPeers.Len()))
	return
}

func (pm *PeerManager) connectToRelayPeers() {

	//Check for out peer connections and connect to more peers.
	inRelayPeers, outRelayPeers := pm.getRelayPeers()
	if inRelayPeers.Len() > 0 &&
		inRelayPeers.Len() > pm.InRelayPeersTarget {
		pm.pruneInRelayConns(inRelayPeers, outRelayPeers)
	}

	if outRelayPeers.Len() > pm.OutRelayPeersTarget {
		return
	}
	totalRelayPeers := inRelayPeers.Len() + outRelayPeers.Len()
	// Establish additional connections if there are peers.
	//What if the not connected peers in peerstore are not relay peers???
	if totalRelayPeers < pm.host.Peerstore().Peers().Len() {
		//Find not connected peers.
		notConnectedPeers := pm.getNotConnectedPers()
		//Figure out outside backoff peers.

		//Connect to eligible peers.
		numPeersToConnect := pm.maxRelayPeers - totalRelayPeers

		if numPeersToConnect > notConnectedPeers.Len() {
			numPeersToConnect = notConnectedPeers.Len() - 1
		}

		pm.connectToPeers(notConnectedPeers[0:numPeersToConnect])
	} //Else: Should we raise some sort of unhealthy event??
}

func (pm *PeerManager) connectToPeers(peers peer.IDSlice) {
	for _, peerID := range peers {
		peerInfo := peer.AddrInfo{
			ID:    peerID,
			Addrs: pm.host.Peerstore().Addrs(peerID),
		}
		pm.peerConnector.publishWork(pm.ctx, peerInfo)
	}
}

func (pm *PeerManager) getNotConnectedPers() (notConnectedPeers peer.IDSlice) {
	for _, peerID := range pm.host.Peerstore().Peers() {
		if pm.host.Network().Connectedness(peerID) != network.Connected {
			notConnectedPeers = append(notConnectedPeers, peerID)
		}
	}
	return
}

func (pm *PeerManager) pruneInRelayConns(inRelayPeers peer.IDSlice, outRelayPeers peer.IDSlice) {

	//Start disconnecting peers, based on what?
	//For now, just disconnect most recently connected peers
	//TODO: Need to have more intelligent way of doing this, maybe peer scores.
	pm.logger.Info("Number of in peer connections exceed targer relay peers, hence pruning",
		zap.Int("inRelayPeers", inRelayPeers.Len()), zap.Int("inRelayPeersTarget", pm.InRelayPeersTarget))
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
func (pm *PeerManager) AddDiscoveredPeer(p PeerData) {

	_ = pm.addPeer(p.AddrInfo.ID, p.AddrInfo.Addrs, p.Origin)

	if p.ENR != nil {
		err := pm.host.Peerstore().(wps.WakuPeerstore).SetENR(p.AddrInfo.ID, p.ENR)
		if err != nil {
			pm.logger.Error("could not store enr", zap.Error(err),
				logging.HostID("peer", p.AddrInfo.ID), zap.String("enr", p.ENR.String()))
		}
	}
}

// addPeer adds peer to only the peerStore.
// It also sets additional metadata such as origin, ENR and supported protocols
func (pm *PeerManager) addPeer(ID peer.ID, addrs []ma.Multiaddr, origin wps.Origin, protocols ...protocol.ID) error {
	pm.logger.Info("adding peer to peerstore", logging.HostID("peer", ID))
	pm.host.Peerstore().AddAddrs(ID, addrs, peerstore.AddressTTL)
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
	return nil
}

// AddPeer adds peer to the peerStore and also to service slots
func (pm *PeerManager) AddPeer(address ma.Multiaddr, origin wps.Origin, protocols ...protocol.ID) (peer.ID, error) {
	//Assuming all addresses have peerId
	info, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		return "", err
	}

	//Add Service peers to serviceSlots.
	for _, proto := range protocols {
		pm.AddPeerToServiceSlot(proto, info.ID, origin)
	}

	//Add to the peer-store
	err = pm.addPeer(info.ID, info.Addrs, origin, protocols...)
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
	for proto, peers := range pm.serviceSlots {
		for i, peer := range peers {
			if peer == peerID {
				pm.serviceSlots[proto][i] = ""
			}
		}
	}
}

// AddPeerToServiceSlot adds a peerID to serviceSlot.
// Adding to peerStore is expected to be already done by caller.
// If relay proto is passed, it is not added to serviceSlot.
func (pm *PeerManager) AddPeerToServiceSlot(proto protocol.ID, peerID peer.ID, origin wps.Origin) {
	if proto == WakuRelayIDv200 {
		pm.logger.Warn("Cannot add Relay peer to service peer slots")
		return
	}

	//For now adding the peer to serviceSlot which means the latest added peer would be given priority.
	//TODO: Ideally we should sort the peers per service and return best peer based on peer score or RTT etc.
	pm.logger.Info("Adding peer to service slots", logging.HostID("peer", peerID),
		zap.String("service", string(proto)))
	pm.serviceSlots[proto] = append(pm.serviceSlots[proto], peerID)
}

// SelectPeer is used to return a random peer that supports a given protocol.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
func (pm *PeerManager) SelectPeer(proto protocol.ID, specificPeers []peer.ID, logger *zap.Logger) (peer.ID, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?

	filteredPeers, err := utils.FilterPeersByProto(pm.host, specificPeers, proto)
	if err != nil {
		return "", err
	}
	if proto == WakuRelayIDv200 {
		return utils.SelectRandomPeer(filteredPeers, pm.logger)
	}

	//Try to fetch from serviceSlot
	peerIDs, ok := pm.serviceSlots[proto]
	if ok || len(peerIDs) > 0 {
		filteredPeers = peerIDs
	}

	return utils.SelectRandomPeer(filteredPeers, pm.logger)
}
