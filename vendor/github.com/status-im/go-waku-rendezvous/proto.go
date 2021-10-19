package rendezvous

import (
	"errors"
	"fmt"
	"time"

	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/record"

	pb "github.com/status-im/go-waku-rendezvous/pb"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
)

var log = logging.Logger("rendezvous")

const (
	RendezvousID_v001 = protocol.ID("/vac/waku/rendezvous/0.0.1")
	DefaultTTL        = 20 // 20 seconds
)

type RendezvousError struct {
	Status pb.Message_ResponseStatus
	Text   string
}

func (e RendezvousError) Error() string {
	return fmt.Sprintf("Rendezvous error: %s (%s)", e.Text, pb.Message_ResponseStatus(e.Status).String())
}

func newRegisterMessage(privKey libp2pCrypto.PrivKey, ns string, pi peer.AddrInfo, ttl int) (*pb.Message, error) {
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

	peerRecord := &peer.PeerRecord{
		PeerID: pi.ID,
		Addrs:  pi.Addrs,
		Seq:    uint64(time.Now().Unix()),
	}

	envelope, err := record.Seal(peerRecord, privKey)
	if err != nil {
		return nil, err
	}

	envPayload, err := envelope.Marshal()
	if err != nil {
		return nil, err
	}

	msg.Register.SignedPeerRecord = envPayload

	return msg, nil
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

func unmarshalSignedPeerRecord(envelopeBytes []byte) (peer.AddrInfo, error) {
	envelope, rec, err := record.ConsumeEnvelope(envelopeBytes, peer.PeerRecordEnvelopeDomain)
	if err != nil {
		return peer.AddrInfo{}, err
	}

	peerRecord, ok := rec.(*peer.PeerRecord)
	if !ok {
		return peer.AddrInfo{}, errors.New("invalid peer record")
	}

	if !peerRecord.PeerID.MatchesPublicKey(envelope.PublicKey) {
		return peer.AddrInfo{}, errors.New("signing key does not match peer record")
	}

	return peer.AddrInfo{ID: peerRecord.PeerID, Addrs: peerRecord.Addrs}, nil
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

func newDiscoverResponse(regs []RegistrationRecord) (*pb.Message_DiscoverResponse, error) {
	r := new(pb.Message_DiscoverResponse)
	r.Status = pb.Message_OK

	rregs := make([]*pb.Message_Register, len(regs))
	for i, reg := range regs {
		rreg := new(pb.Message_Register)
		rns := reg.Ns
		rreg.Ns = rns
		rreg.SignedPeerRecord = reg.PeerEnvelope
		rttl := int64(reg.Ttl)
		rreg.Ttl = rttl
		rregs[i] = rreg
	}

	r.Registrations = rregs

	return r, nil
}

func newDiscoverResponseError(status pb.Message_ResponseStatus, text string) *pb.Message_DiscoverResponse {
	r := new(pb.Message_DiscoverResponse)
	r.Status = status
	r.StatusText = text
	return r
}
