package utils

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

var log = logging.Logger("utils")

var ErrNoPeersAvailable = errors.New("no suitable peers found")
var PingServiceNotAvailable = errors.New("ping service not available")

// SelectPeer is used to return a random peer that supports a given protocol.
func SelectPeer(host host.Host, protocolId string) (*peer.ID, error) {
	// @TODO We need to be more strategic about which peers we dial. Right now we just set one on the service.
	// Ideally depending on the query and our set  of peers we take a subset of ideal peers.
	// This will require us to check for various factors such as:
	//  - which topics they track
	//  - latency?
	//  - default store peer?
	var peers peer.IDSlice
	for _, peer := range host.Peerstore().Peers() {
		protocols, err := host.Peerstore().SupportsProtocols(peer, protocolId)
		if err != nil {
			log.Error("error obtaining the protocols supported by peers", err)
			return nil, err
		}

		if len(protocols) > 0 {
			peers = append(peers, peer)
		}
	}

	if len(peers) >= 1 {
		// TODO: proper heuristic here that compares peer scores and selects "best" one. For now a random peer for the given protocol is returned
		return &peers[rand.Intn(len(peers))], nil // nolint: gosec
	}

	return nil, ErrNoPeersAvailable
}

type pingResult struct {
	p   peer.ID
	rtt time.Duration
}

func SelectPeerWithLowestRTT(ctx context.Context, host host.Host, protocolId string) (*peer.ID, error) {
	var peers peer.IDSlice
	for _, peer := range host.Peerstore().Peers() {
		protocols, err := host.Peerstore().SupportsProtocols(peer, protocolId)
		if err != nil {
			log.Error("error obtaining the protocols supported by peers", err)
			return nil, err
		}

		if len(protocols) > 0 {
			peers = append(peers, peer)
		}
	}

	wg := sync.WaitGroup{}
	waitCh := make(chan struct{})
	pingCh := make(chan pingResult, 1000)

	wg.Add(len(peers))

	go func() {
		for _, p := range peers {
			go func(p peer.ID) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				result := <-ping.Ping(ctx, host, p)
				if result.Error == nil {
					pingCh <- pingResult{
						p:   p,
						rtt: result.RTT,
					}
				}
			}(p)
		}
		wg.Wait()
		close(waitCh)
		close(pingCh)
	}()

	select {
	case <-waitCh:
		var min *pingResult
		for p := range pingCh {
			if min == nil {
				min = &p
			} else {
				if p.rtt < min.rtt {
					min = &p
				}
			}
		}
		if min == nil {
			return nil, ErrNoPeersAvailable
		} else {
			return &min.p, nil
		}
	case <-ctx.Done():
		return nil, ErrNoPeersAvailable
	}
}

func EnodeToMultiAddr(node *enode.Node) (ma.Multiaddr, error) {
	peerID, err := peer.IDFromPublicKey(&ECDSAPublicKey{node.Pubkey()})
	if err != nil {
		return nil, err
	}

	return ma.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", node.IP(), node.TCP(), peerID))
}

func EnodeToPeerInfo(node *enode.Node) (*peer.AddrInfo, error) {
	address, err := EnodeToMultiAddr(node)
	if err != nil {
		return nil, err
	}

	return peer.AddrInfoFromP2pAddr(address)
}

func GetENRandIP(addr ma.Multiaddr, privK *ecdsa.PrivateKey) (*enode.Node, *net.TCPAddr, error) {
	ip, err := addr.ValueForProtocol(ma.P_IP4)
	if err != nil {
		return nil, nil, err
	}

	portStr, err := addr.ValueForProtocol(ma.P_TCP)
	if err != nil {
		return nil, nil, err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, nil, err
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, nil, err
	}

	r := &enr.Record{}

	if port > 0 && port <= math.MaxUint16 {
		r.Set(enr.TCP(uint16(port))) // lgtm [go/incorrect-integer-conversion]
	} else {
		return nil, nil, fmt.Errorf("could not set port %d", port)
	}

	r.Set(enr.IP(net.ParseIP(ip)))

	err = enode.SignV4(r, privK)
	if err != nil {
		return nil, nil, err
	}

	node, err := enode.New(enode.ValidSchemes, r)

	return node, tcpAddr, err
}
