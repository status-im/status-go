package httpserver

import "net"

// ListeningAddr returns the address the server is currently bound to, or nil when
// it is not listening.
func (s *Server) ListeningAddr() *net.TCPAddr {
	return s.address
}

// CachedPort returns the port remembered from the last successful bind. It is what
// keeps URLs stable across ToBackground/ToForeground cycles, and is zero when nothing
// has been cached yet.
func (s *Server) CachedPort() int {
	return s.cachedPort
}

// SetURLStateForTest overrides the address, cached port and config that URL
// construction reads, so that packages embedding Server can exercise URL
// building without binding a socket. Test-only; do not call it elsewhere.
func (s *Server) SetURLStateForTest(address *net.TCPAddr, cachedPort int, config *Config) {
	s.address = address
	s.cachedPort = cachedPort
	s.config = config
}
