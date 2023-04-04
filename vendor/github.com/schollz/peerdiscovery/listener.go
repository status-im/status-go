package peerdiscovery

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	// https://en.wikipedia.org/wiki/User_Datagram_Protocol#Packet_structure
	maxDatagramSize = 66507
)

// PeerState is the state of a peer that has been discovered.
// It contains the address of the peer, the last time it was seen,
// the last payload it sent, and the metadata associated with it.
// To update the metadata, assign your own metadata to the Metadata.Data field.
// The metadata is not protected by a mutex, so you must do this yourself.
type PeerState struct {
	Address     string
	lastSeen    time.Time
	lastPayload []byte
	metadata    *Metadata
}

type LostPeer struct {
	Address     string
	LastSeen    time.Time
	LastPayload []byte
	Metadata    *Metadata
}

func (p *PeerDiscovery) gc() {
	ticker := time.NewTicker(p.settings.Delay * 2)
	defer ticker.Stop()

	for range ticker.C {
		p.Lock()
		for ip, peerState := range p.received {
			if time.Since(peerState.lastSeen) > p.settings.Delay*4 {
				if p.settings.NotifyLost != nil {
					p.settings.NotifyLost(LostPeer{
						Address:     ip,
						LastSeen:    peerState.lastSeen,
						LastPayload: peerState.lastPayload,
						Metadata:    peerState.metadata,
					})
				}

				delete(p.received, ip)
			}
		}
		p.Unlock()
	}
}

// PeerDiscovery is the object that can do the discovery for finding LAN peers.
type PeerDiscovery struct {
	settings Settings

	received map[string]*PeerState
	sync.RWMutex
	exit bool
}

func (p *PeerDiscovery) Shutdown() {
	p.exit = true
}

func (p *PeerDiscovery) ActivePeers() (peers []*PeerState) {
	p.RLock()
	defer p.RUnlock()
	for _, peerState := range p.received {
		peers = append(peers, peerState)
	}
	return
}

// Listen binds to the UDP address and port given and writes packets received
// from that address to a buffer which is passed to a hander
func (p *PeerDiscovery) listen() (recievedBytes []byte, err error) {
	p.RLock()
	address := net.JoinHostPort(p.settings.MulticastAddress, p.settings.Port)
	portNum := p.settings.portNum
	allowSelf := p.settings.AllowSelf
	timeLimit := p.settings.TimeLimit
	notify := p.settings.Notify
	p.RUnlock()
	localIPs := getLocalIPs()

	// get interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	// log.Println(ifaces)

	// Open up a connection
	c, err := net.ListenPacket(fmt.Sprintf("udp%d", p.settings.IPVersion), address)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	group := p.settings.multicastAddressNumbers
	var p2 NetPacketConn
	if p.settings.IPVersion == IPv4 {
		p2 = PacketConn4{ipv4.NewPacketConn(c)}
	} else {
		p2 = PacketConn6{ipv6.NewPacketConn(c)}
	}

	for i := range ifaces {
		p2.JoinGroup(&ifaces[i], &net.UDPAddr{IP: group, Port: portNum})
	}

	start := time.Now()
	// Loop forever reading from the socket
	for {
		buffer := make([]byte, maxDatagramSize)
		var (
			n       int
			src     net.Addr
			errRead error
		)
		n, src, errRead = p2.ReadFrom(buffer)
		if errRead != nil {
			err = errRead
			return
		}

		srcHost, _, _ := net.SplitHostPort(src.String())

		if _, ok := localIPs[srcHost]; ok && !allowSelf {
			continue
		}

		// log.Println(src, hex.Dump(buffer[:n]))

		p.Lock()
		if peer, ok := p.received[srcHost]; ok {
			peer.lastSeen = time.Now()
			peer.lastPayload = buffer[:n]
		} else {
			p.received[srcHost] = &PeerState{
				Address:     srcHost,
				lastPayload: buffer[:n],
				lastSeen:    time.Now(),
				metadata:    &Metadata{},
			}
		}
		p.Unlock()

		if notify != nil {
			notify(Discovered{
				Address: srcHost,
				Payload: buffer[:n],
			})
		}

		p.RLock()
		if len(p.received) >= p.settings.Limit && p.settings.Limit > 0 {
			p.RUnlock()
			break
		}
		if p.exit || timeLimit > 0 && time.Since(start) > timeLimit {
			p.RUnlock()
			break
		}
		p.RUnlock()
	}

	return
}
