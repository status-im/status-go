package activity

import (
	"context"
	"database/sql"
	"fmt"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/services/wallet/common"
)

const NoLimitTimestampForPeriod = 0

type Period struct {
	StartTimestamp int64 `json:"startTimestamp"`
	EndTimestamp   int64 `json:"endTimestamp"`
}

type Type int

const (
	SendAT Type = iota
	ReceiveAT
	BuyAT
	SwapAT
	BridgeAT
	ContractDeploymentAT
	MintAT
)

func allActivityTypesFilter() []Type {
	return []Type{}
}

type Status int

const (
	FailedAS    Status = iota // failed status or at least one failed transaction for multi-transactions
	PendingAS                 // in pending DB or at least one transaction in pending for multi-transactions
	CompleteAS                // success status
	FinalizedAS               // all multi-transactions have success status
)

func allActivityStatusesFilter() []Status {
	return []Status{}
}

type TokenType int

const (
	Native TokenType = iota
	Erc20
	Erc721
	Erc1155
)

type TokenID *hexutil.Big

// Token supports all tokens. Some fields might be optional, depending on the TokenType
type Token struct {
	TokenType TokenType `json:"tokenType"`
	// ChainID is used for TokenType.Native only to lookup the symbol, all chains will be included in the token filter
	ChainID common.ChainID `json:"chainId"`
	Address eth.Address    `json:"address,omitempty"`
	TokenID TokenID        `json:"tokenId,omitempty"`
}

func allTokensFilter() []Token {
	return []Token{}
}

func allAddressesFilter() []eth.Address {
	return []eth.Address{}
}

func allNetworksFilter() []common.ChainID {
	return []common.ChainID{}
}

type Filter struct {
	Period                Period        `json:"period"`
	Types                 []Type        `json:"types"`
	Statuses              []Status      `json:"statuses"`
	CounterpartyAddresses []eth.Address `json:"counterpartyAddresses"`

	// Tokens
	Assets                []Token `json:"assets"`
	Collectibles          []Token `json:"collectibles"`
	FilterOutAssets       bool    `json:"filterOutAssets"`
	FilterOutCollectibles bool    `json:"filterOutCollectibles"`
}

func GetRecipients(ctx context.Context, db *sql.DB, offset int, limit int) (addresses []eth.Address, hasMore bool, err error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			to_address,
			MIN(timestamp) AS min_timestamp
		FROM (
			SELECT
				transfers.tx_to_address as to_address,
				MIN(transfers.timestamp) AS timestamp
			FROM
				transfers
			WHERE
				transfers.multi_transaction_id = 0 AND transfers.tx_to_address NOT NULL
			GROUP BY
				transfers.tx_to_address

			UNION

			SELECT
				pending_transactions.to_address AS to_address,
				MIN(pending_transactions.timestamp) AS timestamp
			FROM
				pending_transactions
			WHERE
				pending_transactions.multi_transaction_id = 0 AND pending_transactions.to_address NOT NULL
			GROUP BY
				pending_transactions.to_address

			UNION

			SELECT
				multi_transactions.to_address AS to_address,
				MIN(multi_transactions.timestamp) AS timestamp
			FROM
				multi_transactions
			GROUP BY
				multi_transactions.to_address
		) AS combined_result
		GROUP BY
			to_address
		ORDER BY
			min_timestamp DESC
		LIMIT ? OFFSET ?;`, limit, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var entries []eth.Address
	for rows.Next() {
		var toAddress eth.Address
		var timestamp int64
		err := rows.Scan(&toAddress, &timestamp)
		if err != nil {
			return nil, false, err
		}
		entries = append(entries, toAddress)
	}

	if err = rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore = len(entries) == limit

	return entries, hasMore, nil
}

func GetOldestTimestamp(ctx context.Context, db *sql.DB, addresses []eth.Address) (timestamp int64, err error) {
	queryFormatString := `
		WITH filter_conditions AS (SELECT ? AS filterAllAddresses),
			filter_addresses(address) AS (
				SELECT * FROM (VALUES %s) WHERE (SELECT filterAllAddresses FROM filter_conditions) = 0
			)

		SELECT
			transfers.tx_from_address AS from_address,
			transfers.tx_to_address AS to_address,
			transfers.timestamp AS timestamp
		FROM transfers, filter_conditions
		WHERE transfers.multi_transaction_id = 0
			AND (filterAllAddresses OR HEX(from_address) IN filter_addresses OR HEX(to_address) IN filter_addresses)

		UNION ALL

		SELECT
			pending_transactions.from_address AS from_address,
			pending_transactions.to_address AS to_address,
			pending_transactions.timestamp AS timestamp
		FROM pending_transactions, filter_conditions
		WHERE pending_transactions.multi_transaction_id = 0
			AND (filterAllAddresses OR HEX(from_address) IN filter_addresses OR HEX(to_address) IN filter_addresses)

		UNION ALL

		SELECT
			multi_transactions.from_address AS from_address,
			multi_transactions.to_address AS to_address,
			multi_transactions.timestamp AS timestamp
		FROM multi_transactions, filter_conditions
		WHERE filterAllAddresses OR HEX(from_address) IN filter_addresses OR HEX(to_address) IN filter_addresses
		ORDER BY timestamp ASC
		LIMIT 1`

	filterAllAddresses := len(addresses) == 0
	involvedAddresses := noEntriesInTmpTableSQLValues
	if !filterAllAddresses {
		involvedAddresses = joinAddresses(addresses)
	}
	queryString := fmt.Sprintf(queryFormatString, involvedAddresses)

	row := db.QueryRowContext(ctx, queryString, filterAllAddresses)
	var fromAddress, toAddress sql.NullString
	err = row.Scan(&fromAddress, &toAddress, &timestamp)
	if err == sql.ErrNoRows {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	return timestamp, nil
}
