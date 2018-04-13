package peers

import (
	"net"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
)

// StartDiscv5 starts discv5 udp listener.
// This is done here to avoid patching p2p server, we can't hold a lock of course
// but no other sub-process should use discovery
func StartDiscv5(server *p2p.Server) (*discv5.Network, error) {
	addr, err := net.ResolveUDPAddr("udp", server.ListenAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}
	realaddr := conn.LocalAddr().(*net.UDPAddr)
	ntab, err := discv5.ListenUDP(server.PrivateKey, conn, realaddr, "", server.NetRestrict)
	if err != nil {
		return nil, err
	}
	if err := ntab.SetFallbackNodes(server.BootstrapNodesV5); err != nil {
		return nil, err
	}
	return ntab, nil
}
