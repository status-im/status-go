package rendezvous

import (
	"errors"
	"fmt"

	pb "github.com/status-im/go-libp2p-rendezvous/pb"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	ma "github.com/multiformats/go-multiaddr"
)

var log = logging.Logger("rendezvous")

const (
	RendezvousID_v001 = protocol.ID("/vac/waku/rendezvous/0.0.1")
	DefaultTTL        = 2 * 3600 // 2hr
)

type RendezvousError struct {
	Status pb.Message_ResponseStatus
	Text   string
}

func (e RendezvousError) Error() string {
	return fmt.Sprintf("Rendezvous error: %s (%s)", e.Text, pb.Message_ResponseStatus(e.Status).String())
}

func newRegisterMessage(ns string, pi peer.AddrInfo, ttl int) *pb.Message {
	msg := new(pb.Message)
	msg.Type = pb.Message_REGISTER
	msg.Register = new(pb.Message_Register)
	if ns != "" {
		msg.Register.Ns = ns
	}
	if ttl > 0 {
		ttl64 := int64(ttl)
		msg.Register.Ttl = ttl64
	}
	msg.Register.Peer = new(pb.Message_PeerInfo)
	msg.Register.Peer.Id = []byte(pi.ID)
	msg.Register.Peer.Addrs = make([][]byte, len(pi.Addrs))
	for i, addr := range pi.Addrs {
		msg.Register.Peer.Addrs[i] = addr.Bytes()
	}
	return msg
}

func newUnregisterMessage(ns string, pid peer.ID) *pb.Message {
	msg := new(pb.Message)
	msg.Type = pb.Message_UNREGISTER
	msg.Unregister = new(pb.Message_Unregister)
	if ns != "" {
		msg.Unregister.Ns = ns
	}
	msg.Unregister.Id = []byte(pid)
	return msg
}

func newDiscoverMessage(ns string, limit int) *pb.Message {
	msg := new(pb.Message)
	msg.Type = pb.Message_DISCOVER
	msg.Discover = new(pb.Message_Discover)
	if ns != "" {
		msg.Discover.Ns = ns
	}
	if limit > 0 {
		limit64 := int64(limit)
		msg.Discover.Limit = limit64
	}
	return msg
}

func pbToPeerInfo(p *pb.Message_PeerInfo) (peer.AddrInfo, error) {
	if p == nil {
		return peer.AddrInfo{}, errors.New("missing peer info")
	}

	id, err := peer.IDFromBytes(p.Id)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	addrs := make([]ma.Multiaddr, 0, len(p.Addrs))
	for _, bs := range p.Addrs {
		addr, err := ma.NewMultiaddrBytes(bs)
		if err != nil {
			log.Errorf("Error parsing multiaddr: %s", err.Error())
			continue
		}
		addrs = append(addrs, addr)
	}

	return peer.AddrInfo{ID: id, Addrs: addrs}, nil
}

func newRegisterResponse(ttl int) *pb.Message_RegisterResponse {
	ttl64 := int64(ttl)
	r := new(pb.Message_RegisterResponse)
	r.Status = pb.Message_OK
	r.Ttl = ttl64
	return r
}

func newRegisterResponseError(status pb.Message_ResponseStatus, text string) *pb.Message_RegisterResponse {
	r := new(pb.Message_RegisterResponse)
	r.Status = status
	r.StatusText = text
	return r
}

func newDiscoverResponse(regs []RegistrationRecord) *pb.Message_DiscoverResponse {
	r := new(pb.Message_DiscoverResponse)
	r.Status = pb.Message_OK

	rregs := make([]*pb.Message_Register, len(regs))
	for i, reg := range regs {
		rreg := new(pb.Message_Register)
		rns := reg.Ns
		rreg.Ns = rns
		rreg.Peer = new(pb.Message_PeerInfo)
		rreg.Peer.Id = []byte(reg.Id)
		rreg.Peer.Addrs = reg.Addrs
		rttl := int64(reg.Ttl)
		rreg.Ttl = rttl
		rregs[i] = rreg
	}

	r.Registrations = rregs

	return r
}

func newDiscoverResponseError(status pb.Message_ResponseStatus, text string) *pb.Message_DiscoverResponse {
	r := new(pb.Message_DiscoverResponse)
	r.Status = status
	r.StatusText = text
	return r
}
