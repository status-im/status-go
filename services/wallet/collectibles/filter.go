package collectibles

import (
	"context"
	"database/sql"
	"errors"

	// used for embedding the sql query in the binary
	_ "embed"

	"github.com/ethereum/go-ethereum/common"

	"github.com/jmoiron/sqlx"

	"github.com/status-im/status-go/protocol/communities/token"
	wcommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/thirdparty"
)

func allCommunityIDsFilter() []string {
	return []string{}
}

func allCommunityPrivilegesLevelsFilter() []token.PrivilegesLevel {
	return []token.PrivilegesLevel{}
}

func allFilter() Filter {
	return Filter{
		CommunityIDs:              allCommunityIDsFilter(),
		CommunityPrivilegesLevels: allCommunityPrivilegesLevelsFilter(),
		FilterCommunity:           All,
	}
}

type FilterCommunityType int

const (
	All FilterCommunityType = iota
	OnlyNonCommunity
	OnlyCommunity
)

type Filter struct {
	CommunityIDs              []string                `json:"community_ids"`
	CommunityPrivilegesLevels []token.PrivilegesLevel `json:"community_privileges_levels"`

	FilterCommunity FilterCommunityType `json:"filter_community"`
}

//go:embed filter.sql
var queryString string

func filterOwnedCollectibles(ctx context.Context, db *sql.DB, chainIDs []wcommon.ChainID, addresses []common.Address, filter Filter, offset int, limit int) ([]thirdparty.CollectibleUniqueID, error) {
	if len(addresses) == 0 {
		return nil, errors.New("no addresses provided")
	}
	if len(chainIDs) == 0 {
		return nil, errors.New("no chainIDs provided")
	}

	filterCommunityTypeAll := filter.FilterCommunity == All
	filterCommunityTypeOnlyNonCommunity := filter.FilterCommunity == OnlyNonCommunity
	filterCommunityTypeOnlyCommunity := filter.FilterCommunity == OnlyCommunity
	communityIDFilterDisabled := len(filter.CommunityIDs) == 0
	if communityIDFilterDisabled {
		// IN clause doesn't work with empty array, so we need to provide a dummy value
		filter.CommunityIDs = []string{""}
	}
	communityPrivilegesLevelDisabled := len(filter.CommunityPrivilegesLevels) == 0
	if communityPrivilegesLevelDisabled {
		// IN clause doesn't work with empty array, so we need to provide a dummy value
		filter.CommunityPrivilegesLevels = []token.PrivilegesLevel{token.PrivilegesLevel(0)}
	}

	query, args, err := sqlx.In(queryString,
		filterCommunityTypeAll, filterCommunityTypeOnlyNonCommunity, filterCommunityTypeOnlyCommunity,
		communityIDFilterDisabled, communityPrivilegesLevelDisabled,
		chainIDs, addresses, filter.CommunityIDs, filter.CommunityPrivilegesLevels,
		limit, offset)
	if err != nil {
		return nil, err
	}

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return thirdparty.RowsToCollectibles(rows)
}
