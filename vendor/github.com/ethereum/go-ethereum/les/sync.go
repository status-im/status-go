// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package les

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/light"

	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/params"
)

var LightEth *LightEthereum = nil
var NodeConfig *params.NodeConfig = nil

const (
	//forceSyncCycle      = 10 * time.Second // Time interval to force syncs, even if few peers are available
	minDesiredPeerCount = 5 // Amount of peers desired to start syncing
)

// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (pm *ProtocolManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	//pm.fetcher.Start()
	//defer pm.fetcher.Stop()
	defer pm.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	//forceSync := time.Tick(forceSyncCycle)
	for {
		select {
		case <-pm.newPeerCh:
			/*			// Make sure we have peers to select from, then sync
						if pm.peers.Len() < minDesiredPeerCount {
							break
						}
						go pm.synchronise(pm.peers.BestPeer())
			*/
		/*case <-forceSync:
		// Force a sync even if not enough peers are present
		go pm.synchronise(pm.peers.BestPeer())
		*/
		case <-pm.noMorePeers:
			return
		}
	}
}

func (pm *ProtocolManager) needToSync(peerHead blockInfo) bool {
	head := pm.blockchain.CurrentHeader()
	currentTd := core.GetTd(pm.chainDb, head.Hash(), head.Number.Uint64())
	return currentTd != nil && peerHead.Td.Cmp(currentTd) > 0
}

// synchronise tries to sync up our local block chain with a remote peer.
func (pm *ProtocolManager) synchronise(peer *peer) {
	// Short circuit if no peers are available
	if peer == nil {
		return
	}

	// Make sure the peer's TD is higher than our own.
	if !pm.needToSync(peer.headBlockInfo()) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	updateChtFromPeer(pm, peer, ctx)
	pm.blockchain.(*light.LightChain).SyncCht(ctx)
	pm.downloader.Synchronise(peer.id, peer.Head(), peer.Td(), downloader.LightSync)
}

// Status.im issue 320: Use GetHeaderProofs to download the latest CHT from a peer.
func updateChtFromPeer(pm *ProtocolManager, peer *peer, ctx context.Context) {
	log.Info("Downloading latest CHT root from peer", "peer.headBlockInfo", peer.headBlockInfo())

	// Formula from lightchain.go:SyncCht: num := cht.Number*ChtFrequency – 1
	var hbl = peer.headBlockInfo()
	var peerHeadBlockNum = hbl.Number
	log.Debug("UpdateChtFromPeer", "peerHeadBlockNum", peerHeadBlockNum)

	var chtnum uint64 = ((peerHeadBlockNum + 1) / light.ChtFrequency) - 1
	var blocknum uint64 = chtnum*light.ChtFrequency - 1
	log.Debug("CHT block values: ", "chtnum", chtnum, "blocknum", blocknum)

	req := &light.ChtRequest{ChtNum: uint64(chtnum), BlockNum: uint64(blocknum)}
	LightEth.LesOdr().Retrieve(ctx, req)
	log.Info("Retrieved latest CHT root from peer, got", "ChtRoot", common.ToHex(req.ChtRoot.Bytes()))

	LightEth.WriteTrustedCht(light.TrustedCht{
		Number: chtnum,
		Root:   req.ChtRoot,
	})
	log.Info("Added trusted CHT",
		"develop", NodeConfig.DevMode,
		"number", chtnum, "hash", common.ToHex(req.ChtRoot.Bytes()))

	log.Info("Sanity check: can download some very old header?")
	var sanityBlock uint64 = blocknum / 100
	sanityHeader, err := light.GetHeaderByNumber(ctx, LightEth.LesOdr(), sanityBlock)
	log.Info("Sanity check result:", "sanityHeader.Number", sanityHeader.Number, "err", err)
}
