package wallet

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// SetupIterativeDownloader configures IterativeDownloader with last known synced block.
func SetupIterativeDownloader(
	db *Database, client HeaderReader, address common.Address, option SyncOption,
	downloader BatchDownloader, size *big.Int, limit *big.Int) (*IterativeDownloader, error) {
	d := &IterativeDownloader{
		client:     client,
		batchSize:  size,
		downloader: downloader,
	}
	earliest, err := db.GetEarliestSynced(address, option)
	if err != nil {
		log.Error("failed to get earliest synced block", "error", err)
		return nil, err
	}
	if earliest == nil {
		previous, err := lastKnownHeader(context.Background(), db, client, limit)
		if err != nil {
			log.Error("failed to get last known header", "error", err)
			return nil, err
		}
		earliest = previous
	}
	d.known = earliest
	return d, nil
}

// BatchDownloader interface for loading transfers in batches in speificed range of blocks.
type BatchDownloader interface {
	GetTransfersInRange(ctx context.Context, from, to *big.Int) ([]Transfer, error)
}

// IterativeDownloader downloads batches of transfers in a specified size.
type IterativeDownloader struct {
	client HeaderReader

	batchSize *big.Int

	downloader BatchDownloader

	known    *DBHeader
	previous *DBHeader
}

// Finished true when earliest block with given sync option is zero.
func (d *IterativeDownloader) Finished() bool {
	return d.known.Number.Cmp(big.NewInt(0)) == 0
}

// Header return last synced header.
func (d *IterativeDownloader) Header() *DBHeader {
	return d.known
}

// Next moves closer to the end on every new iteration.
func (d *IterativeDownloader) Next(parent context.Context) ([]Transfer, error) {
	start := new(big.Int).Sub(d.known.Number, d.batchSize)
	// if start < 0; start = 0
	if start.Cmp(big.NewInt(0)) <= 0 {
		start = big.NewInt(0)
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	from, err := d.client.HeaderByNumber(ctx, start)
	cancel()
	if err != nil {
		log.Error("failed to get header by number", "number", start, "error", err)
		return nil, err
	}
	transfers, err := d.downloader.GetTransfersInRange(parent, start, d.known.Number)
	if err != nil {
		log.Error("failed to get transfer inbetween two bloks", "from", start, "to", d.known.Number, "error", err)
		return nil, err
	}
	// use integers instead of DBHeader
	d.previous, d.known = d.known, toDBHeader(from)
	return transfers, nil
}

// Revert reverts last step progress. Should be used if application failed to process transfers.
// For example failed to persist them.
func (d *IterativeDownloader) Revert() {
	if d.previous != nil {
		d.known = d.previous
	}
}
