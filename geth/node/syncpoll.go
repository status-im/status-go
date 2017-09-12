package node

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/les"
)

// errors
var (
	ErrSyncStartTimeout = errors.New("node synchronization timeout before start")
	ErrSyncAborted      = errors.New("node synchronization timeout before completion")
)

// delays
var (
	// delayCycleForSyncStart sets the timeout for checking state of synchronization starting.
	delayCycleForSyncStart = 100 * time.Millisecond
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

// Poll returns a channel which allows the user to listen for a done signal
// as to when the node has finished syncing or stop due to an error.
func (n *SyncPoll) Poll(ctx context.Context) error {
	if err := n.pollSyncStart(ctx); err != nil {
		return err
	}

	if err := n.pollSyncCompleted(ctx); err != nil {
		return err
	}

	return nil
}

func (n *SyncPoll) pollSyncStart(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ErrSyncStartTimeout
		case <-time.After(100 * time.Millisecond):
			if n.downloader.Synchronising() {
				return nil
			}
		}
	}
}

func (n *SyncPoll) pollSyncCompleted(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ErrSyncAborted
		case <-time.After(100 * time.Millisecond):
			progress := n.downloader.Progress()
			if progress.CurrentBlock >= progress.HighestBlock {
				return nil
			}
		}
	}
}
