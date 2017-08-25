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
	ErrNodeSyncFailedToStart = errors.New("node synchronization failed to start")
	ErrNodeSyncTakesTooLong  = errors.New("node synchronization is taking too long")
)

// delays
var (
	// delayCycleForSyncStart sets the timeout for checking state of synchronization starting.
	delayCycleForSyncStart = 100 * time.Millisecond
)

// SyncPoll provides a structure that allows us to check the status of
// ethereum node synchronization.
type SyncPoll struct {
	eth *les.LightEthereum
}

// NewSyncPoll returns a new instance of SyncPoll.
func NewSyncPoll(leth *les.LightEthereum) *SyncPoll {
	return &SyncPoll{
		eth: leth,
	}
}

// Poll returns a channel which allows the user to listen for a done signal
// as to when the node has finished syncing or stop due to an error.
func (n *SyncPoll) Poll(ctx context.Context) error {
	errChan := make(chan error)
	downloader := n.eth.Downloader()

	syncStart := make(chan struct{})
	go n.pollSyncStart(ctx, syncStart, errChan, downloader)

	// Block to be notified whether error occured or if sync has started
	select {
	case err := <-errChan:
		return err
	case <-syncStart:
	}

	syncCompleted := make(chan struct{})
	go n.pollSyncCompleted(ctx, syncCompleted, errChan, downloader)

	// Block to be notified if node failed to complete sync or if context has expired.
	select {
	case err := <-errChan:
		return err
	case <-syncCompleted:
	}

	return nil
}

func (n *SyncPoll) pollSyncStart(ctx context.Context, syncStart chan struct{}, errorChan chan error, downloader *downloader.Downloader) {
	for {
		select {
		case <-ctx.Done():
			errorChan <- ErrNodeSyncFailedToStart
			return
		case <-time.After(delayCycleForSyncStart):
			if downloader.Synchronising() {
				close(syncStart)
				return
			}
		}
	}
}

func (n *SyncPoll) pollSyncCompleted(ctx context.Context, doneChan chan struct{}, errorChan chan error, downloader *downloader.Downloader) {
	for {
		select {
		case <-ctx.Done():
			errorChan <- ErrNodeSyncTakesTooLong
			return
		case <-time.After(delayCycleForSyncStart):
			progress := downloader.Progress()
			if progress.CurrentBlock >= progress.HighestBlock {
				close(doneChan)
				return
			}
		}
	}
}
