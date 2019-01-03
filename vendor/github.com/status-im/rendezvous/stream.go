package rendezvous

import (
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	inet "github.com/libp2p/go-libp2p-net"
	protocol "github.com/libp2p/go-libp2p-protocol"
)

var (
	ingressTrafficMeter = metrics.NewRegisteredMeter("rendezvous/InboundTraffic", nil)
	egressTrafficMeter  = metrics.NewRegisteredMeter("rendezvous/OutboundTraffic", nil)
)

// InstrumentedStream implements read writer interface and collects metrics.
type InstrumentedStream struct {
	s inet.Stream
}

func (si InstrumentedStream) Write(p []byte) (int, error) {
	n, err := si.s.Write(p)
	egressTrafficMeter.Mark(int64(n))
	return n, err
}

func (si InstrumentedStream) Read(p []byte) (int, error) {
	n, err := si.s.Read(p)
	ingressTrafficMeter.Mark(int64(n))
	return n, err
}

func (si InstrumentedStream) Close() error {
	return si.s.Close()
}

func (si InstrumentedStream) Reset() error {
	return si.s.Reset()
}

func (si InstrumentedStream) SetDeadline(timeout time.Time) error {
	return si.s.SetDeadline(timeout)
}

func (si InstrumentedStream) SetReadDeadline(timeout time.Time) error {
	return si.s.SetReadDeadline(timeout)
}

func (si InstrumentedStream) SetWriteDeadline(timeout time.Time) error {
	return si.s.SetWriteDeadline(timeout)
}

func (si InstrumentedStream) Protocol() protocol.ID {
	return si.s.Protocol()
}

func (si InstrumentedStream) SetProtocol(pid protocol.ID) {
	si.s.SetProtocol(pid)
}

func (si InstrumentedStream) Conn() inet.Conn {
	return si.s.Conn()
}
