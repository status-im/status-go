package activityfetcher

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const (
	maxEntriesPerFetch = 1000
)

// ActivityToken represents a token found in activity transfers
type ActivityToken struct {
	Address  gethcommon.Address
	IsNative bool
}

type ActivityTokenProvider interface {
	GetActivityTokens(chainID uint64, address gethcommon.Address) ([]ActivityToken, error)
}

type ManagerIface interface {
	IsChainSupported(chainID uint64) bool
	FetchActivity(ctx context.Context, chainID uint64, account gethcommon.Address, currentBlock uint64) (thirdparty.ActivityEntryContainer, error)
	GetLastFetchedBlockAndTimestamp(ctx context.Context, chainID uint64, address gethcommon.Address) (*gethrpc.BlockNumber, *time.Time, error)
	ClearAll(ctx context.Context) error
	ActivityTokenProvider
}

type Manager struct {
	fetcher thirdparty.ActivityFetcher
	logger  *zap.Logger
}

func NewManager(fetcher thirdparty.ActivityFetcher) *Manager {
	return &Manager{
		fetcher: fetcher,
		logger:  logutils.ZapLogger().Named("ActivityFetcher"),
	}
}

func (m *Manager) IsChainSupported(chainID uint64) bool {
	return m.fetcher.IsChainSupported(common.ChainID(chainID))
}

func (m *Manager) GetLastFetchedBlockAndTimestamp(ctx context.Context, chainID uint64, address gethcommon.Address) (*gethrpc.BlockNumber, *time.Time, error) {
	return m.fetcher.GetLastFetchedBlockAndTimestamp(ctx, chainID, address)
}

func (m *Manager) FetchActivity(ctx context.Context, chainID uint64, account gethcommon.Address, currentBlock uint64) (thirdparty.ActivityEntryContainer, error) {

	parameters := thirdparty.ActivityFetchParameters{
		Address:   account,
		Order:     thirdparty.NewToOld,
		Direction: thirdparty.Both,
	}
	// Get last fetched block
	lastFetchedBlock, _, err := m.fetcher.GetLastFetchedBlockAndTimestamp(ctx, chainID, account)
	if err != nil {
		m.logger.Error("Failed to get last fetched block", zap.Error(err))
		return thirdparty.ActivityEntryContainer{}, err
	}

	toBlock := gethrpc.BlockNumber(currentBlock)
	parameters.ToBlock = &toBlock

	if lastFetchedBlock == nil {
		fromBlock := gethrpc.BlockNumber(0)
		parameters.FromBlock = &fromBlock
	} else if uint64(lastFetchedBlock.Int64()) >= currentBlock {
		// Nothing to fetch
		return thirdparty.ActivityEntryContainer{}, nil
	} else {
		fromBlock := *lastFetchedBlock + 1
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

	activity, err := m.fetcher.FetchActivity(ctx, chainID, parameters, "", maxEntriesPerFetch)
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

func (m *Manager) ClearAll(ctx context.Context) error {
	return m.fetcher.ClearAll(ctx)
}

// GetActivityTokens returns unique tokens from activity transfers for a given account and chain
func (m *Manager) GetActivityTokens(chainID uint64, address gethcommon.Address) ([]ActivityToken, error) {
	tokens, err := m.fetcher.GetActivityTokens(context.Background(), chainID, address)
	if err != nil {
		return nil, err
	}

	// Convert ac.Token to ActivityToken
	result := make([]ActivityToken, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, ActivityToken{
			Address:  token.Address,
			IsNative: token.TokenType == 0, // Native is 0 in ac.TokenType
		})
	}

	return result, nil
}
