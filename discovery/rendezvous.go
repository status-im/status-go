package discovery

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/rendezvous"
)

const (
	registrationPeriod = 10 * time.Second
	requestTimeout     = 5 * time.Second
	bucketSize         = 10
)

var (
	errNodeIsNil          = errors.New("node cannot be nil")
	errIdentityIsNil      = errors.New("identity cannot be nil")
	errDiscoveryIsStopped = errors.New("discovery is stopped")
)

func NewRendezvous(servers []ma.Multiaddr, identity *ecdsa.PrivateKey, node *enode.Node) (*Rendezvous, error) {
	r := new(Rendezvous)
	r.node = node
	r.identity = identity
	r.servers = servers
	r.registrationPeriod = registrationPeriod
	r.bucketSize = bucketSize
	return r, nil
}

func NewRendezvousWithENR(servers []ma.Multiaddr, record enr.Record) *Rendezvous {
	r := new(Rendezvous)
	r.servers = servers
	r.registrationPeriod = registrationPeriod
	r.bucketSize = bucketSize
	r.record = &record
	return r
}

// Rendezvous is an implementation of discovery interface that uses
// rendezvous client.
type Rendezvous struct {
	mu     sync.RWMutex
	client *rendezvous.Client

	// Root context is used to cancel running requests
	// when Rendezvous is stopped.
	rootCtx       context.Context
	cancelRootCtx context.CancelFunc

	servers            []ma.Multiaddr
	registrationPeriod time.Duration
	bucketSize         int
	node               *enode.Node
	identity           *ecdsa.PrivateKey

	recordMu sync.Mutex
	record   *enr.Record // record is set directly if rendezvous is used in proxy mode
}

func (r *Rendezvous) Running() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client != nil
}

// Start creates client with ephemeral identity.
func (r *Rendezvous) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	client, err := rendezvous.NewEphemeral()
	if err != nil {
		return err
	}
	r.client = &client
	r.rootCtx, r.cancelRootCtx = context.WithCancel(context.Background())
	return nil
}

// Stop removes client reference.
func (r *Rendezvous) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.client == nil {
		return nil
	}
	r.cancelRootCtx()
	if err := r.client.Close(); err != nil {
		return err
	}
	r.client = nil
	return nil
}

func (r *Rendezvous) MakeRecord() (record enr.Record, err error) {
	r.recordMu.Lock()
	defer r.recordMu.Unlock()
	if r.record != nil {
		return *r.record, nil
	}
	if r.node == nil {
		return record, errNodeIsNil
	}
	if r.identity == nil {
		return record, errIdentityIsNil
	}
	record.Set(enr.IP(r.node.IP()))
	record.Set(enr.TCP(r.node.TCP()))
	record.Set(enr.UDP(r.node.UDP()))
	// public key is added to ENR when ENR is signed
	if err := enode.SignV4(&record, r.identity); err != nil {
		return record, err
	}
	r.record = &record
	return record, nil
}

func (r *Rendezvous) register(topic string, record enr.Record) error {
	srv := r.servers[rand.Intn(len(r.servers))]
	ctx, cancel := context.WithTimeout(r.rootCtx, requestTimeout)
	defer cancel()

	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.client == nil {
		return errDiscoveryIsStopped
	}
	err := r.client.Register(ctx, srv, topic, record, r.registrationPeriod)
	if err != nil {
		log.Error("error registering", "topic", topic, "rendezvous server", srv, "err", err)
	}
	return err
}

// Register renews registration in the specified server.
func (r *Rendezvous) Register(topic string, stop chan struct{}) error {
	record, err := r.MakeRecord()
	if err != nil {
		return err
	}
	// sending registration more often than the whole registraton period
	// will ensure that it won't be accidentally removed
	ticker := time.NewTicker(r.registrationPeriod / 2)
	defer ticker.Stop()

	if err := r.register(topic, record); err == context.Canceled {
		return err
	}

	for {
		select {
		case <-stop:
			return nil
		case <-ticker.C:
			if err := r.register(topic, record); err == context.Canceled {
				return err
			} else if err == errDiscoveryIsStopped {
				return nil
			}
		}
	}
}

func (r *Rendezvous) discoverRequest(srv ma.Multiaddr, topic string) ([]enr.Record, error) {
	ctx, cancel := context.WithTimeout(r.rootCtx, requestTimeout)
	defer cancel()
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.client == nil {
		return nil, errDiscoveryIsStopped
	}
	return r.client.Discover(ctx, srv, topic, r.bucketSize)
}

// Discover will search for new records every time period fetched from period channel.
func (r *Rendezvous) Discover(
	topic string, period <-chan time.Duration, found chan<- *discv5.Node, lookup chan<- bool,
) error {
	ticker := time.NewTicker(<-period)
	for {
		select {
		case newPeriod, ok := <-period:
			ticker.Stop()
			if !ok {
				return nil
			}
			ticker = time.NewTicker(newPeriod)
		case <-ticker.C:
			srv := r.servers[rand.Intn(len(r.servers))]
			records, err := r.discoverRequest(srv, topic)
			if err == context.Canceled {
				return err
			} else if err == errDiscoveryIsStopped {
				return nil
			} else if err != nil {
				log.Debug("error fetching records", "topic", topic, "rendezvous server", srv, "err", err)
			} else {
				for i := range records {
					n, err := enrToNode(records[i])
					log.Debug("converted enr to", "ENODE", n.String())
					if err != nil {
						log.Warn("error converting enr record to node", "err", err)

					} else {
						select {
						case found <- n:
						case newPeriod, ok := <-period:
							// closing a period channel is a signal to producer that consumer exited
							ticker.Stop()
							if !ok {
								return nil
							}
							ticker = time.NewTicker(newPeriod)
						}
					}
				}
			}
		}
	}
}

func enrToNode(record enr.Record) (*discv5.Node, error) {
	var (
		key    enode.Secp256k1
		ip     enr.IP
		tport  enr.TCP
		uport  enr.UDP
		nodeID discv5.NodeID
	)
	if err := record.Load(&key); err != nil {
		return nil, err
	}
	ecdsaKey := ecdsa.PublicKey(key)
	nodeID = discv5.PubkeyID(&ecdsaKey)
	if err := record.Load(&ip); err != nil {
		return nil, err
	}
	if err := record.Load(&tport); err != nil {
		return nil, err
	}
	// ignore absence of udp port, as it is optional
	_ = record.Load(&uport)
	return discv5.NewNode(nodeID, net.IP(ip), uint16(uport), uint16(tport)), nil
}
