package activity

import (

	// used for embedding the sql query in the binary
	"context"
	"database/sql"
	_ "embed"

	eth "github.com/ethereum/go-ethereum/common"
	ac "github.com/status-im/status-go/services/wallet/activity/common"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

const NoLimitTimestampForPeriod = 0

type Period struct {
	StartTimestamp int64 `json:"startTimestamp"`
	EndTimestamp   int64 `json:"endTimestamp"`
}

func allActivityTypesFilter() []ac.Type {
	return []ac.Type{}
}

func allNetworksFilter() []common.ChainID {
	return []common.ChainID{}
}

func allActivityStatusesFilter() []ac.Status {
	return []ac.Status{}
}

func allTokensFilter() []ac.Token {
	return []ac.Token{}
}

type Filter struct {
	Period                Period        `json:"period"`
	Types                 []ac.Type     `json:"types"`
	Statuses              []ac.Status   `json:"statuses"`
	CounterpartyAddresses []eth.Address `json:"counterpartyAddresses"`

	// Tokens
	Assets                []ac.Token `json:"assets"`
	Collectibles          []ac.Token `json:"collectibles"`
	FilterOutAssets       bool       `json:"filterOutAssets"`
	FilterOutCollectibles bool       `json:"filterOutCollectibles"`
}

func (f *Filter) IsEmpty() bool {
	return f.Period.StartTimestamp == NoLimitTimestampForPeriod &&
		f.Period.EndTimestamp == NoLimitTimestampForPeriod &&
		len(f.Types) == 0 &&
		len(f.Statuses) == 0 &&
		len(f.CounterpartyAddresses) == 0 &&
		len(f.Assets) == 0 &&
		len(f.Collectibles) == 0 &&
		!f.FilterOutAssets &&
		!f.FilterOutCollectibles
}

// Kept for API compatibility, to be reimplemented
func GetRecipients(ctx context.Context, db *sql.DB, chainIDs []common.ChainID, addresses []eth.Address, offset int, limit int) (recipients []eth.Address, hasMore bool, err error) {
	return []eth.Address{}, false, nil
}

// Kept for API compatibility, to be reimplemented
func GetOldestTimestamp(ctx context.Context, db *sql.DB, addresses []eth.Address) (timestamp uint64, err error) {
	return 0, nil
}

// Kept for API compatibility, to be reimplemented
func GetActivityCollectibles(ctx context.Context, db *sql.DB, chainIDs []common.ChainID, owners []eth.Address, offset int, limit int) ([]thirdparty.CollectibleUniqueID, error) {
	return []thirdparty.CollectibleUniqueID{}, nil
}
