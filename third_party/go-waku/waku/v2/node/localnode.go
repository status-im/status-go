package node

import (
	"context"
	"errors"
	"net"
	"strconv"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	madns "github.com/multiformats/go-multiaddr-dns"
	"go.uber.org/zap"

	ndoeutils "github.com/waku-org/go-waku/waku/v2/node/utils"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func (w *WakuNode) updateLocalNode() error {
	w.localNodeMutex.Lock()
	defer w.localNodeMutex.Unlock()
	return enr.UpdateLocalNode(w.log, w.localNode, &w.localNodeParams)
}

func decapsulateP2P(addr ma.Multiaddr) (ma.Multiaddr, error) {
	p2p, err := addr.ValueForProtocol(ma.P_P2P)
	if err != nil {
		return nil, err
	}

	p2pAddr, err := ma.NewMultiaddr("/p2p/" + p2p)
	if err != nil {
		return nil, err
	}

	addr = addr.Decapsulate(p2pAddr)

	return addr, nil
}

func decapsulateCircuitRelayAddr(ctx context.Context, addr ma.Multiaddr) (ma.Multiaddr, error) {
	_, err := addr.ValueForProtocol(ma.P_CIRCUIT)
	if err != nil {
		return nil, errors.New("not a circuit relay address")
	}

	// We remove the node's multiaddress from the addr
	addr, _ = ma.SplitFunc(addr, func(c ma.Component) bool {
		return c.Protocol().Code == ma.P_CIRCUIT
	})

	// If the multiaddress is a dns4 address, we resolve it
	addrs, err := madns.DefaultResolver.Resolve(ctx, addr)
	if err != nil {
		return nil, err
	}

	if len(addrs) > 0 {
		return addrs[0], nil
	}

	return addr, nil
}

func selectWSListenAddresses(addresses []ma.Multiaddr) ([]ma.Multiaddr, error) {
	var result []ma.Multiaddr
	for _, addr := range addresses {
		// It's a p2p-circuit address. We dont use these at this stage yet
		_, err := addr.ValueForProtocol(ma.P_CIRCUIT)
		if err == nil {
			continue
		}

		_, noWS := addr.ValueForProtocol(ma.P_WSS)
		_, noWSS := addr.ValueForProtocol(ma.P_WS)
		if noWS != nil && noWSS != nil { // Neither WS or WSS found
			continue
		}

		addr, err = decapsulateP2P(addr)
		if err == nil {
			result = append(result, addr)
		}
	}

	return result, nil
}

func selectCircuitRelayListenAddresses(ctx context.Context, addresses []ma.Multiaddr) ([]ma.Multiaddr, error) {
	var result []ma.Multiaddr

	for _, addr := range addresses {
		addr, err := decapsulateCircuitRelayAddr(ctx, addr)
		if err != nil {
			continue
		}

		_, noWS := addr.ValueForProtocol(ma.P_WSS)
		_, noWSS := addr.ValueForProtocol(ma.P_WS)
		if noWS == nil || noWSS == nil { // WS or WSS found
			continue
		}

		result = append(result, addr)
	}

	return result, nil
}

func filter0Port(addresses []ma.Multiaddr) ([]ma.Multiaddr, error) {
	var result []ma.Multiaddr
	for _, addr := range addresses {
		portStr, err := addr.ValueForProtocol(ma.P_TCP)
		if err != nil && !errors.Is(err, multiaddr.ErrProtocolNotFound) {
			return nil, err
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, err
		}

		if port != 0 {
			result = append(result, addr)
		}
	}

	return result, nil
}

func (w *WakuNode) getENRAddresses(ctx context.Context, addrs []ma.Multiaddr) (extAddr *net.TCPAddr, multiaddr []ma.Multiaddr, err error) {
	extAddr, err = ndoeutils.SelectMostExternalAddress(addrs)
	if err != nil {
		return nil, nil, err
	}

	wssAddrs, err := selectWSListenAddresses(addrs)
	if err != nil {
		return nil, nil, err
	}

	circuitAddrs, err := selectCircuitRelayListenAddresses(ctx, addrs)
	if err != nil {
		return nil, nil, err
	}

	if len(circuitAddrs) != 0 {
		// Node is unreachable, hence why we have circuit relay multiaddr
		// We prefer these instead of any ws/s address
		multiaddr = append(multiaddr, circuitAddrs...)
	} else {
		multiaddr = append(multiaddr, wssAddrs...)
	}

	multiaddr, err = filter0Port(multiaddr)
	if err != nil {
		return nil, nil, err
	}

	return
}

func (w *WakuNode) setupENR(ctx context.Context, addrs []ma.Multiaddr) error {
	ipAddr, multiaddresses, err := w.getENRAddresses(ctx, addrs)
	if err != nil {
		w.log.Error("obtaining external address", zap.Error(err))
		return err
	}

	w.localNodeParams.Multiaddrs = multiaddresses
	w.localNodeParams.IPAddr = ipAddr

	err = w.updateLocalNode()
	if err != nil {
		w.log.Error("updating localnode ENR record", zap.Error(err))
		return err
	}

	if w.Relay() != nil {
		err = w.watchTopicShards(ctx)
		if err != nil {
			return err
		}
	}

	w.enrChangeCh <- struct{}{}

	return nil

}

func (w *WakuNode) SetRelayShards(rs protocol.RelayShards) error {
	w.localNodeParams.RelayShards = rs
	err := w.updateLocalNode()
	if err != nil {
		return err
	}
	return nil
}

func (w *WakuNode) watchTopicShards(ctx context.Context) error {
	if !w.watchingRelayShards.CompareAndSwap(false, true) {
		return nil
	}

	evtRelaySubscribed, err := w.Relay().Events().Subscribe(new(relay.EvtRelaySubscribed))
	if err != nil {
		return err
	}

	evtRelayUnsubscribed, err := w.Relay().Events().Subscribe(new(relay.EvtRelayUnsubscribed))
	if err != nil {
		return err
	}

	w.wg.Add(1)

	go func() {
		defer utils.LogOnPanic()
		defer evtRelaySubscribed.Close()
		defer evtRelayUnsubscribed.Close()
		defer w.wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case <-evtRelayUnsubscribed.Out():
			case <-evtRelaySubscribed.Out():
				topics := w.Relay().Topics()
				rs, err := protocol.TopicsToRelayShards(topics...)
				if err != nil {
					w.log.Warn("could not set ENR shard info", zap.Error(err))
					continue
				}

				if len(rs) > 0 {
					if len(rs) > 1 {
						w.log.Warn("could not set ENR shard info", zap.String("error", "multiple clusters found, use sharded topics within the same cluster"))
						continue
					}
				}

				if len(rs) == 1 {
					w.log.Info("updating advertised relay shards in ENR", zap.Any("newShardInfo", rs[0]))
					if len(rs[0].ShardIDs) != len(topics) {
						w.log.Warn("A mix of named and static shards found. ENR shard will contain only the following shards", zap.Any("shards", rs[0]))
					}

					w.localNodeParams.RelayShards = rs[0]
					err = w.updateLocalNode()
					if err != nil {
						w.log.Warn("could not set ENR shard info", zap.Error(err))
						continue
					}

					w.enrChangeCh <- struct{}{}
				}
			}
		}
	}()

	return nil
}

func (w *WakuNode) registerAndMonitorReachability(ctx context.Context) {
	var myEventSub event.Subscription
	var err error
	if myEventSub, err = w.host.EventBus().Subscribe(new(event.EvtLocalReachabilityChanged)); err != nil {
		w.log.Error("failed to register with libp2p for reachability status", zap.Error(err))
		return
	}
	w.wg.Add(1)
	go func() {
		defer utils.LogOnPanic()
		defer myEventSub.Close()
		defer w.wg.Done()

		for {
			select {
			case evt := <-myEventSub.Out():
				reachability := evt.(event.EvtLocalReachabilityChanged).Reachability
				w.log.Info("Node reachability changed", zap.Stringer("newReachability", reachability))
			case <-ctx.Done():
				return
			}
		}
	}()
}
