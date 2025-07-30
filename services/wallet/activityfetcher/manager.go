package activityfetcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	"go.uber.org/zap"
)

const (
	maxEntriesPerFetch = 1000
)

type ManagerIface interface {
	IsChainSupported(chainID uint64) bool
	FetchActivity(ctx context.Context, chainID uint64, account gethcommon.Address, currentBlock uint64) (thirdparty.ActivityEntryContainer, error)
}

type Manager struct {
	fetcher     thirdparty.ActivityFetcher
	persistence *Persistence
	logger      *zap.Logger
}

func NewManager(fetcher thirdparty.ActivityFetcher, persistence *Persistence) *Manager {
	return &Manager{
		fetcher:     fetcher,
		persistence: persistence,
		logger:      logutils.ZapLogger().Named("ActivityFetcher"),
	}
}

func (m *Manager) IsChainSupported(chainID uint64) bool {
	return m.fetcher.IsChainSupported(common.ChainID(chainID))
}

func (m *Manager) FetchActivity(ctx context.Context, chainID uint64, account gethcommon.Address, currentBlock uint64) (thirdparty.ActivityEntryContainer, error) {

	parameters := thirdparty.ActivityFetchParameters{
		Address:   account,
		Order:     thirdparty.NewToOld,
		Direction: thirdparty.Both,
	}
	// Get last fetched block
	lastFetchedBlock, _, err := m.persistence.GetLastFetchedBlockAndTimestamp(ctx, chainID, account)
	if err != nil {
		m.logger.Error("Failed to get last fetched block", zap.Error(err))
		return thirdparty.ActivityEntryContainer{}, err
	}

	toBlock := gethrpc.BlockNumber(currentBlock)
	parameters.ToBlock = &toBlock

	if lastFetchedBlock == nil {
		fromBlock := gethrpc.EarliestBlockNumber
		parameters.FromBlock = &fromBlock
	} else if uint64(lastFetchedBlock.Int64()) >= currentBlock {
		// Nothing to fetch
		return thirdparty.ActivityEntryContainer{}, nil
	} else {
		fromBlock := gethrpc.BlockNumber(*lastFetchedBlock + 1)
		parameters.FromBlock = &fromBlock
	}

	startTime := time.Now()
	m.logger.Debug("Fetching activity",
		zap.String("account", account.Hex()),
		zap.Uint64("chainID", chainID),
		zap.Int64("fromBlock", int64(*parameters.FromBlock)),
		zap.Int64("toBlock", int64(*parameters.ToBlock)),
	)

	if parameters.FromBlock == nil || parameters.FromBlock.Int64() < 0 {
		return thirdparty.ActivityEntryContainer{}, errors.New("fromBlock must be defined")
	}

	if parameters.ToBlock == nil || parameters.ToBlock.Int64() < 0 {
		return thirdparty.ActivityEntryContainer{}, errors.New("toBlock must be defined")
	}

	fmt.Println("Fetching activity", parameters)
	activity, err := m.fetcher.FetchActivity(ctx, chainID, parameters, "", maxEntriesPerFetch)
	if err != nil {
		return thirdparty.ActivityEntryContainer{}, err
	}
	fmt.Println("Fetched activity", len(activity.Items))

	err = m.persistence.SaveActivity(ctx, chainID, parameters, activity)
	if err != nil {
		return thirdparty.ActivityEntryContainer{}, err
	}

	duration := time.Since(startTime)
	m.logger.Debug("Fetch activity completed",
		zap.String("account", account.Hex()),
		zap.Uint64("chainID", chainID),
		zap.Int64("fromBlock", int64(*parameters.FromBlock)),
		zap.Int64("toBlock", int64(*parameters.ToBlock)),
		zap.Duration("duration", duration))

	return activity, nil
}
