package discv5

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/waku-org/go-discover/discover"
	"github.com/waku-org/go-waku/logging"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

var ErrNoDiscV5Listener = errors.New("no discv5 listener")

type DiscoveryV5 struct {
	sync.RWMutex

	params        *discV5Parameters
	host          host.Host
	config        discover.Config
	udpAddr       *net.UDPAddr
	listener      *discover.UDPv5
	localnode     *enode.LocalNode
	peerConnector PeerConnector
	NAT           nat.Interface

	log *zap.Logger

	started bool
	cancel  context.CancelFunc
	wg      *sync.WaitGroup
}

type discV5Parameters struct {
	autoUpdate    bool
	bootnodes     []*enode.Node
	udpPort       uint
	advertiseAddr []multiaddr.Multiaddr
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

func DefaultOptions() []DiscoveryV5Option {
	return []DiscoveryV5Option{
		WithUDPPort(9000),
	}
}

type PeerConnector interface {
	PeerChannel() chan<- peer.AddrInfo
}

func NewDiscoveryV5(host host.Host, priv *ecdsa.PrivateKey, localnode *enode.LocalNode, peerConnector PeerConnector, log *zap.Logger, opts ...DiscoveryV5Option) (*DiscoveryV5, error) {
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
		host:          host,
		peerConnector: peerConnector,
		params:        params,
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

func (d *DiscoveryV5) Start(ctx context.Context) error {
	d.Lock()
	defer d.Unlock()

	d.wg.Wait() // Waiting for any go routines to stop
	ctx, cancel := context.WithCancel(ctx)

	d.cancel = cancel
	d.started = true

	err := d.listen(ctx)
	if err != nil {
		return err
	}

	d.wg.Add(1)
	go d.runDiscoveryV5Loop(ctx)

	return nil
}

func (d *DiscoveryV5) SetBootnodes(nodes []*enode.Node) error {
	if d.listener == nil {
		return ErrNoDiscV5Listener
	}

	return d.listener.SetFallbackNodes(nodes)
}

func (d *DiscoveryV5) Stop() {
	d.Lock()
	defer d.Unlock()

	if d.cancel == nil {
		return
	}

	d.cancel()
	d.started = false

	if d.listener != nil {
		d.listener.Close()
		d.listener = nil
		d.log.Info("stopped Discovery V5")
	}

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

func evaluateNode(node *enode.Node) bool {
	if node == nil {
		return false
	}

	//  TODO: consider node filtering based on ENR; we do not filter based on ENR in the first waku discv5 beta stage
	/*if !isWakuNode(node) {
		return false
	}*/

	_, err := utils.EnodeToPeerInfo(node)

	if err != nil {
		utils.Logger().Named("discv5").Error("obtaining peer info from enode", logging.ENode("enr", node), zap.Error(err))
		return false
	}

	return true
}

func (d *DiscoveryV5) Iterator() (enode.Iterator, error) {
	if d.listener == nil {
		return nil, ErrNoDiscV5Listener
	}

	iterator := d.listener.RandomNodes()
	return enode.Filter(iterator, evaluateNode), nil
}

func (d *DiscoveryV5) iterate(ctx context.Context) error {
	iterator, err := d.Iterator()
	if err != nil {
		return fmt.Errorf("obtaining iterator: %w", err)
	}

	closeCh := make(chan struct{}, 1)
	defer close(closeCh)

	// Closing iterator when context is cancelled or function is returning
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		select {
		case <-ctx.Done():
			iterator.Close()
		case <-closeCh:
			iterator.Close()
		}
	}()

	for {
		if ctx.Err() != nil {
			break
		}

		exists := iterator.Next()
		if !exists {
			break
		}

		_, addresses, err := utils.Multiaddress(iterator.Node())
		if err != nil {
			d.log.Error("extracting multiaddrs from enr", zap.Error(err))
			continue
		}

		peerAddrs, err := peer.AddrInfosFromP2pAddrs(addresses...)
		if err != nil {
			d.log.Error("converting multiaddrs to addrinfos", zap.Error(err))
			continue
		}

		if len(peerAddrs) != 0 {
			select {
			case <-ctx.Done():
				return nil
			case d.peerConnector.PeerChannel() <- peerAddrs[0]:
			}
		}
	}

	return nil
}

func (d *DiscoveryV5) runDiscoveryV5Loop(ctx context.Context) {
	defer d.wg.Done()

	ch := make(chan struct{}, 1)
	ch <- struct{}{} // Initial execution

restartLoop:
	for {
		select {
		case <-ch:
			err := d.iterate(ctx)
			if err != nil {
				d.log.Debug("iterating discv5", zap.Error(err))
				time.Sleep(2 * time.Second)
			}
			ch <- struct{}{}
		case <-ctx.Done():
			close(ch)
			break restartLoop
		}
	}
	d.log.Warn("Discv5 loop stopped")
}

func (d *DiscoveryV5) IsStarted() bool {
	d.RLock()
	defer d.RUnlock()

	return d.started
}
