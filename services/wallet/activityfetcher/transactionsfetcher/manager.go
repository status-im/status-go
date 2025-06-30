package activityfetcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const (
	maxEntriesPerFetch = 1000
)

type ManagerIface interface {
	IsChainSupported(chainID uint64) bool
	FetchActivity(ctx context.Context, chainID uint64, parameters thirdparty.ActivityFetchParameters, cursor string) (thirdparty.ActivityEntryContainer, error)
}

type Manager struct {
	fetcher     thirdparty.ActivityFetcher
	persistence *Persistence
}

func NewManager(fetcher thirdparty.ActivityFetcher, persistence *Persistence) *Manager {
	return &Manager{
		fetcher:     fetcher,
		persistence: persistence,
	}
}

func (m *Manager) IsChainSupported(chainID uint64) bool {
	return m.fetcher.IsChainSupported(common.ChainID(chainID))
}

func (m *Manager) FetchActivity(ctx context.Context, chainID uint64, parameters thirdparty.ActivityFetchParameters, cursor string) (thirdparty.ActivityEntryContainer, error) {
	if parameters.FromBlock == nil || parameters.FromBlock.Int64() < 0 {
		return thirdparty.ActivityEntryContainer{}, errors.New("fromBlock must be defined")
	}

	if parameters.ToBlock == nil || parameters.ToBlock.Int64() < 0 {
		return thirdparty.ActivityEntryContainer{}, errors.New("toBlock must be defined")
	}

	fmt.Println("Fetching activity", parameters)
	activity, err := m.fetcher.FetchActivity(ctx, chainID, parameters, cursor, maxEntriesPerFetch)
	if err != nil {
		return thirdparty.ActivityEntryContainer{}, err
	}
	fmt.Println("Fetched activity", len(activity.Items))

	err = m.persistence.SaveActivity(ctx, chainID, parameters, activity)
	if err != nil {
		return thirdparty.ActivityEntryContainer{}, err
	}

	return activity, nil
}
