package wallet

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type HeaderReader interface {
	HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

// Reactor listens to new blocks and stores transfers into the database.
type Reactor struct {
	client HeaderReader
	db     *Database

	eth   *ETHTransferDownloader
	erc20 *ERC20TransfersDownloader

	wg   sync.WaitGroup
	quit chan struct{}
}

func (r *Reactor) Start() error {
	if r.quit != nil {
		return errors.New("already running")
	}
	r.quit = make(chan struct{})
	r.wg.Add(1)
	go func() {
		r.loop()
		r.wg.Done()
	}()
	return nil
}

func (r *Reactor) Stop() {
	if r.quit == nil {
		return
	}
	close(r.quit)
	r.wg.Wait()
	r.quit = nil
}

func (r *Reactor) loop() {
	ticker := time.NewTicker(5 * time.Second)
	var (
		latest *types.Header
		err    error
	)
	for {
		select {
		case <-r.quit:
			return
		case <-ticker.C:
			var num *big.Int
			if latest == nil {
				latest, err = r.db.LastHeader()
				if err != nil {
					continue
				}
			}
			if latest != nil {
				num = new(big.Int).Add(latest.Number, one)
			}
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			header, err := r.client.HeaderByNumber(ctx, num)
			cancel()
			if err != nil {
				log.Error("failed to get latest block", "number", latest, "error", err)
				continue
			}
			ctx = context.Background()
			ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
			added, removed, err := r.onNewBlock(ctx, latest, header)
			cancel()
			if err != nil {
				log.Error("failed to process new header", "header", header, "error", err)
				continue
			}
			// for each added block get tranfers from downloaders
			all := []Transfer{}
			for i := range added {
				transfers, err := r.getTransfers(added[i])
				if err != nil {
					log.Error("failed to get transfers", "header", header, "error", err)
					continue
				}
				all = append(all, transfers...)
			}
			err = r.db.ProcessTranfers(all, added, removed)
			if err != nil {
				log.Error("failed to persist transfers", "error", err)
				continue
			}
			latest = header
		}
	}
}

func (r *Reactor) getTransfers(header *types.Header) ([]Transfer, error) {
	ethT, err := r.eth.GetTransfers(context.TODO(), header)
	if err != nil {
		return nil, err
	}
	erc20T, err := r.erc20.GetTransfers(context.TODO(), header)
	if err != nil {
		return nil, err
	}
	return append(ethT, erc20T...), nil
}

func (r *Reactor) onNewBlock(ctx context.Context, previous, latest *types.Header) (added, removed []*types.Header, err error) {
	if previous == nil {
		// first node in the cache
		return []*types.Header{latest}, nil, nil
	}
	if previous.Hash() == latest.ParentHash {
		// parent matching previous node in the cache. on the same chain.
		return []*types.Header{latest}, nil, nil
	}
	exists, err := r.db.HeaderExists(latest.Hash())
	if err != nil {
		return nil, nil, err
	}
	if exists {
		return nil, nil, err
	}
	// reorg
	for previous.Hash() != latest.ParentHash {
		removed = append(removed, previous)
		added = append(added, latest)
		latest, err = r.client.HeaderByHash(context.TODO(), latest.ParentHash)
		if err != nil {
			return nil, nil, err
		}
		previous, err = r.db.GetHeaderByNumber(new(big.Int).Add(latest.Number, one.Neg(one)))
		if err != nil {
			return nil, nil, err
		}
	}
	return added, removed, nil
}
