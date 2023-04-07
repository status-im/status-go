package websocket

import (
	"net"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/transport"
)

const maxReadAttempts = 5

type conn struct {
	net.Conn
	readAttempts uint8
	localAddr    addrWrapper
	remoteAddr   addrWrapper
}

var _ net.Conn = conn{}

func (c conn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c conn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c conn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if err == nil && n == 0 && c.readAttempts < maxReadAttempts {
		c.readAttempts++
		// Nothing happened, let's read again. We reached the end of the frame
		// (https://github.com/nhooyr/websocket/blob/master/netconn.go#L118).
		// The next read will block until we get
		// the next frame. We limit here to avoid looping in case of a bunch of
		// empty frames.  Would be better if the websocket library did not
		// return 0, nil here (see https://github.com/nhooyr/websocket/issues/367).  But until, then this is our workaround.
		return c.Read(b)
	}
	return n, err
}

type capableConn struct {
	transport.CapableConn
}

func (c *capableConn) ConnState() network.ConnectionState {
	cs := c.CapableConn.ConnState()
	cs.Transport = "websocket"
	return cs
}
