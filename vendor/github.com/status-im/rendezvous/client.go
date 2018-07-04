package rendezvous

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	ethv4 "github.com/status-im/go-multiaddr-ethv4"
	"github.com/status-im/rendezvous/protocol"
)

var logger = log.New("package", "rendezvous/client")

func NewTemporary() (c Client, err error) {
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		return Client{}, err
	}
	return New(priv)
}

func New(identity crypto.PrivKey) (c Client, err error) {
	opts := []libp2p.Option{
		libp2p.Identity(identity),
	}
	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return c, err
	}
	return Client{
		identity: identity,
		h:        h,
	}, nil
}

type Client struct {
	laddr    ma.Multiaddr
	identity crypto.PrivKey

	h host.Host
}

func (c Client) Register(ctx context.Context, srv ma.Multiaddr, topic string, record enr.Record) error {
	s, err := c.newStream(ctx, srv)
	if err != nil {
		return err
	}
	defer s.Close()
	if err = rlp.Encode(s, protocol.REGISTER); err != nil {
		return err
	}
	if err = rlp.Encode(s, protocol.Register{Topic: topic, Record: record, TTL: uint64(5 * time.Second)}); err != nil {
		return err
	}
	rs := rlp.NewStream(s, 0)
	typ, err := rs.Uint()
	if err != nil {
		return err
	}
	if protocol.MessageType(typ) != protocol.REGISTER_RESPONSE {
		return fmt.Errorf("expected %v as response, but got %v", protocol.REGISTER_RESPONSE, typ)
	}
	var val protocol.RegisterResponse
	if err = rs.Decode(&val); err != nil {
		return err
	}
	logger.Debug("received response to register", "status", val.Status, "message", val.Message)
	if val.Status != protocol.OK {
		return fmt.Errorf("register failed. status code %v", val.Status)
	}
	return nil
}

func (c Client) Discover(ctx context.Context, srv ma.Multiaddr, topic string, limit int) (rst []enr.Record, err error) {
	s, err := c.newStream(ctx, srv)
	if err != nil {
		return
	}
	defer s.Close()
	if err = rlp.Encode(s, protocol.DISCOVER); err != nil {
		return
	}
	if err = rlp.Encode(s, protocol.Discover{Topic: topic, Limit: uint(limit)}); err != nil {
		return
	}
	rs := rlp.NewStream(s, 0)
	typ, err := rs.Uint()
	if err != nil {
		return nil, err
	}
	if protocol.MessageType(typ) != protocol.DISCOVER_RESPONSE {
		return nil, fmt.Errorf("expected %v as response, but got %v", protocol.REGISTER_RESPONSE, typ)
	}
	var val protocol.DiscoverResponse
	if err = rs.Decode(&val); err != nil {
		return
	}
	if val.Status != protocol.OK {
		return nil, fmt.Errorf("discover request failed. status code %v", val.Status)
	}
	logger.Debug("received response to discover request", "status", val.Status, "records lth", len(val.Records))
	return val.Records, nil
}

func (c Client) newStream(ctx context.Context, srv ma.Multiaddr) (rw io.ReadWriteCloser, err error) {
	pid, err := srv.ValueForProtocol(ethv4.P_ETHv4)
	if err != nil {
		return
	}
	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		return
	}
	// TODO there must be a better interface
	targetPeerAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ethv4/%s", pid))
	if err != nil {
		return
	}
	targetAddr := srv.Decapsulate(targetPeerAddr)
	c.h.Peerstore().AddAddr(peerid, targetAddr, 5*time.Second)
	s, err := c.h.NewStream(ctx, peerid, "/rend/0.1.0")
	if err != nil {
		return nil, err
	}
	return InstrumenetedStream{s}, nil
}
