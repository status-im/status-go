package quicreuse

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/quic-go/quic-go"
	quiclogging "github.com/quic-go/quic-go/logging"
)

var quicDialContext = quic.DialContext // so we can mock it in tests

type ConnManager struct {
	reuseUDP4       *reuse
	reuseUDP6       *reuse
	enableDraft29   bool
	enableReuseport bool
	enableMetrics   bool

	serverConfig *quic.Config
	clientConfig *quic.Config

	connsMu sync.Mutex
	conns   map[string]connListenerEntry
}

type connListenerEntry struct {
	refCount int
	ln       *connListener
}

func NewConnManager(statelessResetKey quic.StatelessResetKey, opts ...Option) (*ConnManager, error) {
	cm := &ConnManager{
		enableReuseport: true,
		enableDraft29:   true,
		conns:           make(map[string]connListenerEntry),
	}
	for _, o := range opts {
		if err := o(cm); err != nil {
			return nil, err
		}
	}

	quicConf := quicConfig.Clone()
	quicConf.StatelessResetKey = &statelessResetKey

	var tracers []quiclogging.Tracer
	if qlogTracer != nil {
		tracers = append(tracers, qlogTracer)
	}
	if cm.enableMetrics {
		tracers = append(tracers, newMetricsTracer())
	}
	if len(tracers) > 0 {
		quicConf.Tracer = quiclogging.NewMultiplexedTracer(tracers...)
	}
	serverConfig := quicConf.Clone()
	if !cm.enableDraft29 {
		serverConfig.Versions = []quic.VersionNumber{quic.Version1}
	}

	cm.clientConfig = quicConf
	cm.serverConfig = serverConfig
	if cm.enableReuseport {
		cm.reuseUDP4 = newReuse()
		cm.reuseUDP6 = newReuse()
	}
	return cm, nil
}

func (c *ConnManager) getReuse(network string) (*reuse, error) {
	switch network {
	case "udp4":
		return c.reuseUDP4, nil
	case "udp6":
		return c.reuseUDP6, nil
	default:
		return nil, errors.New("invalid network: must be either udp4 or udp6")
	}
}

func (c *ConnManager) ListenQUIC(addr ma.Multiaddr, tlsConf *tls.Config, allowWindowIncrease func(conn quic.Connection, delta uint64) bool) (Listener, error) {
	if !c.enableDraft29 {
		if _, err := addr.ValueForProtocol(ma.P_QUIC); err == nil {
			return nil, errors.New("can't listen on `/quic` multiaddr (QUIC draft 29 version) when draft 29 support is disabled")
		}
	}

	netw, host, err := manet.DialArgs(addr)
	if err != nil {
		return nil, err
	}
	laddr, err := net.ResolveUDPAddr(netw, host)
	if err != nil {
		return nil, err
	}

	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	key := laddr.String()
	entry, ok := c.conns[key]
	if !ok {
		conn, err := c.listen(netw, laddr)
		if err != nil {
			return nil, err
		}
		ln, err := newConnListener(conn, c.serverConfig, c.enableDraft29)
		if err != nil {
			return nil, err
		}
		key = conn.LocalAddr().String()
		entry = connListenerEntry{ln: ln}
	}
	l, err := entry.ln.Add(tlsConf, allowWindowIncrease, func() { c.onListenerClosed(key) })
	if err != nil {
		if entry.refCount <= 0 {
			entry.ln.Close()
		}
		return nil, err
	}
	entry.refCount++
	c.conns[key] = entry
	return l, nil
}

func (c *ConnManager) onListenerClosed(key string) {
	c.connsMu.Lock()
	defer c.connsMu.Unlock()

	entry := c.conns[key]
	entry.refCount = entry.refCount - 1
	if entry.refCount <= 0 {
		delete(c.conns, key)
		entry.ln.Close()
	} else {
		c.conns[key] = entry
	}
}

func (c *ConnManager) listen(network string, laddr *net.UDPAddr) (pConn, error) {
	if c.enableReuseport {
		reuse, err := c.getReuse(network)
		if err != nil {
			return nil, err
		}
		return reuse.Listen(network, laddr)
	}

	conn, err := net.ListenUDP(network, laddr)
	if err != nil {
		return nil, err
	}
	return &noreuseConn{conn}, nil
}

func (c *ConnManager) DialQUIC(ctx context.Context, raddr ma.Multiaddr, tlsConf *tls.Config, allowWindowIncrease func(conn quic.Connection, delta uint64) bool) (quic.Connection, error) {
	naddr, v, err := FromQuicMultiaddr(raddr)
	if err != nil {
		return nil, err
	}
	netw, host, err := manet.DialArgs(raddr)
	if err != nil {
		return nil, err
	}

	quicConf := c.clientConfig.Clone()
	quicConf.AllowConnectionWindowIncrease = allowWindowIncrease

	if v == quic.Version1 {
		// The endpoint has explicit support for QUIC v1, so we'll only use that version.
		quicConf.Versions = []quic.VersionNumber{quic.Version1}
	} else if v == quic.VersionDraft29 {
		quicConf.Versions = []quic.VersionNumber{quic.VersionDraft29}
	} else {
		return nil, errors.New("unknown QUIC version")
	}

	pconn, err := c.Dial(netw, naddr)
	if err != nil {
		return nil, err
	}
	conn, err := quicDialContext(ctx, pconn, naddr, host, tlsConf, quicConf)
	if err != nil {
		pconn.DecreaseCount()
		return nil, err
	}
	return conn, nil
}

func (c *ConnManager) Dial(network string, raddr *net.UDPAddr) (pConn, error) {
	if c.enableReuseport {
		reuse, err := c.getReuse(network)
		if err != nil {
			return nil, err
		}
		return reuse.Dial(network, raddr)
	}

	var laddr *net.UDPAddr
	switch network {
	case "udp4":
		laddr = &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	case "udp6":
		laddr = &net.UDPAddr{IP: net.IPv6zero, Port: 0}
	}
	conn, err := net.ListenUDP(network, laddr)
	if err != nil {
		return nil, err
	}
	return &noreuseConn{conn}, nil
}

func (c *ConnManager) Protocols() []int {
	if c.enableDraft29 {
		return []int{ma.P_QUIC, ma.P_QUIC_V1}
	}
	return []int{ma.P_QUIC_V1}
}

func (c *ConnManager) Close() error {
	if !c.enableReuseport {
		return nil
	}
	if err := c.reuseUDP6.Close(); err != nil {
		return err
	}
	return c.reuseUDP4.Close()
}
