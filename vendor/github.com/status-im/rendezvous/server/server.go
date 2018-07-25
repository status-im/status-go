package server

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/status-im/rendezvous/protocol"
)

var logger = log.New("package", "rendezvous/server")

const (
	longestTTL          = 20 * time.Second
	cleanerPeriod       = 2 * time.Second
	maxLimit       uint = 10
	maxTopicLength      = 50
)

// NewServer creates instance of the server.
func NewServer(laddr ma.Multiaddr, identity crypto.PrivKey, s Storage) *Server {
	srv := Server{
		laddr:         laddr,
		identity:      identity,
		storage:       s,
		cleaner:       NewCleaner(),
		writeTimeout:  10 * time.Second,
		readTimeout:   10 * time.Second,
		cleanerPeriod: cleanerPeriod,
	}
	return &srv
}

// Server provides rendezbous service over libp2p stream.
type Server struct {
	laddr    ma.Multiaddr
	identity crypto.PrivKey

	writeTimeout time.Duration
	readTimeout  time.Duration

	storage       Storage
	cleaner       *Cleaner
	cleanerPeriod time.Duration

	h    host.Host
	addr ma.Multiaddr

	wg   sync.WaitGroup
	quit chan struct{}
}

// Addr returns full server multiaddr (identity included).
func (srv *Server) Addr() ma.Multiaddr {
	return srv.addr
}

// Start creates listener.
func (srv *Server) Start() error {
	if err := srv.startListener(); err != nil {
		return err
	}
	if err := srv.startCleaner(); err != nil {
		return err
	}
	// once server is restarted all cleaner info is lost. so we need to rebuild it
	return srv.storage.IterateAllKeys(func(key RecordsKey, ttl time.Time) error {
		srv.cleaner.Add(ttl, key.String())
		return nil
	})
}

func (srv *Server) startCleaner() error {
	srv.quit = make(chan struct{})
	srv.wg.Add(1)
	go func() {
		for {
			select {
			case <-time.After(srv.cleanerPeriod):
				srv.purgeOutdated()
			case <-srv.quit:
				srv.wg.Done()
				return
			}
		}
	}()
	return nil
}

func (srv *Server) startListener() error {
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(srv.laddr.String()),
		libp2p.Identity(srv.identity),
	}
	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return err
	}
	srv.h = h
	srv.h.SetStreamHandler(protocol.VERSION, func(s net.Stream) {
		defer s.Close()
		rs := rlp.NewStream(s, 0)
		s.SetReadDeadline(time.Now().Add(srv.readTimeout))
		typ, err := rs.Uint()
		if err != nil {
			logger.Error("error reading message type", "error", err)
			return
		}
		s.SetReadDeadline(time.Now().Add(srv.readTimeout))
		resptype, resp, err := srv.msgParser(protocol.MessageType(typ), rs)
		if err != nil {
			logger.Error("error parsing message", "error", err)
			return
		}
		s.SetWriteDeadline(time.Now().Add(srv.writeTimeout))
		if err = rlp.Encode(s, resptype); err != nil {
			logger.Error("error writing response", "type", resptype, "error", err)
			return
		}
		s.SetWriteDeadline(time.Now().Add(srv.writeTimeout))
		if err = rlp.Encode(s, resp); err != nil {
			logger.Error("error encoding response", "resp", resp, "error", err)
		}
	})
	addr, err := ma.NewMultiaddr(fmt.Sprintf("/ethv4/%s", h.ID().Pretty()))
	if err != nil {
		return err
	}
	srv.addr = srv.laddr.Encapsulate(addr)
	logger.Info("server started", "address", srv.laddr.Encapsulate(addr))
	return nil
}

// Stop closes listener and waits till all helper goroutines are stopped.
func (srv *Server) Stop() {
	if srv.quit == nil {
		return
	}
	select {
	case <-srv.quit:
		return
	default:
	}
	close(srv.quit)
	srv.wg.Wait()
	if srv.h != nil {
		srv.h.Close()
	}
}

func (srv *Server) purgeOutdated() {
	key := srv.cleaner.PopOneSince(time.Now())
	if len(key) == 0 {
		return
	}
	if err := srv.storage.RemoveByKey(key); err != nil {
		logger.Error("error removing key from storage", "key", key, "error", err)
	}
}

// Decoder is a decoder!
type Decoder interface {
	Decode(val interface{}) error
}

func (srv *Server) msgParser(typ protocol.MessageType, d Decoder) (resptype protocol.MessageType, resp interface{}, err error) {
	switch typ {
	case protocol.REGISTER:
		var msg protocol.Register
		resptype = protocol.REGISTER_RESPONSE
		if err = d.Decode(&msg); err != nil {
			return resptype, protocol.RegisterResponse{Status: protocol.E_INVALID_CONTENT}, nil
		}
		resp, err = srv.register(msg)
		return resptype, resp, err
	case protocol.DISCOVER:
		var msg protocol.Discover
		resptype = protocol.DISCOVER_RESPONSE
		if err = d.Decode(&msg); err != nil {
			return resptype, protocol.DiscoverResponse{Status: protocol.E_INVALID_CONTENT}, nil
		}
		limit := msg.Limit
		if msg.Limit > maxLimit {
			limit = maxLimit
		}
		records, err := srv.storage.GetRandom(msg.Topic, limit)
		if err != nil {
			return resptype, protocol.DiscoverResponse{Status: protocol.E_INTERNAL_ERROR}, err
		}
		return resptype, protocol.DiscoverResponse{Status: protocol.OK, Records: records}, nil
	default:
		// don't send the response
		return 0, nil, errors.New("unknown request type")
	}
}

func (srv *Server) register(msg protocol.Register) (protocol.RegisterResponse, error) {
	if len(msg.Topic) == 0 || len(msg.Topic) > maxTopicLength {
		return protocol.RegisterResponse{Status: protocol.E_INVALID_NAMESPACE}, nil
	}
	if time.Duration(msg.TTL) > longestTTL {
		return protocol.RegisterResponse{Status: protocol.E_INVALID_TTL}, nil
	}
	if !msg.Record.Signed() {
		return protocol.RegisterResponse{Status: protocol.E_INVALID_ENR}, nil
	}
	ttl := time.Now().Add(time.Duration(msg.TTL))
	key, err := srv.storage.Add(msg.Topic, msg.Record, ttl)
	if err != nil {
		return protocol.RegisterResponse{Status: protocol.E_INTERNAL_ERROR}, err
	}
	srv.cleaner.Add(ttl, key)
	return protocol.RegisterResponse{Status: protocol.OK}, nil
}
