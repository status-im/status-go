package node

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/les"
	"github.com/status-im/status-go/geth/log"
)

// errors
var (
	ErrStartAborted = errors.New("node synchronization timeout before starting")
	ErrSyncAborted  = errors.New("node synchronization timeout before completion")
)

// SyncPoll provides a structure that allows us to check the status of
// ethereum node synchronization.
type SyncPoll struct {
	downloader *downloader.Downloader
}

// NewSyncPoll returns a new instance of SyncPoll.
func NewSyncPoll(leth *les.LightEthereum) *SyncPoll {
	return &SyncPoll{
		downloader: leth.Downloader(),
	}
}

// Poll checks for the status of blockchain synchronization and returns an error
// if the blockchain failed to start synchronizing or fails to complete, within
// the time provided by the passed in context.
func (n *SyncPoll) Poll(ctx context.Context) error {
	if err := n.pollSyncStart(ctx); err != nil {
		return err
	}

	if err := n.waitSyncCompleted(ctx); err != nil {
		return err
	}

	return nil
}

func (n *SyncPoll) pollSyncStart(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ErrStartAborted
		case <-time.After(100 * time.Millisecond):
			if n.downloader.Synchronising() {
				log.Info("Block synchronization progress just started")
				return nil
			}
		}
	}
}

func (n *SyncPoll) waitSyncCompleted(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ErrSyncAborted
		case <-time.After(100 * time.Millisecond):
			progress := n.downloader.Progress()

			// If information on highest block has not being retrieved then wait.
			if progress.HighestBlock <= 0 && progress.StartingBlock <= 0 {
				time.Sleep(300 * time.Millisecond)
				continue
			}

			if progress.CurrentBlock >= progress.HighestBlock {
				log.Info("Block synchronization just finished")
				return nil
			}
		}
	}
}
