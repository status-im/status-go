package discovery

import (
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ma "github.com/multiformats/go-multiaddr"
)

// V5ToRendezvousProxy registers each LES record in rendezvous servers.
type V5ToRendezvousProxy struct {
	original *DiscV5
	servers  []ma.Multiaddr // rendezvous servers

	identities map[discv5.NodeID]*Rendezvous
}

func NewProxy(original *DiscV5, servers []ma.Multiaddr) V5ToRendezvousProxy {
	return V5ToRendezvousProxy{
		original:   original,
		servers:    servers,
		identities: map[discv5.NodeID]*Rendezvous{},
	}
}

func (proxy *V5ToRendezvousProxy) Run(topic string, stop chan struct{}) error {
	period := make(chan time.Duration, 1)
	period <- 30 * time.Second
	found := make(chan *discv5.Node, 10)
	lookup := make(chan bool)
	go proxy.original.Discover(topic, period, found, lookup)

	for {
		select {
		case <-stop:
			close(period)
			return nil
		case <-lookup:
		case n := <-found:
			if _, exist := proxy.identities[n.ID]; exist {
				continue
			}
			log.Debug("proxying new record", "topic", topic, "identity", n.String())
			record := enr.Record{}
			record.Set(enr.IP(n.IP))
			record.Set(enr.TCP(n.TCP))
			record.Set(enr.UDP(n.UDP))
			record.Set(Proxied(n.ID))
			key, err := crypto.GenerateKey() // we need separate key for each identity, records are stored based on it
			if err != nil {
				log.Error("unable to generate private key", "error", err)
				continue
			}
			if err := enr.SignV4(&record, key); err != nil {
				log.Error("unable to sign enr record", "error", err)
				continue
			}

			r := NewRendezvousWithENR(proxy.servers, record)
			proxy.identities[n.ID] = r
			go r.Register(topic, stop)
		}
	}
}

// Proxied is an Entry for ENR
type Proxied discv5.NodeID

func (p Proxied) ENRKey() string {
	return "proxied"
}
