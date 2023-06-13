package activity

import (
	"context"
	"database/sql"

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

// TODO: consider sorting by saved address and contacts to offload the client from doing it at runtime
func GetRecipients(ctx context.Context, db *sql.DB, offset int, limit int) (addresses []eth.Address, hasMore bool, err error) {
	rows, err := db.QueryContext(ctx, `
		SELECT
			transfers.address as to_address,
			transfers.timestamp AS timestamp
		FROM transfers
		WHERE transfers.multi_transaction_id = 0

		UNION ALL

		SELECT
			pending_transactions.to_address AS to_address,
			pending_transactions.timestamp AS timestamp
		FROM pending_transactions
		WHERE pending_transactions.multi_transaction_id = 0

		UNION ALL

		SELECT
			multi_transactions.to_address AS to_address,
			multi_transactions.timestamp AS timestamp
		FROM multi_transactions
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?`, limit, offset)
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
