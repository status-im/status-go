package wallet

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// SetupIterativeDownloader configures IterativeDownloader with last known synced block.
func SetupIterativeDownloader(
	db *Database, client HeaderReader, option SyncOption,
	downloader BatchDownloader, size *big.Int) (*IterativeDownloader, error) {
	d := &IterativeDownloader{
		client:     client,
		batchSize:  size,
		downloader: downloader,
	}
	earliest, err := db.GetEarliestSynced(option)
	if err != nil {
		log.Error("failed to get earliest synced block", "error", err)
		return nil, err
	}
	if earliest == nil {
		previous, err := lastKnownHeader(db, client)
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
func (d *IterativeDownloader) Next() ([]Transfer, error) {
	start := new(big.Int).Sub(d.known.Number, d.batchSize)
	// if start < 0; start = 0
	if start.Cmp(big.NewInt(0)) <= 0 {
		start = big.NewInt(0)
	}
	from, err := d.client.HeaderByNumber(context.Background(), start)
	if err != nil {
		log.Error("failed to get header by number", "number", start, "error", err)
		return nil, err
	}
	transfers, err := d.downloader.GetTransfersInRange(context.Background(), start, d.known.Number)
	if err != nil {
		log.Error("failed to get transfer inbetween two bloks", "from", start, "to", d.known.Number, "error", err)
		return nil, err
	}
	// use integers instead of DBHeader
	d.previous, d.known = d.known, toDBHeader(from)
	return transfers, nil
}

// Revert reverts last step progress. Should be used if application failued to process transfers.
// For example failed to persist them.
func (d *IterativeDownloader) Revert() {
	if d.previous != nil {
		d.known = d.previous
	}
}

func SetupBinaryIterativeDownloader(db *Database, client *ethclient.Client, address common.Address,
	option SyncOption, downloader BatchDownloader) (*BinaryIterativeDownloader, error) {
	d := &BinaryIterativeDownloader{
		client:     client,
		downloader: downloader,
		address:    address,
	}
	earliest, err := db.GetEarliestSynced(option)
	if err != nil {
		log.Error("failed to get earliest synced block", "error", err)
		return nil, err
	}
	if earliest == nil {
		previous, err := lastKnownHeader(db, client)
		if err != nil {
			log.Error("failed to get last known header", "error", err)
			return nil, err
		}
		earliest = previous
	}
	d.lastDownloaded = earliest
	d.low = big.NewInt(0)
	d.high = earliest.Number
	return d, nil
}

// BinaryIterativeDownloader uses approach similar to binary search to find differences balance differences between several blocks.
type BinaryIterativeDownloader struct {
	client                       *ethclient.Client
	address                      common.Address
	downloader                   BatchDownloader
	high, low, prevHigh, prevLow *big.Int
	lastDownloaded               *DBHeader
}

func (d *BinaryIterativeDownloader) updateLastDownloaded() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	header, err := d.client.HeaderByNumber(ctx, d.high)
	cancel()
	if err != nil {
		return err
	}
	d.lastDownloaded = toDBHeader(header)
	return nil
}

func (d *BinaryIterativeDownloader) Next() ([]Transfer, error) {
	log.Debug("comparing balances between", "low", d.low, "high", d.high)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	hbalance, err := d.client.BalanceAt(ctx, d.address, d.high)
	cancel()
	if err != nil {
		return nil, err
	}
	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	lbalance, err := d.client.BalanceAt(ctx, d.address, d.low)
	cancel()
	if err != nil {
		return nil, err
	}
	if lbalance.Cmp(hbalance) != 0 {
		log.Debug("balances between are not equal",
			"low", d.low, "high", d.high,
			"diff", new(big.Int).Sub(hbalance, lbalance))
		if new(big.Int).Sub(d.high, d.low).Cmp(one) == 0 {
			log.Debug("higher block is a direct child. downloading transfers")
			// TODO(dshulyak) maybe use downloader for single block
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			transfers, err := d.downloader.GetTransfersInRange(ctx, d.low, d.high)
			cancel()
			if err != nil {
				return nil, err
			}
			err = d.updateLastDownloaded()
			if err != nil {
				return nil, err
			}
			// for example transfers found between 49 and 50
			// set high = 49 and low = 25 instead of 49 and 0
			d.prevHigh, d.prevLow = d.high, d.low
			d.high = d.low
			d.low = new(big.Int).Div(d.high, big.NewInt(2))
			return transfers, nil
		}
		d.prevHigh, d.prevLow = d.high, d.low
		mid := new(big.Int).Add(d.high, d.low)
		err = d.updateLastDownloaded()
		if err != nil {
			return nil, err
		}
		d.high = d.low
		d.low = mid.Div(mid, big.NewInt(2))
		if d.low.Cmp(one) >= 0 {
			d.low = d.low.Sub(d.low, one)
		}
		return nil, nil
	}
	log.Debug("balances between are equal",
		"low", d.low, "high", d.high)
	// TODO(dshulyak) DRY
	d.prevHigh, d.prevLow = d.high, d.low
	mid := new(big.Int).Add(d.high, d.low)
	err = d.updateLastDownloaded()
	if err != nil {
		return nil, err
	}
	d.high = d.low
	d.low = mid.Div(mid, big.NewInt(2))
	if d.low.Cmp(one) >= 0 {
		d.low = d.low.Sub(d.low, one)
	}
	return nil, nil
}

func (d *BinaryIterativeDownloader) Finished() bool {
	return d.high.Cmp(zero) == 0
}

func (d *BinaryIterativeDownloader) Header() *DBHeader {
	return d.lastDownloaded
}

func (d *BinaryIterativeDownloader) Revert() {
	if d.prevHigh != nil {
		d.high, d.low = d.prevHigh, d.prevLow
	}
}
