package peermanager

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	wps "github.com/waku-org/go-waku/waku/v2/peerstore"
	waku_proto "github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

type PeerSet map[peer.ID]struct{}

func PeerInSet(peers PeerSet, peer peer.ID) bool {
	if len(peers) > 0 {
		if _, ok := peers[peer]; ok {
			return true
		}
	}
	return false
}

// SelectPeerByContentTopic is used to return a random peer that supports a given protocol for given contentTopic.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol and contentTopic, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
func (pm *PeerManager) SelectPeerByContentTopics(proto protocol.ID, contentTopics []string, specificPeers ...peer.ID) (peer.ID, error) {
	pubsubTopics := []string{}
	for _, cTopic := range contentTopics {
		pubsubTopic, err := waku_proto.GetPubSubTopicFromContentTopic(cTopic)
		if err != nil {
			pm.logger.Debug("selectPeer: failed to get contentTopic from pubsubTopic", zap.String("contentTopic", cTopic))
			return "", err
		}
		pubsubTopics = append(pubsubTopics, pubsubTopic)
	}
	peers, err := pm.SelectPeers(PeerSelectionCriteria{PubsubTopics: pubsubTopics, Proto: proto, SpecificPeers: specificPeers})
	if err != nil {
		return "", err
	}
	return peers[0], nil
}

// SelectRandom is used to return a random peer that supports a given protocol.
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the service slot.
// If a peer cannot be found in the service slot, a peer will be selected from node peerstore
// if pubSubTopic is specified, peer is selected from list that support the pubSubTopic
func (pm *PeerManager) SelectRandom(criteria PeerSelectionCriteria) (peer.IDSlice, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?

	peerIDs, err := pm.selectServicePeer(criteria)
	if err == nil && len(peerIDs) == criteria.MaxPeers {
		return maps.Keys(peerIDs), nil
	}
	if !errors.Is(err, utils.ErrNoPeersAvailable) {
		pm.logger.Debug("could not retrieve random peer from slot",
			zap.String("protocol", string(criteria.Proto)),
			zap.Strings("pubsubTopics", criteria.PubsubTopics),
			zap.Error(err))
		return nil, err
	}
	if len(peerIDs) == 0 {
		peerIDs = make(PeerSet)
	}

	// if not found in serviceSlots or proto == WakuRelayIDv200 (service slots don't support WakuRelayIDv200)
	peerSet := criteria.SpecificPeers
	if len(peerSet) == 0 {
		peerSet = pm.host.Peerstore().Peers()
	}
	filteredPeers, err := pm.FilterPeersByProto(peerSet, criteria.ExcludePeers, criteria.Proto)
	if err != nil {
		return nil, err
	}
	if len(filteredPeers) == 0 && criteria.Proto != relay.WakuRelayID_v200 {
		return nil, utils.ErrNoPeersAvailable
	}
	if len(criteria.PubsubTopics) > 0 {
		filteredPeers = pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopics(criteria.PubsubTopics, filteredPeers...)
	}

	// Prefer peers we are already connected to, so we don't churn on dialing
	// stale/unreachable peers learned via discovery or peer-exchange.
	filteredPeers = pm.preferConnectedPeers(filteredPeers)

	//Not passing excludePeers as filterPeers are already considering excluded ones.
	randomPeers, err := selectRandomPeers(filteredPeers, nil, min(criteria.MaxPeers, len(peerIDs)))
	if err != nil && len(peerIDs) == 0 {
		return nil, err
	}

	for tmpPeer := range randomPeers {
		peerIDs[tmpPeer] = struct{}{}
	}
	return maps.Keys(peerIDs), nil
}

func getRandom(filter PeerSet, count int, excludePeers PeerSet) (PeerSet, error) {
	selectedPeers := make(PeerSet)
	for pID := range filter {
		if PeerInSet(excludePeers, pID) {
			continue
		}
		//Map's iterator in golang works using randomness and hence not random function is being used.
		selectedPeers[pID] = struct{}{}
		if len(selectedPeers) == count {
			break
		}
	}
	if len(selectedPeers) == 0 {
		return nil, utils.ErrNoPeersAvailable
	}
	return selectedPeers, nil
}

// preferConnectedPeers returns the subset of the given peers that we are
// currently connected to. If none are connected it returns the input unchanged.
//
// This biases peer selection toward reachable peers (e.g. an already-connected
// fleet/service node) instead of stale or unreachable peers learned via
// discv5/peer-exchange. Without this, selection is connectedness-blind and a
// light node can repeatedly pick dead peers for filter/lightpush, churning on
// failed dials and missing messages while a perfectly good connected peer is
// available (see status-im/status-go#7513).
func (pm *PeerManager) preferConnectedPeers(peers peer.IDSlice) peer.IDSlice {
	connected := make(peer.IDSlice, 0, len(peers))
	for _, p := range peers {
		if pm.host.Network().Connectedness(p) == network.Connected {
			connected = append(connected, p)
		}
	}
	if len(connected) > 0 {
		return connected
	}
	return peers
}

// selects count random peers from list of peers
func selectRandomPeers(peers peer.IDSlice, excludePeers PeerSet, count int) (PeerSet, error) {
	filteredPeerMap := PeerSliceToMap(peers)
	return getRandom(filteredPeerMap, count, excludePeers)
}

func PeerSliceToMap(peers peer.IDSlice) PeerSet {
	peerSet := make(PeerSet, peers.Len())
	for _, peer := range peers {
		peerSet[peer] = struct{}{}
	}
	return peerSet
}

/*
selectServicePeer tries to select peer from serviceSlot first based on criteria.
serviceSlots is a map of peerMap by protocol.ID.
- Slots are created automatically in getPeers.
- Slots are not created for relay.WakuRelayID_v200.
Therefore for Relay protocol, selectServicePeer will always return ErrNoPeersAvailable.

If there is no pubsubTopics criteria, a random peer from the selected slot is returned.
Otherwise, peers from the slot are filtered by pubsubTopics and random peers are selected from the filtered list.

If no peer is found in the slot, on-demand discovery is triggered for the given pubsubTopics and protocol.
The function retries once to fetch peers from the slot after discovery.
*/
func (pm *PeerManager) selectServicePeer(criteria PeerSelectionCriteria) (PeerSet, error) {
	peers := make(PeerSet)
	var err error
	for retryCnt := 0; retryCnt < 1; retryCnt++ {
		//Try to fetch from serviceSlot
		slot := pm.serviceSlots.getPeers(criteria.Proto)
		if slot == nil {
			continue
		}
		if len(criteria.PubsubTopics) == 0 || (len(criteria.PubsubTopics) == 1 && criteria.PubsubTopics[0] == "") {
			return slot.getRandom(criteria.MaxPeers, criteria.ExcludePeers)
		}

		//PubsubTopic based selection
		slot.mu.RLock()
		keys := make([]peer.ID, 0, len(slot.m))
		for peerID := range slot.m {
			if PeerInSet(criteria.ExcludePeers, peerID) {
				continue
			}
			keys = append(keys, peerID)
		}
		slot.mu.RUnlock()
		selectedPeers := pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopics(criteria.PubsubTopics, keys...)
		// Prefer already-connected service peers over stale/unreachable ones.
		selectedPeers = pm.preferConnectedPeers(selectedPeers)
		tmpPeers, err := selectRandomPeers(selectedPeers, criteria.ExcludePeers, criteria.MaxPeers)
		for tmpPeer := range tmpPeers {
			peers[tmpPeer] = struct{}{}
		}
		if err == nil && len(peers) == criteria.MaxPeers {
			return peers, nil
		}

		//Trigger on-demand discovery for this topic and connect to peer immediately.
		//For now discover at least 1 peer for the criteria
		pm.logger.Debug("discovering peers by pubsubTopic", zap.Strings("pubsubTopics", criteria.PubsubTopics))
		pm.discoverPeersByPubsubTopics(criteria.PubsubTopics, criteria.Proto, criteria.Ctx, 1)

		//Try to fetch peers again.
	}
	if len(peers) == 0 {
		pm.logger.Debug("could not retrieve random peer from slot", zap.Error(err))
	}
	return peers, utils.ErrNoPeersAvailable
}

// PeerSelectionCriteria is the selection Criteria that is used by PeerManager to select peers.
type PeerSelectionCriteria struct {
	SelectionType PeerSelection   `json:"selectionType"`
	Proto         protocol.ID     `json:"protocolId"`
	PubsubTopics  []string        `json:"pubsubTopics"`
	SpecificPeers peer.IDSlice    `json:"specificPeers"`
	MaxPeers      int             `json:"maxPeerCount"`
	Ctx           context.Context `json:"-"`
	ExcludePeers  PeerSet         `json:"excludePeers"`
}

func (psc PeerSelectionCriteria) String() string {
	pscJson, err := json.Marshal(psc)
	if err != nil {
		return ""
	}
	return string(pscJson)
}

// SelectPeers selects a peer based on selectionType specified.
// Context is required only in case of selectionType set to LowestRTT
func (pm *PeerManager) SelectPeers(criteria PeerSelectionCriteria) (peer.IDSlice, error) {
	if criteria.MaxPeers == 0 {
		criteria.MaxPeers = 1
	}
	excPeers := maps.Keys(criteria.ExcludePeers)
	var excPeer peer.ID
	if len(excPeers) > 0 {
		excPeer = excPeers[0]
	}
	pm.logger.Debug("Select Peers", zap.Stringer("selectionCriteria", criteria), zap.Stringer("excludedPeers", excPeer))
	switch criteria.SelectionType {
	case Automatic:
		return pm.SelectRandom(criteria)
	case LowestRTT:
		peerID, err := pm.SelectPeerWithLowestRTT(criteria)
		if err != nil {
			return nil, err
		}
		//TODO: Update this once peer Ping cache PR is merged into this code.
		return []peer.ID{peerID}, nil
	case ProtoPubSubTopicOnly:
		peers, err := pm.SelectPeersByProto(criteria.Proto, criteria.SpecificPeers, criteria.PubsubTopics)
		if err != nil {
			return nil, err
		}
		return peers, nil
	default:
		return nil, errors.New("unknown peer selection type specified")
	}
}

// SelectPeerWithLowestRTT will select a peer that supports a specific protocol with the lowest reply time
// If a list of specific peers is passed, the peer will be chosen from that list assuming
// it supports the chosen protocol, otherwise it will chose a peer from the node peerstore
// TO OPTIMIZE: As of now the peer with lowest RTT is identified when select is called, this should be optimized
// to maintain the RTT as part of peer-scoring and just select based on that.
func (pm *PeerManager) SelectPeerWithLowestRTT(criteria PeerSelectionCriteria) (peer.ID, error) {
	var peers peer.IDSlice
	var err error
	if criteria.Ctx == nil {
		pm.logger.Warn("context is not passed for peerSelectionwithRTT, using background context")
		criteria.Ctx = context.Background()
	}

	if len(criteria.PubsubTopics) == 0 || (len(criteria.PubsubTopics) == 1 && criteria.PubsubTopics[0] == "") {
		peers = pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopics(criteria.PubsubTopics, criteria.SpecificPeers...)
	}

	peers, err = pm.FilterPeersByProto(peers, criteria.ExcludePeers, criteria.Proto)
	if err != nil {
		return "", err
	}

	return pm.rttCache.FastestPeer(criteria.Ctx, peers)
}

// FilterPeersByProto filters list of peers that support specified protocols.
// If specificPeers is nil, all peers in the host's peerStore are considered for filtering.
func (pm *PeerManager) FilterPeersByProto(specificPeers peer.IDSlice, excludePeers PeerSet, proto ...protocol.ID) (peer.IDSlice, error) {
	peerSet := specificPeers
	if len(peerSet) == 0 {
		peerSet = pm.host.Peerstore().Peers()
	}
	var peers peer.IDSlice
	for _, peer := range peerSet {
		protocols, err := pm.host.Peerstore().SupportsProtocols(peer, proto...)
		if err != nil {
			return nil, err
		}
		if len(protocols) > 0 {
			//Maybe we can optimize below set of statements a better way??
			if PeerInSet(excludePeers, peer) {
				continue
			}
			peers = append(peers, peer)
		}
	}
	return peers, nil
}

func (pm *PeerManager) SelectPeersByProto(protocol protocol.ID, specificPeers peer.IDSlice, pubsubTopics []string) (peer.IDSlice, error) {
	var selectedPeers peer.IDSlice
	selectedPeers, err := pm.FilterPeersByProto(specificPeers, nil, protocol)
	if err != nil {
		return nil, err
	}
	selectedPeers = pm.host.Peerstore().(wps.WakuPeerstore).PeersByPubSubTopics(pubsubTopics, selectedPeers...)
	if len(selectedPeers) == 0 {
		return nil, utils.ErrNoPeersAvailable
	}
	return selectedPeers, nil
}
