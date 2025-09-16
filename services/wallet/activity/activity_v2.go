package activity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	eth "github.com/ethereum/go-ethereum/common"

	ac "github.com/status-im/status-go/services/wallet/activity/common"
	wCommon "github.com/status-im/status-go/services/wallet/common"
)

type FilterDependencies struct {
	db *sql.DB
	// use token.TokenType, token.ChainID and token.Address to find the available symbol
	tokenSymbol func(token ac.Token) string
	// use the chainID and symbol to look up token.TokenType and token.Address. Return nil if not found
	tokenFromSymbol func(chainID *wCommon.ChainID, symbol string) *ac.Token
	// use to get current timestamp
	currentTimestamp func() int64
}

type TransactionOrigin int

const (
	SentTransaction TransactionOrigin = iota
	FetchedTransaction
)

func (s TransactionOrigin) String() string {
	switch s {
	case SentTransaction:
		return "sent"
	case FetchedTransaction:
		return "fetched"
	default:
		return "unknown"
	}
}

type OrderedTransactionID struct {
	ac.TransactionIdentity
	Timestamp int64
	Source    TransactionOrigin
}

// getActivityEntries is the main entry point for the implementation
// using SQL-based unified ordering approach. It retrieves activity entries
// for given addresses and chain IDs with pagination support.
func (s *Service) getActivityEntries(ctx context.Context, f fullFilterParams, offset int, limit int) ([]Entry, error) {
	if f.version != V2 {
		return nil, fmt.Errorf("unsupported version: %s, expected %s", f.version, V2)
	}

	txIDs, _, err := getTransactionsOrder(ctx, s.getDeps().db, f.addresses, f.chainIDs, f.filter, offset, limit)
	if err != nil {
		return nil, err
	}

	if len(txIDs) == 0 {
		return []Entry{}, nil
	}

	entries, err := getEntriesByTransactionIDs(ctx, s.getDeps(), txIDs)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// getTransactionsOrder returns deduplicated, ordered transaction IDs from both sent and fetched sources.
// It returns the ordered list of transaction identities and total count for pagination.
func getTransactionsOrder(ctx context.Context, db *sql.DB, addresses []eth.Address, chainIDs []wCommon.ChainID, filter Filter, offset int, limit int) ([]OrderedTransactionID, int64, error) {
	if len(addresses) == 0 {
		return nil, 0, ErrNoAddressesProvided
	}
	if len(chainIDs) == 0 {
		return nil, 0, ErrNoChainIDsProvided
	}

	query := buildTransactionOrderQuery(len(chainIDs), len(addresses), limit != ac.NoLimit)
	args := buildQueryArgs(chainIDs, addresses, limit, offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query transactions order: %w", err)
	}
	defer rows.Close()

	return extractTransactionIdentities(rows)
}

// extractTransactionIdentities processes SQL result rows into TransactionID slice.
func extractTransactionIdentities(rows *sql.Rows) ([]OrderedTransactionID, int64, error) {
	var results []OrderedTransactionID
	var totalCount int64

	for rows.Next() {
		var txID OrderedTransactionID
		var chainID uint64
		var hashBytes []byte
		var addressBytes []byte

		err := rows.Scan(&chainID, &hashBytes, &txID.Timestamp, &txID.Source, &addressBytes, &totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan transaction row: %w", err)
		}

		txID.ChainID = wCommon.ChainID(chainID)
		txID.Hash = eth.BytesToHash(hashBytes)
		txID.Address = eth.BytesToAddress(addressBytes)
		results = append(results, txID)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating transaction rows: %w", err)
	}

	return results, totalCount, nil
}

// buildPlaceholders creates SQL placeholders for IN clauses
func buildPlaceholders(count int) string {
	if count == 0 {
		return ""
	}
	result := strings.Repeat("?,", count)
	return result[:len(result)-1] // Remove the trailing comma
}

// buildTransactionOrderQuery constructs the query for getting transaction order.
// This query merges sent and fetched transactions, deduplicates by preferring sent sources,
// and applies ordering
// String building used instead of Squirrel for this complex query because it is easier to comprehend this way
func buildTransactionOrderQuery(chainIDCount, addressCount int, withPagination bool) string {
	query := `
    SELECT 
        chain_id, 
        tx_hash, 
        timestamp, 
        source, 
        address,
        COUNT(*) OVER () AS total_count
    FROM (
        SELECT
            chain_id,
            tx_hash,
            timestamp,
            source,
            address,
            ROW_NUMBER() OVER (
                PARTITION BY chain_id, tx_hash, address
                ORDER BY source, timestamp DESC
            ) AS rn
        FROM (
            SELECT
                st.chain_id,
                st.tx_hash,
                tt.timestamp,
                0 AS source,
                rip.from_address AS address
            FROM sent_transactions st
            JOIN tracked_transactions tt ON st.chain_id = tt.chain_id AND st.tx_hash = tt.tx_hash
            JOIN route_path_transactions rpt ON st.chain_id = rpt.chain_id AND st.tx_hash = rpt.tx_hash
            JOIN route_input_parameters rip ON rpt.uuid = rip.uuid
            WHERE st.chain_id IN (` + buildPlaceholders(chainIDCount) + `)
                AND rip.from_address IN (` + buildPlaceholders(addressCount) + `)

            UNION ALL

            SELECT
                fat.chain_id,
                CASE 
                    WHEN substr(fat.hash, 1, 2) = '0x' THEN unhex(substr(fat.hash, 3))
                    ELSE unhex(fat.hash)
                END AS tx_hash,
                CAST(strftime('%s', json_extract(fat.transfer, '$.metadata.blockTimestamp')) AS INTEGER) AS timestamp,
                1 AS source,
                fat.address AS address
            FROM fetched_alchemy_transfers fat
            WHERE fat.chain_id IN (` + buildPlaceholders(chainIDCount) + `)
                AND fat.address IN (` + buildPlaceholders(addressCount) + `)
        )
    ) 
    WHERE rn = 1
    ORDER BY timestamp DESC, chain_id, tx_hash, address`

	if withPagination {
		query += `
    LIMIT ? OFFSET ?`
	}

	return query
}

// buildQueryArgs constructs the arguments for the getTransactionsOrder query.
func buildQueryArgs(chainIDs []wCommon.ChainID, addresses []eth.Address, limit int, offset int) []interface{} {
	capacity := 2 * (len(chainIDs) + len(addresses))
	if limit != ac.NoLimit {
		capacity += 2
	}
	args := make([]interface{}, 0, capacity)

	for _, chainID := range chainIDs {
		args = append(args, uint64(chainID))
	}
	for _, addr := range addresses {
		args = append(args, addr.Bytes())
	}
	for _, chainID := range chainIDs {
		args = append(args, uint64(chainID))
	}
	for _, addr := range addresses {
		args = append(args, addr.Bytes())
	}
	if limit > 0 {
		args = append(args, limit, offset)
	}

	return args
}

// getEntriesByTransactionIDs fetches full entry details for given transaction IDs
// maintaining the order from the input slice
func getEntriesByTransactionIDs(ctx context.Context, deps FilterDependencies, txIDs []OrderedTransactionID) ([]Entry, error) {
	sentEntries, err := getSentEntriesByIDs(ctx, deps, txIDs)
	if err != nil {
		return nil, err
	}

	fetchedEntries, err := getFetchedEntriesByIDs(ctx, deps, txIDs)
	if err != nil {
		return nil, err
	}

	entries := mergeAndOrderEntries(sentEntries, fetchedEntries, txIDs)

	return entries, nil
}

// mergeAndOrderEntries combines entries from sent and fetched sources
func mergeAndOrderEntries(sentEntries, fetchedEntries map[string]Entry, orderedIDs []OrderedTransactionID) []Entry {
	entries := make([]Entry, 0, len(orderedIDs))

	for _, txID := range orderedIDs {
		key := txID.Key()

		if entry, exists := sentEntries[key]; exists {
			entries = append(entries, entry)
		} else if entry, exists := fetchedEntries[key]; exists {
			entries = append(entries, entry)
		}
	}

	return entries
}

func getFinalizationPeriod(chainID wCommon.ChainID) int64 {
	switch uint64(chainID) {
	case wCommon.EthereumMainnet, wCommon.EthereumSepolia:
		return ac.L1FinalizationDuration
	case wCommon.BSCMainnet, wCommon.BSCTestnet:
		return ac.BSCFinalizationDuration
	}

	return ac.L2FinalizationDuration
}

// lookupAndFillInTokens ignores NFTs
func lookupAndFillInTokens(deps FilterDependencies, tokenOut *ac.Token, tokenIn *ac.Token) (symbolOut *string, symbolIn *string) {
	if tokenOut != nil && tokenOut.TokenID == nil {
		symbol := deps.tokenSymbol(*tokenOut)
		if len(symbol) > 0 {
			symbolOut = wCommon.NewAndSet(symbol)
		}
	}
	if tokenIn != nil && tokenIn.TokenID == nil {
		symbol := deps.tokenSymbol(*tokenIn)
		if len(symbol) > 0 {
			symbolIn = wCommon.NewAndSet(symbol)
		}
	}
	return symbolOut, symbolIn
}
