package rendezvous

import (
	"github.com/ethereum/go-ethereum/metrics"
	inet "github.com/libp2p/go-libp2p-net"
)

var (
	ingressTrafficMeter = metrics.NewRegisteredMeter("rendezvous/InboundTraffic", nil)
	egressTrafficMeter  = metrics.NewRegisteredMeter("rendezvous/OutboundTraffic", nil)
)

// InstrumenetedStream implements read writer interface and collects metrics.
type InstrumenetedStream struct {
	s inet.Stream
}

func (si InstrumenetedStream) Write(p []byte) (int, error) {
	n, err := si.s.Write(p)
	egressTrafficMeter.Mark(int64(n))
	return n, err
}

func (si InstrumenetedStream) Read(p []byte) (int, error) {
	n, err := si.s.Read(p)
	ingressTrafficMeter.Mark(int64(n))
	return n, err
}

func (si InstrumenetedStream) Close() error {
	return si.s.Close()
}
