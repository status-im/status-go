package discv5

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-discover/discover"
	"github.com/waku-org/go-waku/logging"
	v2 "github.com/waku-org/go-waku/waku/v2"
	"github.com/waku-org/go-waku/waku/v2/metrics"
	"github.com/waku-org/go-waku/waku/v2/peers"
	"github.com/waku-org/go-waku/waku/v2/protocol/enr"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

var ErrNoDiscV5Listener = errors.New("no discv5 listener")

type PeerConnector interface {
	Subscribe(context.Context, <-chan v2.PeerData)
}

type DiscoveryV5 struct {
	params    *discV5Parameters
	host      host.Host
	config    discover.Config
	udpAddr   *net.UDPAddr
	listener  *discover.UDPv5
	localnode *enode.LocalNode

	peerConnector PeerConnector
	peerCh        chan v2.PeerData
	NAT           nat.Interface

	log *zap.Logger

	started atomic.Bool
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
}

type discV5Parameters struct {
	autoUpdate    bool
	autoFindPeers bool
	bootnodes     []*enode.Node
	udpPort       uint
	advertiseAddr []multiaddr.Multiaddr
	loopPredicate func(*enode.Node) bool
}

type DiscoveryV5Option func(*discV5Parameters)

var protocolID = [6]byte{'d', '5', 'w', 'a', 'k', 'u'}

func WithAutoUpdate(autoUpdate bool) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.autoUpdate = autoUpdate
	}
}

func WithBootnodes(bootnodes []*enode.Node) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.bootnodes = bootnodes
	}
}

func WithAdvertiseAddr(addr []multiaddr.Multiaddr) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.advertiseAddr = addr
	}
}

func WithUDPPort(port uint) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.udpPort = port
	}
}

func WithPredicate(predicate func(*enode.Node) bool) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.loopPredicate = predicate
	}
}

func WithAutoFindPeers(find bool) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.autoFindPeers = find
	}
}

func DefaultOptions() []DiscoveryV5Option {
	return []DiscoveryV5Option{
		WithUDPPort(9000),
		WithAutoFindPeers(true),
	}
}

func NewDiscoveryV5(priv *ecdsa.PrivateKey, localnode *enode.LocalNode, peerConnector PeerConnector, log *zap.Logger, opts ...DiscoveryV5Option) (*DiscoveryV5, error) {
	params := new(discV5Parameters)
	optList := DefaultOptions()
	optList = append(optList, opts...)
	for _, opt := range optList {
		opt(params)
	}

	logger := log.Named("discv5")

	var NAT nat.Interface = nil
	if params.advertiseAddr == nil {
		NAT = nat.Any()
	}

	return &DiscoveryV5{
		params:        params,
		peerConnector: peerConnector,
		NAT:           NAT,
		wg:            &sync.WaitGroup{},
		localnode:     localnode,
		config: discover.Config{
			PrivateKey: priv,
			Bootnodes:  params.bootnodes,
			V5Config: discover.V5Config{
				ProtocolID: &protocolID,
			},
		},
		udpAddr: &net.UDPAddr{
			IP:   net.IPv4zero,
			Port: int(params.udpPort),
		},
		log: logger,
	}, nil
}

func (d *DiscoveryV5) Node() *enode.Node {
	return d.localnode.Node()
}

func (d *DiscoveryV5) listen(ctx context.Context) error {
	conn, err := net.ListenUDP("udp", d.udpAddr)
	if err != nil {
		return err
	}

	d.udpAddr = conn.LocalAddr().(*net.UDPAddr)

	if d.NAT != nil && !d.udpAddr.IP.IsLoopback() {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			nat.Map(d.NAT, ctx.Done(), "udp", d.udpAddr.Port, d.udpAddr.Port, "go-waku discv5 discovery")
		}()

	}

	d.localnode.SetFallbackUDP(d.udpAddr.Port)

	listener, err := discover.ListenV5(conn, d.localnode, d.config)
	if err != nil {
		return err
	}

	d.listener = listener

	d.log.Info("started Discovery V5",
		zap.Stringer("listening", d.udpAddr),
		logging.TCPAddr("advertising", d.localnode.Node().IP(), d.localnode.Node().TCP()))
	d.log.Info("Discovery V5: discoverable ENR ", logging.ENode("enr", d.localnode.Node()))

	return nil
}

// Sets the host to be able to mount or consume a protocol
func (d *DiscoveryV5) SetHost(h host.Host) {
	d.host = h
}

// only works if the discovery v5 hasn't been started yet.
func (d *DiscoveryV5) Start(ctx context.Context) error {
	// compare and swap sets the discovery v5 to `started` state
	// and prevents multiple calls to the start method by being atomic.
	if !d.started.CompareAndSwap(false, true) {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	d.peerCh = make(chan v2.PeerData)
	d.peerConnector.Subscribe(ctx, d.peerCh)

	err := d.listen(ctx)
	if err != nil {
		return err
	}

	if d.params.autoFindPeers {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			d.runDiscoveryV5Loop(ctx)
		}()
	}

	return nil
}

func (d *DiscoveryV5) SetBootnodes(nodes []*enode.Node) error {
	if d.listener == nil {
		return ErrNoDiscV5Listener
	}

	return d.listener.SetFallbackNodes(nodes)
}

// only works if the discovery v5 is in running state
// so we can assume that cancel method is set
func (d *DiscoveryV5) Stop() {
	if !d.started.CompareAndSwap(true, false) { // if Discoveryv5 is running, set started to false
		return
	}

	d.cancel()

	if d.listener != nil {
		d.listener.Close()
		d.listener = nil
		d.log.Info("stopped Discovery V5")
	}

	d.wg.Wait()

	defer func() {
		if r := recover(); r != nil {
			d.log.Info("recovering from panic and quitting")
		}
	}()

	close(d.peerCh)
}

/*
func isWakuNode(node *enode.Node) bool {
	enrField := new(utils.WakuEnrBitfield)
	if err := node.Record().Load(enr.WithEntry(utils.WakuENRField, &enrField)); err != nil {
		if !enr.IsNotFound(err) {
			utils.Logger().Named("discv5").Error("could not retrieve port for enr ", zap.Any("node", node))
		}
		return false
	}

	if enrField != nil {
		return *enrField != uint8(0)
	}

	return false
}
*/

func evaluateNode(node *enode.Node) bool {
	if node == nil {
		return false
	}

	//  TODO: consider node filtering based on ENR; we do not filter based on ENR in the first waku discv5 beta stage
	/*if !isWakuNode(node) {
		return false
	}*/

	_, err := enr.EnodeToPeerInfo(node)

	if err != nil {
		metrics.RecordDiscV5Error(context.Background(), "peer_info_failure")
		utils.Logger().Named("discv5").Error("obtaining peer info from enode", logging.ENode("enr", node), zap.Error(err))
		return false
	}

	return true
}

// get random nodes from DHT via discv5 listender
// used for caching enr address in peerExchange
// used for connecting to peers in discovery_connector
func (d *DiscoveryV5) Iterator() (enode.Iterator, error) {
	if d.listener == nil {
		return nil, ErrNoDiscV5Listener
	}

	iterator := enode.Filter(d.listener.RandomNodes(), evaluateNode)
	if d.params.loopPredicate != nil {
		return enode.Filter(iterator, d.params.loopPredicate), nil
	} else {
		return iterator, nil
	}
}

func (d *DiscoveryV5) FindPeersWithPredicate(ctx context.Context, predicate func(*enode.Node) bool) (enode.Iterator, error) {
	if d.listener == nil {
		return nil, ErrNoDiscV5Listener
	}

	iterator := enode.Filter(d.listener.RandomNodes(), evaluateNode)
	if predicate != nil {
		iterator = enode.Filter(iterator, predicate)
	}

	return iterator, nil
}

func (d *DiscoveryV5) FindPeersWithShard(ctx context.Context, cluster, index uint16) (enode.Iterator, error) {
	if d.listener == nil {
		return nil, ErrNoDiscV5Listener
	}

	iterator := enode.Filter(d.listener.RandomNodes(), evaluateNode)

	predicate := func(node *enode.Node) bool {
		rs, err := enr.RelaySharding(node.Record())
		if err != nil || rs == nil {
			return false
		}
		return rs.Contains(cluster, index)
	}

	return enode.Filter(iterator, predicate), nil
}

func (d *DiscoveryV5) Iterate(ctx context.Context, iterator enode.Iterator, onNode func(*enode.Node, peer.AddrInfo) error) {
	defer iterator.Close()

	for iterator.Next() { // while next exists, run for loop
		_, addresses, err := enr.Multiaddress(iterator.Node())
		if err != nil {
			metrics.RecordDiscV5Error(context.Background(), "peer_info_failure")
			d.log.Error("extracting multiaddrs from enr", zap.Error(err))
			continue
		}

		peerAddrs, err := peer.AddrInfosFromP2pAddrs(addresses...)
		if err != nil {
			metrics.RecordDiscV5Error(context.Background(), "peer_info_failure")
			d.log.Error("converting multiaddrs to addrinfos", zap.Error(err))
			continue
		}

		if len(peerAddrs) != 0 {
			err := onNode(iterator.Node(), peerAddrs[0])
			if err != nil {
				d.log.Error("processing node", zap.Error(err))
			}
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// Iterates over the nodes found via discv5 belonging to the node's current shard, and sends them to peerConnector
func (d *DiscoveryV5) peerLoop(ctx context.Context) error {
	iterator, err := d.Iterator()
	if err != nil {
		metrics.RecordDiscV5Error(context.Background(), "iterator_failure")
		return fmt.Errorf("obtaining iterator: %w", err)
	}

	iterator = enode.Filter(iterator, func(n *enode.Node) bool {
		localRS, err := enr.RelaySharding(d.localnode.Node().Record())
		if err != nil {
			return false
		}

		if localRS == nil { // No shard registered, so no need to check for shards
			return true
		}

		nodeRS, err := enr.RelaySharding(n.Record())
		if err != nil || nodeRS == nil {
			return false
		}

		if nodeRS.Cluster != localRS.Cluster {
			return false
		}

		// Contains any
		for _, idx := range localRS.Indices {
			if nodeRS.Contains(localRS.Cluster, idx) {
				return true
			}
		}

		return false
	})

	defer iterator.Close()

	d.Iterate(ctx, iterator, func(n *enode.Node, p peer.AddrInfo) error {
		peer := v2.PeerData{
			Origin:   peers.Discv5,
			AddrInfo: p,
			ENR:      n,
		}

		select {
		case d.peerCh <- peer:
		case <-ctx.Done():
			return nil
		}

		return nil
	})

	return nil
}

func (d *DiscoveryV5) runDiscoveryV5Loop(ctx context.Context) {

restartLoop:
	for {
		err := d.peerLoop(ctx)
		if err != nil {
			d.log.Debug("iterating discv5", zap.Error(err))
		}

		t := time.NewTimer(5 * time.Second)
		select {
		case <-t.C:
			t.Stop()
		case <-ctx.Done():
			t.Stop()
			break restartLoop
		}
	}
	d.log.Warn("Discv5 loop stopped")
}

func (d *DiscoveryV5) IsStarted() bool {
	return d.started.Load()
}
