package rendezvous

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	pb "github.com/status-im/go-libp2p-rendezvous/pb"

	ggio "github.com/gogo/protobuf/io"

	"github.com/libp2p/go-libp2p-core/host"
	inet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

var (
	DiscoverAsyncInterval = 2 * time.Minute
)

type RendezvousPoint interface {
	Register(ctx context.Context, ns string, ttl int) (time.Duration, error)
	Discover(ctx context.Context, ns string, limit int) ([]Registration, error)
	DiscoverAsync(ctx context.Context, ns string) (<-chan Registration, error)
}

type Registration struct {
	Peer peer.AddrInfo
	Ns   string
	Ttl  int
}

type RendezvousClient interface {
	Register(ctx context.Context, ns string, ttl int) (time.Duration, error)
	Discover(ctx context.Context, ns string, limit int) ([]peer.AddrInfo, error)
	DiscoverAsync(ctx context.Context, ns string) (<-chan peer.AddrInfo, error)
}

func NewRendezvousPoint(host host.Host) RendezvousPoint {
	return &rendezvousPoint{
		host: host,
	}
}

type rendezvousPoint struct {
	host host.Host
}

func NewRendezvousClient(host host.Host) RendezvousClient {
	return NewRendezvousClientWithPoint(NewRendezvousPoint(host))
}

func NewRendezvousClientWithPoint(rp RendezvousPoint) RendezvousClient {
	return &rendezvousClient{rp: rp}
}

type rendezvousClient struct {
	rp RendezvousPoint
}

func (r *rendezvousPoint) getRandomPeer() (peer.ID, error) {
	var peerIDs []peer.ID
	for _, peer := range r.host.Peerstore().Peers() {
		protocols, err := r.host.Peerstore().SupportsProtocols(peer, string(RendezvousID_v001))
		if err != nil {
			log.Error("error obtaining the protocols supported by peers", err)
			return "", err
		}
		if len(protocols) > 0 {
			peerIDs = append(peerIDs, peer)
		}
	}

	if len(peerIDs) == 0 {
		return "", errors.New("no peers available")
	}

	return peerIDs[rand.Intn(len(peerIDs))], nil // nolint: gosec
}

func (rp *rendezvousPoint) Register(ctx context.Context, ns string, ttl int) (time.Duration, error) {
	randomPeer, err := rp.getRandomPeer()
	if err != nil {
		return 0, err
	}

	s, err := rp.host.NewStream(ctx, randomPeer, RendezvousID_v001)
	if err != nil {
		return 0, err
	}
	defer s.Reset()

	r := ggio.NewDelimitedReader(s, inet.MessageSizeMax)
	w := ggio.NewDelimitedWriter(s)

	req := newRegisterMessage(ns, peer.AddrInfo{ID: rp.host.ID(), Addrs: rp.host.Addrs()}, ttl)
	err = w.WriteMsg(req)
	if err != nil {
		return 0, err
	}

	var res pb.Message
	err = r.ReadMsg(&res)
	if err != nil {
		return 0, err
	}

	if res.GetType() != pb.Message_REGISTER_RESPONSE {
		return 0, fmt.Errorf("Unexpected response: %s", res.GetType().String())
	}

	response := res.GetRegisterResponse()
	status := response.GetStatus()
	if status != pb.Message_OK {
		return 0, RendezvousError{Status: status, Text: res.GetRegisterResponse().GetStatusText()}
	}

	return time.Duration(response.Ttl) * time.Second, nil
}

func (rc *rendezvousClient) Register(ctx context.Context, ns string, ttl int) (time.Duration, error) {
	if ttl < 120 {
		return 0, fmt.Errorf("registration TTL is too short")
	}

	returnedTTL, err := rc.rp.Register(ctx, ns, ttl)
	if err != nil {
		return 0, err
	}

	go registerRefresh(ctx, rc.rp, ns, ttl)
	return returnedTTL, nil
}

func registerRefresh(ctx context.Context, rz RendezvousPoint, ns string, ttl int) {
	var refresh time.Duration
	errcount := 0

	for {
		if errcount > 0 {
			// do randomized exponential backoff, up to ~4 hours
			if errcount > 7 {
				errcount = 7
			}
			backoff := 2 << uint(errcount)
			refresh = 5*time.Minute + time.Duration(rand.Intn(backoff*60000))*time.Millisecond
		} else {
			refresh = time.Duration(ttl-30) * time.Second
		}

		select {
		case <-time.After(refresh):
		case <-ctx.Done():
			return
		}

		_, err := rz.Register(ctx, ns, ttl)
		if err != nil {
			log.Errorf("Error registering [%s]: %s", ns, err.Error())
			errcount++
		} else {
			errcount = 0
		}
	}
}

func (rp *rendezvousPoint) Discover(ctx context.Context, ns string, limit int) ([]Registration, error) {
	randomPeer, err := rp.getRandomPeer()
	if err != nil {
		return nil, err
	}

	s, err := rp.host.NewStream(ctx, randomPeer, RendezvousID_v001)
	if err != nil {
		return nil, err
	}
	defer s.Reset()

	r := ggio.NewDelimitedReader(s, inet.MessageSizeMax)
	w := ggio.NewDelimitedWriter(s)

	return discoverQuery(ns, limit, r, w)
}

func discoverQuery(ns string, limit int, r ggio.Reader, w ggio.Writer) ([]Registration, error) {
	req := newDiscoverMessage(ns, limit)
	err := w.WriteMsg(req)
	if err != nil {
		return nil, err
	}

	var res pb.Message
	err = r.ReadMsg(&res)
	if err != nil {
		return nil, err
	}

	if res.GetType() != pb.Message_DISCOVER_RESPONSE {
		return nil, fmt.Errorf("Unexpected response: %s", res.GetType().String())
	}

	status := res.GetDiscoverResponse().GetStatus()
	if status != pb.Message_OK {
		return nil, RendezvousError{Status: status, Text: res.GetDiscoverResponse().GetStatusText()}
	}

	regs := res.GetDiscoverResponse().GetRegistrations()
	result := make([]Registration, 0, len(regs))
	for _, reg := range regs {
		pi, err := pbToPeerInfo(reg.GetPeer())
		if err != nil {
			log.Errorf("Invalid peer info: %s", err.Error())
			continue
		}
		result = append(result, Registration{Peer: pi, Ns: reg.GetNs(), Ttl: int(reg.GetTtl())})
	}

	return result, nil
}

func (rp *rendezvousPoint) DiscoverAsync(ctx context.Context, ns string) (<-chan Registration, error) {
	randomPeer, err := rp.getRandomPeer()
	if err != nil {
		return nil, err
	}

	s, err := rp.host.NewStream(ctx, randomPeer, RendezvousID_v001)
	if err != nil {
		return nil, err
	}

	ch := make(chan Registration)
	go discoverAsync(ctx, ns, s, ch)
	return ch, nil
}

func discoverAsync(ctx context.Context, ns string, s inet.Stream, ch chan Registration) {
	defer s.Reset()
	defer close(ch)

	r := ggio.NewDelimitedReader(s, inet.MessageSizeMax)
	w := ggio.NewDelimitedWriter(s)

	const batch = 200

	var (
		regs []Registration
		err  error
	)

	for {
		regs, err = discoverQuery(ns, batch, r, w)
		if err != nil {
			// TODO robust error recovery
			//      - handle closed streams with backoff + new stream
			log.Errorf("Error in discovery [%s]: %s", ns, err.Error())
			return
		}

		for _, reg := range regs {
			select {
			case ch <- reg:
			case <-ctx.Done():
				return
			}
		}

		if len(regs) < batch {
			// TODO adaptive backoff for heavily loaded rendezvous points
			select {
			case <-time.After(DiscoverAsyncInterval):
			case <-ctx.Done():
				return
			}
		}
	}
}

func (rc *rendezvousClient) Discover(ctx context.Context, ns string, limit int) ([]peer.AddrInfo, error) {
	regs, err := rc.rp.Discover(ctx, ns, limit)
	if err != nil {
		return nil, err
	}

	pinfos := make([]peer.AddrInfo, len(regs))
	for i, reg := range regs {
		pinfos[i] = reg.Peer
	}

	return pinfos, nil
}

func (rc *rendezvousClient) DiscoverAsync(ctx context.Context, ns string) (<-chan peer.AddrInfo, error) {
	rch, err := rc.rp.DiscoverAsync(ctx, ns)
	if err != nil {
		return nil, err
	}

	ch := make(chan peer.AddrInfo)
	go discoverPeersAsync(ctx, rch, ch)
	return ch, nil
}

func discoverPeersAsync(ctx context.Context, rch <-chan Registration, ch chan peer.AddrInfo) {
	defer close(ch)
	for {
		select {
		case reg, ok := <-rch:
			if !ok {
				return
			}

			select {
			case ch <- reg.Peer:
			case <-ctx.Done():
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
