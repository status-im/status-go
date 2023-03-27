package rendezvous

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	ggio "github.com/gogo/protobuf/io"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/host"
	inet "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	pb "github.com/berty/go-libp2p-rendezvous/pb"
)

type client struct {
	ctx           context.Context
	host          host.Host
	mu            sync.Mutex
	streams       map[string]inet.Stream
	subscriptions map[string]map[string]chan *Registration
}

func NewSyncInMemClient(ctx context.Context, h host.Host) *client {
	return &client{
		ctx:           ctx,
		host:          h,
		streams:       map[string]inet.Stream{},
		subscriptions: map[string]map[string]chan *Registration{},
	}
}

func (c *client) getStreamToPeer(pidStr string) (inet.Stream, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if stream, ok := c.streams[pidStr]; ok {
		return stream, nil
	}

	pid, err := peer.Decode(pidStr)
	if err != nil {
		return nil, fmt.Errorf("unable to decode peer id: %w", err)
	}

	stream, err := c.host.NewStream(c.ctx, pid, ServiceProto)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to peer: %w", err)
	}

	go c.streamListener(stream)

	return stream, nil
}

func (c *client) streamListener(s inet.Stream) {
	r := ggio.NewDelimitedReader(s, inet.MessageSizeMax)
	record := &pb.RegistrationRecord{}

	for {
		err := r.ReadMsg(record)
		if err != nil {
			log.Errorf("unable to decode message: %s", err.Error())
			return
		}

		pid, err := peer.Decode(record.Id)
		if err != nil {
			log.Warnf("invalid peer id: %s", err.Error())
			continue
		}

		maddrs := make([]multiaddr.Multiaddr, len(record.Addrs))
		for i, addrBytes := range record.Addrs {
			maddrs[i], err = multiaddr.NewMultiaddrBytes(addrBytes)
			if err != nil {
				log.Warnf("invalid multiaddr: %s", err.Error())
				continue
			}
		}

		c.mu.Lock()
		subscriptions, ok := c.subscriptions[record.Ns]
		if ok {
			for _, subscription := range subscriptions {
				subscription <- &Registration{
					Peer: peer.AddrInfo{
						ID:    pid,
						Addrs: maddrs,
					},
					Ns:  record.Ns,
					Ttl: int(record.Ttl),
				}
			}
		}
		c.mu.Unlock()
	}
}

func (c *client) Subscribe(ctx context.Context, syncDetails string) (<-chan *Registration, error) {
	ctxUUID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("unable to generate uuid: %w", err)
	}

	psDetails := &PubSubSubscriptionDetails{}

	err = json.Unmarshal([]byte(syncDetails), psDetails)
	if err != nil {
		return nil, fmt.Errorf("unable to decode json: %w", err)
	}

	s, err := c.getStreamToPeer(psDetails.PeerID)
	if err != nil {
		return nil, fmt.Errorf("unable to get stream to peer: %w", err)
	}

	w := ggio.NewDelimitedWriter(s)

	err = w.WriteMsg(&pb.Message{
		Type: pb.Message_DISCOVER_SUBSCRIBE,
		DiscoverSubscribe: &pb.Message_DiscoverSubscribe{
			Ns: psDetails.ChannelName,
		}})
	if err != nil {
		return nil, fmt.Errorf("unable to query server")
	}

	ch := make(chan *Registration)
	c.mu.Lock()
	if _, ok := c.subscriptions[psDetails.ChannelName]; !ok {
		c.subscriptions[psDetails.ChannelName] = map[string]chan *Registration{}
	}

	c.subscriptions[psDetails.ChannelName][ctxUUID.String()] = ch
	c.mu.Unlock()

	go func() {
		<-ctx.Done()
		c.mu.Lock()
		delete(c.subscriptions[psDetails.ChannelName], ctxUUID.String())
		c.mu.Unlock()
		close(ch)
	}()

	return ch, nil
}

func (c *client) GetServiceType() string {
	return ServiceType
}

var _ RendezvousSyncClient = (*client)(nil)
