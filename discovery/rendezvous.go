package discovery

import (
	"context"
	"crypto/ecdsa"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/rendezvous"
)

const (
	registrationPeriod = 10 * time.Second
	bucketSize         = 10
)

func NewRendezvous(servers []ma.Multiaddr, identity *ecdsa.PrivateKey, node *discover.Node) (*Rendezvous, error) {
	r := new(Rendezvous)
	r.servers = servers
	r.registrationPeriod = registrationPeriod
	r.bucketSize = bucketSize

	r.record = enr.Record{}
	r.record.Set(enr.IP(node.IP))
	r.record.Set(enr.TCP(node.TCP))
	r.record.Set(enr.UDP(node.UDP))
	// public key is added to ENR when ENR is signed
	if err := enr.SignV4(&r.record, identity); err != nil {
		return nil, err
	}
	return r, nil
}

// Rendezvous is an implementation of discovery interface that uses
// rendezvous client.
type Rendezvous struct {
	mu     sync.RWMutex
	client *rendezvous.Client

	servers            []ma.Multiaddr
	registrationPeriod time.Duration
	bucketSize         int
	record             enr.Record
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
	client, err := rendezvous.NewTemporary()
	if err != nil {
		return err
	}
	r.client = &client
	return nil
}

// Stop removes client reference.
func (r *Rendezvous) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.client = nil
	return nil
}

func (r *Rendezvous) register(topic string) {
	srv := r.servers[rand.Intn(len(r.servers))]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	r.mu.RLock()
	defer r.mu.RUnlock()
	defer cancel()
	if err := r.client.Register(ctx, srv, topic, r.record); err != nil {
		log.Debug("error registering", "topic", topic, "rendevous server", srv, "err", err)
	}
}

// Register renews registration in the specified server.
func (r *Rendezvous) Register(topic string, stop chan struct{}) error {
	ticker := time.NewTicker(r.registrationPeriod)
	defer ticker.Stop()
	r.register(topic)
	for {
		select {
		case <-stop:
			return nil
		case <-ticker.C:
			r.register(topic)
		}
	}
}

// Discover will search for new records every time period fetched from period channel.
func (r *Rendezvous) Discover(
	topic string, period <-chan time.Duration,
	found chan<- *discv5.Node, lookup chan<- bool) error {
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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			r.mu.RLock()
			records, err := r.client.Discover(ctx, srv, topic, r.bucketSize)
			r.mu.RUnlock()
			cancel()
			if err != nil {
				log.Debug("error fetching records", "topic", topic, "rendezvous server", srv, "err", err)
				continue
			}
			for i := range records {
				n, err := enrToNode(records[i])
				if err != nil {
					log.Warn("error converting enr record to node", "err", err)
				}
				found <- n
			}
		}
	}
}

func enrToNode(record enr.Record) (*discv5.Node, error) {
	var (
		key   enr.Secp256k1
		ip    enr.IP
		tport enr.TCP
		uport enr.UDP
	)
	if err := record.Load(&key); err != nil {
		return nil, err
	}
	if err := record.Load(&ip); err != nil {
		return nil, err
	}
	if err := record.Load(&tport); err != nil {
		return nil, err
	}
	// ignore absence of udp port, as it is optional
	_ = record.Load(&uport)
	ecdsaKey := ecdsa.PublicKey(key)
	return discv5.NewNode(discv5.PubkeyID(&ecdsaKey), net.IP(ip), uint16(uport), uint16(tport)), nil
}
