package peerdiscovery

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// IPVersion specifies the version of the Internet Protocol to be used.
type IPVersion uint

const (
	IPv4 IPVersion = 4
	IPv6 IPVersion = 6
)

// Discovered is the structure of the discovered peers,
// which holds their local address (port removed) and
// a payload if there is one.
type Discovered struct {
	// Address is the local address of a discovered peer.
	Address string
	// Payload is the associated payload from discovered peer.
	Payload []byte

	Metadata *Metadata
}

// Metadata is the metadata associated with a discovered peer.
// To update the metadata, assign your own metadata to the Metadata.Data field.
// The metadata is not protected by a mutex, so you must do this yourself.
// The metadata update happens by pointer, to keep the library backwards compatible.
type Metadata struct {
	Data interface{}
}

func (d Discovered) String() string {
	return fmt.Sprintf("address: %s, payload: %s", d.Address, d.Payload)
}

// Settings are the settings that can be specified for
// doing peer discovery.
type Settings struct {
	// Limit is the number of peers to discover, use < 1 for unlimited.
	Limit int
	// Port is the port to broadcast on (the peers must also broadcast using the same port).
	// The default port is 9999.
	Port string
	// MulticastAddress specifies the multicast address.
	// You should be able to use any of 224.0.0.0/4 or ff00::/8.
	// By default it uses the Simple Service Discovery Protocol
	// address (239.255.255.250 for IPv4 or ff02::c for IPv6).
	MulticastAddress string
	// Payload is the bytes that are sent out with each broadcast. Must be short.
	Payload []byte
	// PayloadFunc is the function that will be called to dynamically generate payload
	// before every broadcast. If this pointer is nil `Payload` field will be broadcasted instead.
	PayloadFunc func() []byte
	// Delay is the amount of time between broadcasts. The default delay is 1 second.
	Delay time.Duration
	// TimeLimit is the amount of time to spend discovering, if the limit is not reached.
	// A negative limit indiciates scanning until the limit was reached or, if an
	// unlimited scanning was requested, no timeout.
	// The default time limit is 10 seconds.
	TimeLimit time.Duration
	// StopChan is a channel to stop the peer discvoery immediatley after reception.
	StopChan chan struct{}
	// AllowSelf will allow discovery the local machine (default false)
	AllowSelf bool
	// DisableBroadcast will not allow sending out a broadcast
	DisableBroadcast bool
	// IPVersion specifies the version of the Internet Protocol (default IPv4)
	IPVersion IPVersion
	// Notify will be called each time a new peer was discovered.
	// The default is nil, which means no notification whatsoever.
	Notify func(Discovered)

	// NotifyLost will be called each time a peer was lost.
	// The default is nil, which means no notification whatsoever.
	// This function should not take too long to execute, as it is called
	// from the peer garbage collector.
	NotifyLost func(LostPeer)

	portNum                 int
	multicastAddressNumbers net.IP
}

type NetPacketConn interface {
	JoinGroup(ifi *net.Interface, group net.Addr) error
	SetMulticastInterface(ini *net.Interface) error
	SetMulticastTTL(int) error
	ReadFrom(buf []byte) (int, net.Addr, error)
	WriteTo(buf []byte, dst net.Addr) (int, error)
}

// Discover will use the created settings to scan for LAN peers. It will return
// an array of the discovered peers and their associate payloads. It will not
// return broadcasts sent to itself.
func Discover(settings ...Settings) (discoveries []Discovered, err error) {
	_, discoveries, err = newPeerDiscovery(settings...)
	if err != nil {
		return nil, err
	}
	return discoveries, nil
}

func NewPeerDiscovery(settings ...Settings) (pd *PeerDiscovery, err error) {
	pd, discoveries, err := newPeerDiscovery(settings...)

	if notify := pd.settings.Notify; notify != nil {
		for _, d := range discoveries {
			notify(d)
		}
	}

	return pd, err
}

func newPeerDiscovery(settings ...Settings) (pd *PeerDiscovery, discoveries []Discovered, err error) {
	s := Settings{}
	if len(settings) > 0 {
		s = settings[0]
	}
	p, err := initialize(s)
	if err != nil {
		return nil, nil, err
	}

	p.RLock()
	address := net.JoinHostPort(p.settings.MulticastAddress, p.settings.Port)
	portNum := p.settings.portNum

	tickerDuration := p.settings.Delay
	timeLimit := p.settings.TimeLimit
	p.RUnlock()

	ifaces, err := filterInterfaces(p.settings.IPVersion == IPv4)
	if err != nil {
		return nil, nil, err
	}
	if len(ifaces) == 0 {
		err = fmt.Errorf("no multicast interface found")
		return nil, nil, err
	}

	// Open up a connection
	c, err := net.ListenPacket(fmt.Sprintf("udp%d", p.settings.IPVersion), address)
	if err != nil {
		return nil, nil, err
	}
	defer c.Close()

	group := p.settings.multicastAddressNumbers

	// ipv{4,6} have an own PacketConn, which does not implement net.PacketConn
	var p2 NetPacketConn
	if p.settings.IPVersion == IPv4 {
		p2 = PacketConn4{ipv4.NewPacketConn(c)}
	} else {
		p2 = PacketConn6{ipv6.NewPacketConn(c)}
	}

	for i := range ifaces {
		p2.JoinGroup(&ifaces[i], &net.UDPAddr{IP: group, Port: portNum})
	}

	go p.listen()
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()
	start := time.Now()

	for {
		p.RLock()
		if len(p.received) >= p.settings.Limit && p.settings.Limit > 0 {
			p.exit = true
		}
		p.RUnlock()

		if !s.DisableBroadcast {
			payload := p.settings.Payload
			if p.settings.PayloadFunc != nil {
				payload = p.settings.PayloadFunc()
			}
			// write to multicast
			broadcast(p2, payload, ifaces, &net.UDPAddr{IP: group, Port: portNum})
		}

		select {
		case <-p.settings.StopChan:
			p.exit = true
		case <-ticker.C:
		}

		if p.exit || timeLimit > 0 && time.Since(start) > timeLimit {
			break
		}
	}

	if !s.DisableBroadcast {
		payload := p.settings.Payload
		if p.settings.PayloadFunc != nil {
			payload = p.settings.PayloadFunc()
		}
		// send out broadcast that is finished
		broadcast(p2, payload, ifaces, &net.UDPAddr{IP: group, Port: portNum})
	}

	p.RLock()

	discoveries = make([]Discovered, len(p.received))
	i := 0
	for ip, peerState := range p.received {
		discoveries[i] = Discovered{
			Address:  ip,
			Payload:  peerState.lastPayload,
			Metadata: peerState.metadata,
		}
		i++
	}

	p.RUnlock()

	go p.gc()

	return p, discoveries, nil
}
