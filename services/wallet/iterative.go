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
	downloader BatchDownloader, size *big.Int, to *DBHeader) (*IterativeDownloader, error) {
	from, err := db.GetLatestSynced(address, option)
	if err != nil {
		log.Error("failed to get latest synced block", "error", err)
		return nil, err
	}
	if from == nil {
		from = &DBHeader{Number: zero}
	}
	log.Debug("iterative downloader", "address", address, "from", from.Number, "to", to.Number)
	d := &IterativeDownloader{
		client:     client,
		batchSize:  size,
		downloader: downloader,
		from:       from,
		to:         to,
	}
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

	from, to *DBHeader
	previous *DBHeader
}

// Finished true when earliest block with given sync option is zero.
func (d *IterativeDownloader) Finished() bool {
	return d.from.Number.Cmp(d.to.Number) == 0
}

// Header return last synced header.
func (d *IterativeDownloader) Header() *DBHeader {
	return d.previous
}

// Next moves closer to the end on every new iteration.
func (d *IterativeDownloader) Next(parent context.Context) ([]Transfer, error) {
	to := new(big.Int).Add(d.from.Number, d.batchSize)
	// if start < 0; start = 0
	if to.Cmp(d.to.Number) == 1 {
		to = d.to.Number
	}
	transfers, err := d.downloader.GetTransfersInRange(parent, d.from.Number, to)
	if err != nil {
		log.Error("failed to get transfer inbetween two bloks", "from", d.from.Number, "to", to, "error", err)
		return nil, err
	}
	// use integers instead of DBHeader
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	header, err := d.client.HeaderByNumber(ctx, to)
	cancel()
	if err != nil {
		log.Error("failed to get header by number", "from", d.from.Number, "to", to, "error", err)
		return nil, err
	}
	d.previous, d.from = d.from, toDBHeader(header)
	return transfers, nil
}

// Revert reverts last step progress. Should be used if application failed to process transfers.
// For example failed to persist them.
func (d *IterativeDownloader) Revert() {
	if d.previous != nil {
		d.from = d.previous
	}
}
