package discovery

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	proxyStart = "start"
	proxyStop  = "stop"
)

type proxyEvent struct {
	ID   discv5.NodeID
	Type string
	Time time.Time
}

type ProxyOptions struct {
	Topic          string
	Servers        []ma.Multiaddr
	Limit          int
	LivenessWindow time.Duration
}

// ProxyToRendezvous proxies records discovered using original to rendezvous servers for specified topic.
func ProxyToRendezvous(original Discovery, stop chan struct{}, feed *event.Feed, opts ProxyOptions) error {
	var (
		identities      = map[discv5.NodeID]*Rendezvous{}
		lastSeen        = map[discv5.NodeID]time.Time{}
		closers         = map[discv5.NodeID]chan struct{}{}
		period          = make(chan time.Duration, 1)
		found           = make(chan *discv5.Node, 10)
		lookup          = make(chan bool)
		total           = 0
		livenessWatcher = time.NewTicker(opts.LivenessWindow / 10)
		wg              sync.WaitGroup
	)
	defer livenessWatcher.Stop()
	period <- 1 * time.Second
	wg.Add(1)
	go func() {
		if err := original.Discover(opts.Topic, period, found, lookup); err != nil {
			log.Error("discover request failed", "topic", opts.Topic, "error", err)
		}
		wg.Done()
	}()
	for {
		select {
		case <-stop:
			close(period)
			wg.Wait()
			return nil
		case <-lookup:
		case <-livenessWatcher.C:
			for n := range identities {
				if _, exist := lastSeen[n]; !exist {
					continue
				}
				// closeRequest is sent every time window after record was seen.
				// record must be discovered again during same time window otherwise it will be removed.
				if time.Since(lastSeen[n]) >= opts.LivenessWindow {
					close(closers[n])
					_ = identities[n].Stop()
					delete(identities, n)
					delete(lastSeen, n)
					delete(closers, n)
					total--
					log.Info("proxy for a record was removed", "identity", n.String(), "total", total)
					feed.Send(proxyEvent{n, proxyStop, time.Now()})
				}
			}
		case n := <-found:
			_, exist := identities[n.ID]
			// skip new record if we reached a limit.
			if !exist && total == opts.Limit {
				continue
			}
			lastSeen[n.ID] = time.Now()
			if exist {
				log.Debug("received an update for existing identity", "identity", n.String())
				continue
			}
			feed.Send(proxyEvent{n.ID, proxyStart, lastSeen[n.ID]})
			total++
			log.Info("proxying new record", "topic", opts.Topic, "identity", n.String(), "total", total)
			record, err := makeProxiedENR(n)
			if err != nil {
				log.Error("error converting discovered node to ENR", "node", n.String(), "error", err)
			}
			r := NewRendezvousWithENR(opts.Servers, record)
			identities[n.ID] = r
			closers[n.ID] = make(chan struct{})
			if err := r.Start(); err != nil {
				log.Error("unable to start rendezvous proxying", "servers", opts.Servers, "error", err)
			}
			wg.Add(1)
			go func() {
				if err := r.Register(opts.Topic, closers[n.ID]); err != nil {
					log.Error("register error", "topic", opts.Topic, "error", err)
				}
				wg.Done()
			}()
		}
	}
}

func makeProxiedENR(n *discv5.Node) (enr.Record, error) {
	record := enr.Record{}
	record.Set(enr.IP(n.IP))
	record.Set(enr.TCP(n.TCP))
	record.Set(enr.UDP(n.UDP))
	record.Set(Proxied(n.ID))
	key, err := crypto.GenerateKey() // we need separate key for each identity, records are stored based on it
	if err != nil {
		return record, fmt.Errorf("unable to generate private key. error : %v", err)
	}
	if err := enode.SignV4(&record, key); err != nil {
		return record, fmt.Errorf("unable to sign enr record. error: %v", err)
	}
	return record, nil
}

// Proxied is an Entry for ENR
type Proxied discv5.NodeID

// ENRKey returns unique key that is used by ENR.
func (p Proxied) ENRKey() string {
	return "proxied"
}
