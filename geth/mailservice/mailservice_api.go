package mailservice

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/log"
)

const (
	// defaultWorkTime is a work time reported in messages sent to MailServer nodes.
	defaultWorkTime = 5

	// gcTimeout defines timeout to release not used connection with mailserver peers.
	gcTimeout = 5 * time.Minute
	// gcPeriod defines how often to run garbage collector.
	gcPeriod = 3 * time.Second

	// peerConnectTimeout defines timeout to connect with a peer.
	peerConnectTimeout = 10 * time.Second
)

var (
	// ErrInvalidMailServerPeer is returned when it fails to parse enode from params.
	ErrInvalidMailServerPeer = errors.New("invalid mailServerPeer value")
	// ErrInvalidSymKeyID is returned when it fails to get a symmetric key.
	ErrInvalidSymKeyID = errors.New("invalid symKeyID value")
	// ErrNoServersInConfig is returned when no trusted peers are configured for p2p.Server
	ErrNoServersInConfig = errors.New("no mailservers are available")
)

// PublicAPI defines a MailServer public API.
type PublicAPI struct {
	service *MailService

	newConnectedPeers chan *discover.Node
}

// NewPublicAPI returns a new PublicAPI.
func NewPublicAPI(service *MailService) *PublicAPI {
	api := &PublicAPI{
		service:           service,
		newConnectedPeers: make(chan *discover.Node),
	}
	go api.runTrustedPeersGC(gcTimeout, gcPeriod)
	return api
}

func (api *PublicAPI) provider() ServiceProvider {
	return api.service.provider
}

// addPeer tries to connect with a peer and waits till connection will be established.
func (api *PublicAPI) addPeer(peer *discover.Node, timeout time.Duration) error {
	log.Debug("adding a peer", "peer", peer.String())
	server, err := api.provider().Server()
	if err != nil {
		return err
	}
	events := make(chan *p2p.PeerEvent, 10)
	sub := server.SubscribeEvents(events)
	defer sub.Unsubscribe()
	server.AddPeer(peer)
	for {
		select {
		case ev := <-events:
			if ev.Type == p2p.PeerEventTypeAdd && ev.Peer == peer.ID {
				log.Debug("peer added", "peer", peer.String())
				return nil
			}
		case <-time.After(timeout):
			return fmt.Errorf("failed to add a peer: %s", peer.String())
		}
	}
}

// runTrustedPeersGC collects connections that weren't used for defined
// garbage collector timeout.
func (api *PublicAPI) runTrustedPeersGC(timeout, period time.Duration) {
	connectedPeers := map[*discover.Node]time.Time{}
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-api.service.quit:
			log.Debug("shutdown mailserver api gc")
			return
		case peer := <-api.newConnectedPeers:
			connectedPeers[peer] = time.Now()
		case <-ticker.C:
			for peer, lastUsed := range connectedPeers {
				if time.Since(lastUsed) >= timeout {
					server, err := api.provider().Server()
					// node was stopped
					if err != nil {
						log.Debug("exit from mailserver api gc. failed to get a server", "error", err)
						return
					}
					server.RemovePeer(peer)
					delete(connectedPeers, peer)
				}
			}
		}
	}
}

// choosePeer loops over trusted nodes, and returns one that is connected or
// tries to establish a connection.
func (api *PublicAPI) choosePeer() (*discover.Node, error) {
	server, err := api.provider().Server()
	if err != nil {
		return nil, err
	}
	if len(server.Config.TrustedNodes) == 0 {
		return nil, ErrNoServersInConfig
	}
	// we are not relying on GC for this to avoid any mismatch between
	// real data and GC. GC used only to disconnect peers that didn't
	// disconnect themself
	connected := map[discover.NodeID]struct{}{}
	for _, peer := range server.Peers() {
		connected[peer.ID()] = struct{}{}
	}
	for _, trusted := range server.Config.TrustedNodes {
		if _, exist := connected[trusted.ID]; exist {
			return trusted, nil
		}
	}
	// TODO(dshulyak) choose randomly
	peer := server.Config.TrustedNodes[0]
	if err := api.addPeer(peer, peerConnectTimeout); err != nil {
		return nil, err
	}
	return peer, nil
}

// RequestMessages sends a request for historic messages to a MailServer.
func (api *PublicAPI) RequestMessages(_ context.Context, r MessagesRequest) (bool, error) {
	log.Info("RequestMessages", "request", r)

	setMessagesRequestDefaults(&r)

	shh, err := api.provider().WhisperService()
	if err != nil {
		return false, err
	}
	server, err := api.provider().Server()
	if err != nil {
		return false, err
	}
	peer, err := api.choosePeer()
	if err != nil {
		return false, err
	}
	// renew gc timer
	api.newConnectedPeers <- peer
	symKey, err := shh.GetSymKey(r.SymKeyID)
	if err != nil {
		return false, fmt.Errorf("%v: %v", ErrInvalidSymKeyID, err)
	}

	envelope, err := makeEnvelop(makePayload(r), symKey, server.PrivateKey, shh.MinPow())
	if err != nil {
		return false, err
	}
	if err := shh.RequestHistoricMessages(peer.ID[:], envelope); err != nil {
		return false, err
	}

	return true, nil
}

// MessagesRequest is a payload send to a MailServer to get messages.
type MessagesRequest struct {
	// MailServerPeer is MailServer's enode address.
	MailServerPeer string `json:"mailServerPeer"`

	// From is a lower bound of time range (optional).
	// Default is 24 hours back from now.
	From uint32 `json:"from"`

	// To is a upper bound of time range (optional).
	// Default is now.
	To uint32 `json:"to"`

	// Topic is a regular Whisper topic.
	Topic whisper.TopicType `json:"topic"`

	// SymKeyID is an ID of a symmetric key to authenticate to MailServer.
	// It's derived from MailServer password.
	SymKeyID string `json:"symKeyID"`
}

func setMessagesRequestDefaults(r *MessagesRequest) {
	// set From and To defaults
	if r.From == 0 && r.To == 0 {
		r.From = uint32(time.Now().UTC().Add(-24 * time.Hour).Unix())
		r.To = uint32(time.Now().UTC().Unix())
	}
}

// makeEnvelop makes an envelop for a historic messages request.
// Symmetric key is used to authenticate to MailServer.
// PK is the current node ID.
func makeEnvelop(payload []byte, symKey []byte, nodeID *ecdsa.PrivateKey, pow float64) (*whisper.Envelope, error) {
	params := whisper.MessageParams{
		PoW:      pow,
		Payload:  payload,
		KeySym:   symKey,
		WorkTime: defaultWorkTime,
		Src:      nodeID,
	}
	message, err := whisper.NewSentMessage(&params)
	if err != nil {
		return nil, err
	}
	return message.Wrap(&params)
}

// makePayload makes a specific payload for MailServer to request historic messages.
func makePayload(r MessagesRequest) []byte {
	// first 8 bytes are lowed and upper bounds as uint32
	data := make([]byte, 8+whisper.TopicLength)
	binary.BigEndian.PutUint32(data, r.From)
	binary.BigEndian.PutUint32(data[4:], r.To)
	copy(data[8:], r.Topic[:])
	return data
}
