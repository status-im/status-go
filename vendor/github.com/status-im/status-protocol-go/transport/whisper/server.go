package whisper

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/pkg/errors"
)

type Server interface {
	Connected(enode.ID) (bool, error)
	AddPeer(string) error
	NodeID() *ecdsa.PrivateKey
}

// dialOpts used in Dial function.
type dialOpts struct {
	// PollInterval is used for time.Ticker. Must be greated then zero.
	PollInterval time.Duration
}

// dial selected peer and wait until it is connected.
func dial(ctx context.Context, srv Server, peer string, opts dialOpts) error {
	if opts.PollInterval == 0 {
		return errors.New("poll interval cannot be zero")
	}
	if err := srv.AddPeer(peer); err != nil {
		return err
	}
	parsed, err := enode.ParseV4(peer)
	if err != nil {
		return err
	}
	connected, err := srv.Connected(parsed.ID())
	if err != nil {
		return err
	}
	if connected {
		return nil
	}
	period := time.NewTicker(opts.PollInterval)
	defer period.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-period.C:
			connected, err := srv.Connected(parsed.ID())
			if err != nil {
				return err
			}
			if connected {
				return nil
			}
		}
	}
}
