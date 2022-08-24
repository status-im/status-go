package discv5

import (
	"context"
	"crypto/ecdsa"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/status-im/go-discover/discover"
	"github.com/status-im/go-waku/logging"
	"github.com/status-im/go-waku/waku/v2/utils"
	"go.uber.org/zap"
)

type DiscoveryV5 struct {
	sync.Mutex

	discovery.Discovery

	params    *discV5Parameters
	host      host.Host
	config    discover.Config
	udpAddr   *net.UDPAddr
	listener  *discover.UDPv5
	localnode *enode.LocalNode
	NAT       nat.Interface
	quit      chan struct{}

	log *zap.Logger

	wg *sync.WaitGroup

	peerCache peerCache
}

type peerCache struct {
	sync.RWMutex
	recs map[peer.ID]peerRecord
	rng  *rand.Rand
}

type peerRecord struct {
	expire int64
	peer   peer.AddrInfo
}

type discV5Parameters struct {
	autoUpdate    bool
	bootnodes     []*enode.Node
	udpPort       int
	advertiseAddr *net.IP
}

type DiscoveryV5Option func(*discV5Parameters)

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

func WithAdvertiseAddr(addr net.IP) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.advertiseAddr = &addr
	}
}

func WithUDPPort(port int) DiscoveryV5Option {
	return func(params *discV5Parameters) {
		params.udpPort = port
	}
}

func DefaultOptions() []DiscoveryV5Option {
	return []DiscoveryV5Option{
		WithUDPPort(9000),
	}
}

func NewDiscoveryV5(host host.Host, priv *ecdsa.PrivateKey, localnode *enode.LocalNode, log *zap.Logger, opts ...DiscoveryV5Option) (*DiscoveryV5, error) {
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
		host:   host,
		params: params,
		NAT:    NAT,
		wg:     &sync.WaitGroup{},
		peerCache: peerCache{
			rng:  rand.New(rand.NewSource(rand.Int63())),
			recs: make(map[peer.ID]peerRecord),
		},
		localnode: localnode,
		config: discover.Config{
			PrivateKey: priv,
			Bootnodes:  params.bootnodes,
			ValidNodeFn: func(n enode.Node) bool {
				// TODO: track https://github.com/status-im/nim-waku/issues/770 for improvements over validation func
				return evaluateNode(&n)
			},
			V5Config: discover.V5Config{
				ProtocolID: [6]byte{'d', '5', 'w', 'a', 'k', 'u'},
			},
		},
		udpAddr: &net.UDPAddr{
			IP:   net.IPv4zero,
			Port: params.udpPort,
		},
		log: logger,
	}, nil
}

func (d *DiscoveryV5) listen() error {
	conn, err := net.ListenUDP("udp", d.udpAddr)
	if err != nil {
		return err
	}

	d.udpAddr = conn.LocalAddr().(*net.UDPAddr)
	if d.NAT != nil && !d.udpAddr.IP.IsLoopback() {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			nat.Map(d.NAT, d.quit, "udp", d.udpAddr.Port, d.udpAddr.Port, "go-waku discv5 discovery")
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

func (d *DiscoveryV5) Start() error {
	d.Lock()
	defer d.Unlock()

	d.wg.Wait() // Waiting for other go routines to stop

	d.quit = make(chan struct{}, 1)

	err := d.listen()
	if err != nil {
		return err
	}

	return nil
}

func (d *DiscoveryV5) Stop() {
	d.Lock()
	defer d.Unlock()

	close(d.quit)

	d.listener.Close()
	d.listener = nil

	d.log.Info("stopped Discovery V5")

	d.wg.Wait()
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

func hasTCPPort(node *enode.Node) bool {
	enrTCP := new(enr.TCP)
	if err := node.Record().Load(enr.WithEntry(enrTCP.ENRKey(), enrTCP)); err != nil {
		if !enr.IsNotFound(err) {
			utils.Logger().Named("discv5").Error("retrieving port for enr", logging.ENode("enr", node))
		}
		return false
	}

	return true
}

func evaluateNode(node *enode.Node) bool {
	if node == nil || node.IP() == nil {
		return false
	}

	//  TODO: consider node filtering based on ENR; we do not filter based on ENR in the first waku discv5 beta stage
	if /*!isWakuNode(node) ||*/ !hasTCPPort(node) {
		return false
	}

	_, err := utils.EnodeToPeerInfo(node)

	if err != nil {
		utils.Logger().Named("discv5").Error("obtaining peer info from enode", logging.ENode("enr", node), zap.Error(err))
		return false
	}

	return true
}

func (d *DiscoveryV5) Advertise(ctx context.Context, ns string, opts ...discovery.Option) (time.Duration, error) {
	// Get options
	var options discovery.Options
	err := options.Apply(opts...)
	if err != nil {
		return 0, err
	}

	// TODO: once discv5 spec introduces capability and topic discovery, implement this function

	return 20 * time.Minute, nil
}

func (d *DiscoveryV5) iterate(ctx context.Context, iterator enode.Iterator, limit int, doneCh chan struct{}) {
	defer d.wg.Done()

	for {
		if len(d.peerCache.recs) >= limit {
			break
		}

		if ctx.Err() != nil {
			break
		}

		exists := iterator.Next()
		if !exists {
			break
		}

		addresses, err := utils.Multiaddress(iterator.Node())
		if err != nil {
			d.log.Error("extracting multiaddrs from enr", zap.Error(err))
			continue
		}

		peerAddrs, err := peer.AddrInfosFromP2pAddrs(addresses...)
		if err != nil {
			d.log.Error("converting multiaddrs to addrinfos", zap.Error(err))
			continue
		}

		for _, p := range peerAddrs {
			d.peerCache.recs[p.ID] = peerRecord{
				expire: time.Now().Unix() + 3600, // Expires in 1hr
				peer:   p,
			}
		}

	}

	close(doneCh)
}

func (d *DiscoveryV5) removeExpiredPeers() int {
	// Remove all expired entries from cache
	currentTime := time.Now().Unix()
	newCacheSize := len(d.peerCache.recs)

	for p := range d.peerCache.recs {
		rec := d.peerCache.recs[p]
		if rec.expire < currentTime {
			newCacheSize--
			delete(d.peerCache.recs, p)
		}
	}

	return newCacheSize
}

func (d *DiscoveryV5) FindPeers(ctx context.Context, topic string, opts ...discovery.Option) (<-chan peer.AddrInfo, error) {
	// Get options
	var options discovery.Options
	err := options.Apply(opts...)
	if err != nil {
		return nil, err
	}

	const maxLimit = 100
	limit := options.Limit
	if limit == 0 || limit > maxLimit {
		limit = maxLimit
	}

	// We are ignoring the topic. Future versions might use a map[string]*peerCache instead where the string represents the pubsub topic

	d.peerCache.Lock()
	defer d.peerCache.Unlock()

	cacheSize := d.removeExpiredPeers()

	// Discover new records if we don't have enough
	if cacheSize < limit && d.listener != nil {
		d.Lock()

		iterator := d.listener.RandomNodes()
		iterator = enode.Filter(iterator, evaluateNode)
		defer iterator.Close()

		doneCh := make(chan struct{})

		d.wg.Add(1)
		go d.iterate(ctx, iterator, limit, doneCh)

		select {
		case <-ctx.Done():
		case <-doneCh:
		}

		d.Unlock()
	}

	// Randomize and fill channel with available records
	count := len(d.peerCache.recs)
	if limit < count {
		count = limit
	}

	chPeer := make(chan peer.AddrInfo, count)

	perm := d.peerCache.rng.Perm(len(d.peerCache.recs))[0:count]
	permSet := make(map[int]int)
	for i, v := range perm {
		permSet[v] = i
	}

	sendLst := make([]*peer.AddrInfo, count)
	iter := 0
	for k := range d.peerCache.recs {
		if sendIndex, ok := permSet[iter]; ok {
			peerInfo := d.peerCache.recs[k].peer
			sendLst[sendIndex] = &peerInfo
		}
		iter++
	}

	for _, send := range sendLst {
		chPeer <- *send
	}

	close(chPeer)

	return chPeer, err
}
