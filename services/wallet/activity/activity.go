package activity

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/multiaccounts/accounts"
	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/transactions"

	"golang.org/x/exp/constraints"
)

type PayloadType = int

// Beware: pleas update multiTransactionTypeToActivityType if changing this enum
const (
	MultiTransactionPT PayloadType = iota + 1
	SimpleTransactionPT
	PendingTransactionPT
)

const keypairAccountsTable = "keypairs_accounts"

var (
	ZeroAddress = eth.Address{}
)

type TransferType = int

const (
	TransferTypeEth TransferType = iota + 1
	TransferTypeErc20
	TransferTypeErc721
	TransferTypeErc1155
)

type Entry struct {
	payloadType    PayloadType
	transaction    *transfer.TransactionIdentity
	id             transfer.MultiTransactionIDType
	timestamp      int64
	activityType   Type
	activityStatus Status
	amountOut      *hexutil.Big // Used for activityType SendAT, SwapAT, BridgeAT
	amountIn       *hexutil.Big // Used for activityType ReceiveAT, BuyAT, SwapAT, BridgeAT
	tokenOut       *Token       // Used for activityType SendAT, SwapAT, BridgeAT
	tokenIn        *Token       // Used for activityType ReceiveAT, BuyAT, SwapAT, BridgeAT
	symbolOut      *string
	symbolIn       *string
	sender         *eth.Address
	recipient      *eth.Address
	chainIDOut     *common.ChainID
	chainIDIn      *common.ChainID
	transferType   *TransferType
}

type jsonSerializationTemplate struct {
	PayloadType    PayloadType                     `json:"payloadType"`
	Transaction    *transfer.TransactionIdentity   `json:"transaction"`
	ID             transfer.MultiTransactionIDType `json:"id"`
	Timestamp      int64                           `json:"timestamp"`
	ActivityType   Type                            `json:"activityType"`
	ActivityStatus Status                          `json:"activityStatus"`
	AmountOut      *hexutil.Big                    `json:"amountOut"`
	AmountIn       *hexutil.Big                    `json:"amountIn"`
	TokenOut       *Token                          `json:"tokenOut,omitempty"`
	TokenIn        *Token                          `json:"tokenIn,omitempty"`
	SymbolOut      *string                         `json:"symbolOut,omitempty"`
	SymbolIn       *string                         `json:"symbolIn,omitempty"`
	Sender         *eth.Address                    `json:"sender,omitempty"`
	Recipient      *eth.Address                    `json:"recipient,omitempty"`
	ChainIDOut     *common.ChainID                 `json:"chainIdOut,omitempty"`
	ChainIDIn      *common.ChainID                 `json:"chainIdIn,omitempty"`
	TransferType   *TransferType                   `json:"transferType,omitempty"`
}

func (e *Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonSerializationTemplate{
		PayloadType:    e.payloadType,
		Transaction:    e.transaction,
		ID:             e.id,
		Timestamp:      e.timestamp,
		ActivityType:   e.activityType,
		ActivityStatus: e.activityStatus,
		AmountOut:      e.amountOut,
		AmountIn:       e.amountIn,
		TokenOut:       e.tokenOut,
		TokenIn:        e.tokenIn,
		SymbolOut:      e.symbolOut,
		SymbolIn:       e.symbolIn,
		Sender:         e.sender,
		Recipient:      e.recipient,
		ChainIDOut:     e.chainIDOut,
		ChainIDIn:      e.chainIDIn,
		TransferType:   e.transferType,
	})
}

func (e *Entry) UnmarshalJSON(data []byte) error {
	aux := jsonSerializationTemplate{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	e.payloadType = aux.PayloadType
	e.transaction = aux.Transaction
	e.id = aux.ID
	e.timestamp = aux.Timestamp
	e.activityType = aux.ActivityType
	e.activityStatus = aux.ActivityStatus
	e.amountOut = aux.AmountOut
	e.amountIn = aux.AmountIn
	e.tokenOut = aux.TokenOut
	e.tokenIn = aux.TokenIn
	e.symbolOut = aux.SymbolOut
	e.symbolIn = aux.SymbolIn
	e.sender = aux.Sender
	e.recipient = aux.Recipient
	e.chainIDOut = aux.ChainIDOut
	e.chainIDIn = aux.ChainIDIn
	e.transferType = aux.TransferType
	return nil
}

func newActivityEntryWithPendingTransaction(transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status) Entry {
	return newActivityEntryWithTransaction(true, transaction, timestamp, activityType, activityStatus)
}

func newActivityEntryWithSimpleTransaction(transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status) Entry {
	return newActivityEntryWithTransaction(false, transaction, timestamp, activityType, activityStatus)
}

func newActivityEntryWithTransaction(pending bool, transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status) Entry {
	payloadType := SimpleTransactionPT
	if pending {
		payloadType = PendingTransactionPT
	}

	return Entry{
		payloadType:    payloadType,
		transaction:    transaction,
		id:             0,
		timestamp:      timestamp,
		activityType:   activityType,
		activityStatus: activityStatus,
	}
}

func NewActivityEntryWithMultiTransaction(id transfer.MultiTransactionIDType, timestamp int64, activityType Type, activityStatus Status) Entry {
	return Entry{
		payloadType:    MultiTransactionPT,
		id:             id,
		timestamp:      timestamp,
		activityType:   activityType,
		activityStatus: activityStatus,
	}
}

func (e *Entry) PayloadType() PayloadType {
	return e.payloadType
}

func multiTransactionTypeToActivityType(mtType transfer.MultiTransactionType) Type {
	if mtType == transfer.MultiTransactionSend {
		return SendAT
	} else if mtType == transfer.MultiTransactionSwap {
		return SwapAT
	} else if mtType == transfer.MultiTransactionBridge {
		return BridgeAT
	}
	panic("unknown multi transaction type")
}

func sliceContains[T constraints.Ordered](slice []T, item T) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func sliceChecksCondition[T any](slice []T, condition func(*T) bool) bool {
	for i := range slice {
		if condition(&slice[i]) {
			return true
		}
	}
	return false
}

func joinItems[T interface{}](items []T, itemConversion func(T) string) string {
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	if itemConversion == nil {
		itemConversion = func(item T) string {
			return fmt.Sprintf("%v", item)
		}
	}
	for i, item := range items {
		if i == 0 {
			sb.WriteString("(")
		} else {
			sb.WriteString("),(")
		}
		sb.WriteString(itemConversion(item))
	}
	sb.WriteString(")")

	return sb.String()
}

func joinAddresses(addresses []eth.Address) string {
	return joinItems(addresses, func(a eth.Address) string {
		return fmt.Sprintf("'%s'", strings.ToUpper(hex.EncodeToString(a[:])))
	})
}

func activityTypesToMultiTransactionTypes(trTypes []Type) []transfer.MultiTransactionType {
	mtTypes := make([]transfer.MultiTransactionType, 0, len(trTypes))
	for _, t := range trTypes {
		var mtType transfer.MultiTransactionType
		if t == SendAT {
			mtType = transfer.MultiTransactionSend
		} else if t == SwapAT {
			mtType = transfer.MultiTransactionSwap
		} else if t == BridgeAT {
			mtType = transfer.MultiTransactionBridge
		} else {
			continue
		}
		mtTypes = append(mtTypes, mtType)
	}
	return mtTypes
}

const (
	fromTrType = byte(1)
	toTrType   = byte(2)

	// TODO optimization: consider implementing nullable []byte instead of using strings for addresses
	//                      or insert binary (X'...' syntax) directly into the query
	//
	// Query includes duplicates, will return multiple rows for the same transaction if both to and from addresses
	// are in the address list.
	//
	// The addresses list will have priority in deciding the source of the duplicate transaction. However, if the
	// if the addresses list is empty, and all addresses should be included, the accounts table will be used
	// see filter_addresses temp table is used
	// The switch for tr_type is used to de-conflict the source for the two entries for the same transaction
	//
	// UNION ALL is used to avoid the overhead of DISTINCT given that we don't expect to have duplicate entries outside
	// the sender and receiver addresses being in the list which is handled separately
	//
	// Only status FailedAS, PendingAS and CompleteAS are returned. FinalizedAS requires correlation with blockchain
	// current state. As an optimization we can approximate it by using timestamp information or last known block number
	//
	// Token filtering has two parts
	// 1. Filtering by symbol (multi_transactions and pending_transactions tables) where the chain ID is ignored, basically the
	//    filter_networks will account for that
	// 2. Filtering by token identity (chain and address for transfers table) where the symbol is ignored and all the
	//      token identities must be provided
	queryFormatString = `
	WITH filter_conditions AS (
		SELECT
			? AS startFilterDisabled,
			? AS startTimestamp,
			? AS endFilterDisabled,
			? AS endTimestamp,

			? AS filterActivityTypeAll,
			? AS filterActivityTypeSend,
			? AS filterActivityTypeReceive,

			? AS fromTrType,
			? AS toTrType,

			? AS filterAllAddresses,
			? AS filterAllToAddresses,

			? AS filterAllActivityStatus,
			? AS filterStatusCompleted,
			? AS filterStatusFailed,
			? AS filterStatusFinalized,
			? AS filterStatusPending,

			? AS statusFailed,
			? AS statusSuccess,
			? AS statusPending,

			? AS includeAllTokenTypeAssets,

			? AS includeAllNetworks,

			? AS pendingStatus
		),
		filter_addresses(address) AS (
			SELECT HEX(address) FROM %s WHERE (SELECT filterAllAddresses FROM filter_conditions) != 0
			UNION ALL
			SELECT * FROM (VALUES %s) WHERE (SELECT filterAllAddresses FROM filter_conditions) = 0
		),
		filter_to_addresses(address) AS (
			VALUES %s
		),
		assets_token_codes(token_code) AS (
			VALUES %s
		),
		assets_erc20(chain_id, token_address) AS (
			VALUES %s
		),
		filter_networks(network_id) AS (
			VALUES %s
		),
		tr_status AS (
			SELECT
				multi_transaction_id,
				MIN(status) AS min_status,
				COUNT(*) AS count,
				network_id
			FROM
				transfers
			WHERE transfers.loaded == 1
				AND transfers.multi_transaction_id != 0
			GROUP BY transfers.multi_transaction_id
		),
		tr_network_ids AS (
			SELECT
				multi_transaction_id
			FROM
				transfers
			WHERE transfers.loaded == 1
				AND transfers.multi_transaction_id != 0
				AND network_id IN filter_networks
			GROUP BY transfers.multi_transaction_id
		),
		pending_status AS (
			SELECT
				multi_transaction_id,
				COUNT(*) AS count,
				network_id
			FROM
				pending_transactions, filter_conditions
			WHERE pending_transactions.multi_transaction_id != 0 AND pending_transactions.status = pendingStatus
			GROUP BY pending_transactions.multi_transaction_id
		),
		pending_network_ids AS (
			SELECT
				multi_transaction_id
			FROM
				pending_transactions, filter_conditions
			WHERE pending_transactions.multi_transaction_id != 0 AND pending_transactions.status = pendingStatus
				AND pending_transactions.network_id IN filter_networks
			GROUP BY pending_transactions.multi_transaction_id
		)
	SELECT
		transfers.hash AS transfer_hash,
		NULL AS pending_hash,
		transfers.network_id AS network_id,
		0 AS multi_tx_id,
		transfers.timestamp AS timestamp,
		NULL AS mt_type,

		CASE
			WHEN from_join.address IS NOT NULL AND to_join.address IS NULL THEN fromTrType
			WHEN to_join.address IS NOT NULL AND from_join.address IS NULL THEN toTrType
			WHEN from_join.address IS NOT NULL AND to_join.address IS NOT NULL THEN
				CASE
					WHEN from_join.address < to_join.address THEN fromTrType
					ELSE toTrType
				END
			ELSE NULL
		END as tr_type,

		transfers.tx_from_address AS from_address,
		transfers.tx_to_address AS to_address,
		transfers.address AS owner_address,
		transfers.amount_padded128hex AS tr_amount,
		NULL AS mt_from_amount,
		NULL AS mt_to_amount,

		CASE
			WHEN transfers.status IS 1 THEN statusSuccess
			ELSE statusFailed
		END AS agg_status,

		1 AS agg_count,

		transfers.token_address AS token_address,
		transfers.token_id AS token_id,
		NULL AS token_code,
		NULL AS from_token_code,
		NULL AS to_token_code,
		NULL AS out_network_id,
		NULL AS in_network_id,
		transfers.type AS type,
		transfers.contract_address AS contract_address
	FROM transfers, filter_conditions
	LEFT JOIN
		filter_addresses from_join ON HEX(transfers.tx_from_address) = from_join.address
	LEFT JOIN
		filter_addresses to_join ON HEX(transfers.tx_to_address) = to_join.address
	WHERE
		transfers.loaded == 1
		AND transfers.multi_transaction_id = 0
		AND ((startFilterDisabled OR transfers.timestamp >= startTimestamp)
			AND (endFilterDisabled OR transfers.timestamp <= endTimestamp)
		)
		AND (filterActivityTypeAll
			OR (filterActivityTypeSend
				AND (filterAllAddresses
					OR (HEX(transfers.tx_from_address) IN filter_addresses)
				)
			)
			OR (filterActivityTypeReceive
				AND (filterAllAddresses
					OR (HEX(transfers.tx_to_address) IN filter_addresses))
			)
		)
		AND (filterAllAddresses
			OR (HEX(transfers.tx_from_address) IN filter_addresses)
			OR (HEX(transfers.tx_to_address) IN filter_addresses)
		)
		AND (filterAllToAddresses
			OR (HEX(transfers.tx_to_address) IN filter_to_addresses)
		)
		AND (includeAllTokenTypeAssets OR (transfers.type = "eth" AND ("ETH" IN assets_token_codes))
			OR (transfers.type = "erc20" AND ((transfers.network_id, HEX(transfers.token_address)) IN assets_erc20))
		)
		AND (includeAllNetworks OR (transfers.network_id IN filter_networks))
		AND (filterAllActivityStatus OR ((filterStatusCompleted OR filterStatusFinalized) AND transfers.status = 1)
			OR (filterStatusFailed AND transfers.status = 0)
		)

	UNION ALL

	SELECT
		NULL AS transfer_hash,
		pending_transactions.hash AS pending_hash,
		pending_transactions.network_id AS network_id,
		0 AS multi_tx_id,
		pending_transactions.timestamp AS timestamp,
		NULL AS mt_type,

		CASE
			WHEN from_join.address IS NOT NULL AND to_join.address IS NULL THEN fromTrType
			WHEN to_join.address IS NOT NULL AND from_join.address IS NULL THEN toTrType
			WHEN from_join.address IS NOT NULL AND to_join.address IS NOT NULL THEN
				CASE
					WHEN from_join.address < to_join.address THEN fromTrType
					ELSE toTrType
				END
			ELSE NULL
		END as tr_type,

		pending_transactions.from_address AS from_address,
		pending_transactions.to_address AS to_address,
		NULL AS owner_address,
		pending_transactions.value AS tr_amount,
		NULL AS mt_from_amount,
		NULL AS mt_to_amount,

		statusPending AS agg_status,
		1 AS agg_count,

		NULL AS token_address,
		NULL AS token_id,
		pending_transactions.symbol AS token_code,
		NULL AS from_token_code,
		NULL AS to_token_code,
		NULL AS out_network_id,
		NULL AS in_network_id,
		pending_transactions.type AS type,
		NULL as contract_address
	FROM pending_transactions, filter_conditions
	LEFT JOIN
		filter_addresses from_join ON HEX(pending_transactions.from_address) = from_join.address
	LEFT JOIN
		filter_addresses to_join ON HEX(pending_transactions.to_address) = to_join.address
	WHERE pending_transactions.multi_transaction_id = 0 AND pending_transactions.status = pendingStatus
		AND (filterAllActivityStatus OR filterStatusPending)
		AND ((startFilterDisabled OR timestamp >= startTimestamp)
			AND (endFilterDisabled OR timestamp <= endTimestamp)
		)
		AND (filterActivityTypeAll OR filterActivityTypeSend)
		AND (filterAllAddresses
			OR (HEX(pending_transactions.from_address) IN filter_addresses)
			OR (HEX(pending_transactions.to_address) IN filter_addresses)
		)
		AND (filterAllToAddresses
			OR (HEX(pending_transactions.to_address) IN filter_to_addresses)
		)
		AND (includeAllTokenTypeAssets OR (UPPER(pending_transactions.symbol) IN assets_token_codes))
		AND (includeAllNetworks OR (pending_transactions.network_id IN filter_networks))

	UNION ALL

	SELECT
		NULL AS transfer_hash,
		NULL AS pending_hash,
		NULL AS network_id,
		multi_transactions.ROWID AS multi_tx_id,
		multi_transactions.timestamp AS timestamp,
		multi_transactions.type AS mt_type,
		NULL as tr_type,
		multi_transactions.from_address AS from_address,
		multi_transactions.to_address AS to_address,
		NULL AS owner_address,
		NULL AS tr_amount,
		multi_transactions.from_amount AS mt_from_amount,
		multi_transactions.to_amount AS mt_to_amount,

		CASE
			WHEN tr_status.min_status = 1 AND COALESCE(pending_status.count, 0) = 0 THEN statusSuccess
			WHEN tr_status.min_status = 0 THEN statusFailed
			ELSE statusPending
		END AS agg_status,

		COALESCE(tr_status.count, 0) + COALESCE(pending_status.count, 0) AS agg_count,

		NULL AS token_address,
		NULL AS token_id,
		NULL AS token_code,
		multi_transactions.from_asset AS from_token_code,
		multi_transactions.to_asset AS to_token_code,
		multi_transactions.from_network_id AS out_network_id,
		multi_transactions.to_network_id AS in_network_id,
		NULL AS type,
		NULL as contract_address
	FROM multi_transactions, filter_conditions
	LEFT JOIN tr_status ON multi_transactions.ROWID = tr_status.multi_transaction_id
	LEFT JOIN pending_status ON multi_transactions.ROWID = pending_status.multi_transaction_id
	WHERE
		((startFilterDisabled OR multi_transactions.timestamp >= startTimestamp)
			AND (endFilterDisabled OR multi_transactions.timestamp <= endTimestamp)
		)
		AND (filterActivityTypeAll OR (multi_transactions.type IN (%s)))
		AND (filterAllAddresses
			OR (HEX(multi_transactions.from_address) IN filter_addresses)
			OR (HEX(multi_transactions.to_address) IN filter_addresses)
		)
		AND (filterAllToAddresses
			OR (HEX(multi_transactions.to_address) IN filter_to_addresses)
		)
		AND (includeAllTokenTypeAssets
			OR (multi_transactions.from_asset != '' AND (UPPER(multi_transactions.from_asset) IN assets_token_codes))
			OR (multi_transactions.to_asset != '' AND (UPPER(multi_transactions.to_asset) IN assets_token_codes))
		)
		AND (filterAllActivityStatus
			OR ((filterStatusCompleted OR filterStatusFinalized) AND agg_status = statusSuccess)
			OR (filterStatusFailed AND agg_status = statusFailed) OR (filterStatusPending AND agg_status = statusPending)
		)
		AND (includeAllNetworks
			OR (multi_transactions.from_network_id IN filter_networks)
			OR (multi_transactions.to_network_id IN filter_networks)
			OR (multi_transactions.from_network_id IS NULL
				AND multi_transactions.to_network_id IS NULL
				AND (EXISTS (SELECT 1 FROM tr_network_ids WHERE multi_transactions.ROWID = tr_network_ids.multi_transaction_id)
					OR EXISTS (SELECT 1 FROM pending_network_ids WHERE multi_transactions.ROWID = pending_network_ids.multi_transaction_id)
				)
			)
		)

	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?`

	noEntriesInTmpTableSQLValues           = "(NULL)"
	noEntriesInTwoColumnsTmpTableSQLValues = "(NULL, NULL)"
)

type FilterDependencies struct {
	db         *sql.DB
	accountsDb *accounts.Database
	// use token.TokenType, token.ChainID and token.Address to find the available symbol
	tokenSymbol func(token Token) string
	// use the chainID and symbol to look up token.TokenType and token.Address. Return nil if not found
	tokenFromSymbol func(chainID *common.ChainID, symbol string) *Token
}

// getActivityEntries queries the transfers, pending_transactions, and multi_transactions tables
// based on filter parameters and arguments
// it returns metadata for all entries ordered by timestamp column
//
// Adding a no-limit option was never considered or required.
func getActivityEntries(ctx context.Context, deps FilterDependencies, addresses []eth.Address, chainIDs []common.ChainID, filter Filter, offset int, limit int) ([]Entry, error) {
	includeAllTokenTypeAssets := len(filter.Assets) == 0 && !filter.FilterOutAssets

	// Used for symbol bearing tables multi_transactions and pending_transactions
	assetsTokenCodes := noEntriesInTmpTableSQLValues
	// Used for identity bearing tables transfers
	assetsERC20 := noEntriesInTwoColumnsTmpTableSQLValues
	if !includeAllTokenTypeAssets && !filter.FilterOutAssets {
		symbolsSet := make(map[string]struct{})
		var symbols []string
		for _, item := range filter.Assets {
			symbol := deps.tokenSymbol(item)
			if _, ok := symbolsSet[symbol]; !ok {
				symbols = append(symbols, symbol)
				symbolsSet[symbol] = struct{}{}
			}
		}
		assetsTokenCodes = joinItems(symbols, func(s string) string {
			return fmt.Sprintf("'%s'", s)
		})

		if sliceChecksCondition(filter.Assets, func(item *Token) bool { return item.TokenType == Erc20 }) {
			assetsERC20 = joinItems(filter.Assets, func(item Token) string {
				if item.TokenType == Erc20 {
					// SQL HEX() (Blob->Hex) conversion returns uppercase digits with no 0x prefix
					return fmt.Sprintf("%d, '%s'", item.ChainID, strings.ToUpper(item.Address.Hex()[2:]))
				}
				return ""
			})
		}
	}

	// construct chain IDs
	includeAllNetworks := len(chainIDs) == 0
	networks := noEntriesInTmpTableSQLValues
	if !includeAllNetworks {
		networks = joinItems(chainIDs, nil)
	}

	startFilterDisabled := !(filter.Period.StartTimestamp > 0)
	endFilterDisabled := !(filter.Period.EndTimestamp > 0)
	filterActivityTypeAll := len(filter.Types) == 0
	filterAllAddresses := len(addresses) == 0
	filterAllToAddresses := len(filter.CounterpartyAddresses) == 0
	includeAllStatuses := len(filter.Statuses) == 0

	filterStatusPending := false
	filterStatusCompleted := false
	filterStatusFailed := false
	filterStatusFinalized := false
	if !includeAllStatuses {
		filterStatusPending = sliceContains(filter.Statuses, PendingAS)
		filterStatusCompleted = sliceContains(filter.Statuses, CompleteAS)
		filterStatusFailed = sliceContains(filter.Statuses, FailedAS)
		filterStatusFinalized = sliceContains(filter.Statuses, FinalizedAS)
	}

	involvedAddresses := noEntriesInTmpTableSQLValues
	if !filterAllAddresses {
		involvedAddresses = joinAddresses(addresses)
	}
	toAddresses := noEntriesInTmpTableSQLValues
	if !filterAllToAddresses {
		toAddresses = joinAddresses(filter.CounterpartyAddresses)
	}

	mtTypes := activityTypesToMultiTransactionTypes(filter.Types)
	joinedMTTypes := joinItems(mtTypes, func(t transfer.MultiTransactionType) string {
		return strconv.Itoa(int(t))
	})

	// Since the filter query needs addresses which are in a different database, we need to update the
	// keypairs_accounts table in the current database with the latest addresses from the accounts database
	err := updateKeypairsAccountsTable(deps.accountsDb, deps.db)
	if err != nil {
		return nil, err
	}

	queryString := fmt.Sprintf(queryFormatString, keypairAccountsTable, involvedAddresses, toAddresses, assetsTokenCodes, assetsERC20, networks,
		joinedMTTypes)

	rows, err := deps.db.QueryContext(ctx, queryString,
		startFilterDisabled, filter.Period.StartTimestamp, endFilterDisabled, filter.Period.EndTimestamp,
		filterActivityTypeAll, sliceContains(filter.Types, SendAT), sliceContains(filter.Types, ReceiveAT),
		fromTrType, toTrType,
		filterAllAddresses, filterAllToAddresses,
		includeAllStatuses, filterStatusCompleted, filterStatusFailed, filterStatusFinalized, filterStatusPending,
		FailedAS, CompleteAS, PendingAS,
		includeAllTokenTypeAssets,
		includeAllNetworks,
		transactions.Pending,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var transferHash, pendingHash []byte
		var chainID, outChainIDDB, inChainIDDB, multiTxID, aggregatedCount sql.NullInt64
		var timestamp int64
		var dbMtType, dbTrType sql.NullByte
		var toAddress, fromAddress eth.Address
		var toAddressDB, ownerAddressDB, contractAddressDB, dbTokenID sql.RawBytes
		var tokenAddress, contractAddress *eth.Address
		var aggregatedStatus int
		var dbTrAmount sql.NullString
		var dbMtFromAmount, dbMtToAmount, contractType sql.NullString
		var tokenCode, fromTokenCode, toTokenCode sql.NullString
		var transferType *TransferType
		err := rows.Scan(&transferHash, &pendingHash, &chainID, &multiTxID, &timestamp, &dbMtType, &dbTrType, &fromAddress,
			&toAddressDB, &ownerAddressDB, &dbTrAmount, &dbMtFromAmount, &dbMtToAmount, &aggregatedStatus, &aggregatedCount,
			&tokenAddress, &dbTokenID, &tokenCode, &fromTokenCode, &toTokenCode, &outChainIDDB, &inChainIDDB, &contractType, &contractAddressDB)
		if err != nil {
			return nil, err
		}

		if len(toAddressDB) > 0 {
			toAddress = eth.BytesToAddress(toAddressDB)
		}

		if contractType.Valid {
			transferType = contractTypeFromDBType(contractType.String)
		}
		if len(contractAddressDB) > 0 {
			contractAddress = new(eth.Address)
			*contractAddress = eth.BytesToAddress(contractAddressDB)
		}

		getActivityType := func(trType sql.NullByte) (activityType Type, filteredAddress eth.Address) {
			if trType.Valid {
				if trType.Byte == fromTrType {
					if toAddress == ZeroAddress && transferType != nil && *transferType == TransferTypeEth && contractAddress != nil && *contractAddress != ZeroAddress {
						return ContractDeploymentAT, fromAddress
					}
					return SendAT, fromAddress
				} else if trType.Byte == toTrType {
					if fromAddress == ZeroAddress && transferType != nil && *transferType == TransferTypeErc721 {
						return MintAT, toAddress
					}
					return ReceiveAT, toAddress
				}
			}
			log.Warn(fmt.Sprintf("unexpected activity type. Missing from [%s] or to [%s] in addresses?", fromAddress, toAddress))
			return ReceiveAT, toAddress
		}

		// Can be mapped directly because the values are injected into the query
		activityStatus := Status(aggregatedStatus)
		var outChainID, inChainID *common.ChainID
		var entry Entry
		var tokenID TokenID
		if len(dbTokenID) > 0 {
			t := new(big.Int).SetBytes(dbTokenID)
			tokenID = (*hexutil.Big)(t)
		}

		if transferHash != nil && chainID.Valid {
			// Extract activity type: SendAT/ReceiveAT
			activityType, _ := getActivityType(dbTrType)

			ownerAddress := eth.BytesToAddress(ownerAddressDB)
			inAmount, outAmount := getTrInAndOutAmounts(activityType, dbTrAmount)

			// Extract tokens and chains
			var involvedToken *Token
			if tokenAddress != nil && *tokenAddress != ZeroAddress {
				involvedToken = &Token{TokenType: Erc20, ChainID: common.ChainID(chainID.Int64), TokenID: tokenID, Address: *tokenAddress}
			} else {
				involvedToken = &Token{TokenType: Native, ChainID: common.ChainID(chainID.Int64), TokenID: tokenID}
			}

			entry = newActivityEntryWithSimpleTransaction(
				&transfer.TransactionIdentity{ChainID: common.ChainID(chainID.Int64),
					Hash:    eth.BytesToHash(transferHash),
					Address: ownerAddress,
				},
				timestamp, activityType, activityStatus,
			)

			// Extract tokens
			if activityType == SendAT {
				entry.tokenOut = involvedToken
				outChainID = new(common.ChainID)
				*outChainID = common.ChainID(chainID.Int64)
			} else {
				entry.tokenIn = involvedToken
				inChainID = new(common.ChainID)
				*inChainID = common.ChainID(chainID.Int64)
			}

			entry.symbolOut, entry.symbolIn = lookupAndFillInTokens(deps, entry.tokenOut, entry.tokenIn)

			// Complete the data
			entry.amountOut = outAmount
			entry.amountIn = inAmount
		} else if pendingHash != nil && chainID.Valid {
			// Extract activity type: SendAT/ReceiveAT
			activityType, _ := getActivityType(dbTrType)

			inAmount, outAmount := getTrInAndOutAmounts(activityType, dbTrAmount)

			outChainID = new(common.ChainID)
			*outChainID = common.ChainID(chainID.Int64)

			entry = newActivityEntryWithPendingTransaction(
				&transfer.TransactionIdentity{ChainID: common.ChainID(chainID.Int64),
					Hash: eth.BytesToHash(pendingHash),
				},
				timestamp, activityType, activityStatus,
			)

			// Extract tokens
			if tokenCode.Valid {
				cID := common.ChainID(chainID.Int64)
				entry.tokenOut = deps.tokenFromSymbol(&cID, tokenCode.String)
			}
			entry.symbolOut, entry.symbolIn = lookupAndFillInTokens(deps, entry.tokenOut, nil)

			// Complete the data
			entry.amountOut = outAmount
			entry.amountIn = inAmount

		} else if multiTxID.Valid {
			mtInAmount, mtOutAmount := getMtInAndOutAmounts(dbMtFromAmount, dbMtToAmount)

			// Extract activity type: SendAT/SwapAT/BridgeAT
			activityType := multiTransactionTypeToActivityType(transfer.MultiTransactionType(dbMtType.Byte))

			if outChainIDDB.Valid && outChainIDDB.Int64 != 0 {
				outChainID = new(common.ChainID)
				*outChainID = common.ChainID(outChainIDDB.Int64)
			}
			if inChainIDDB.Valid && inChainIDDB.Int64 != 0 {
				inChainID = new(common.ChainID)
				*inChainID = common.ChainID(inChainIDDB.Int64)
			}

			entry = NewActivityEntryWithMultiTransaction(transfer.MultiTransactionIDType(multiTxID.Int64),
				timestamp, activityType, activityStatus)

			// Extract tokens
			if fromTokenCode.Valid {
				entry.tokenOut = deps.tokenFromSymbol(outChainID, fromTokenCode.String)
				entry.symbolOut = common.NewAndSet(fromTokenCode.String)
			}
			if toTokenCode.Valid {
				entry.tokenIn = deps.tokenFromSymbol(inChainID, toTokenCode.String)
				entry.symbolIn = common.NewAndSet(toTokenCode.String)
			}

			// Complete the data
			entry.amountOut = mtOutAmount
			entry.amountIn = mtInAmount
		} else {
			return nil, errors.New("invalid row data")
		}

		// Complete common data
		entry.sender = &fromAddress
		entry.recipient = &toAddress
		entry.sender = &fromAddress
		entry.recipient = &toAddress
		entry.chainIDOut = outChainID
		entry.chainIDIn = inChainID
		entry.transferType = transferType

		entries = append(entries, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func getTrInAndOutAmounts(activityType Type, trAmount sql.NullString) (inAmount *hexutil.Big, outAmount *hexutil.Big) {
	if trAmount.Valid {
		amount, ok := new(big.Int).SetString(trAmount.String, 16)
		if ok {
			switch activityType {
			case SendAT:
				inAmount = (*hexutil.Big)(big.NewInt(0))
				outAmount = (*hexutil.Big)(amount)
				return
			case ReceiveAT:
				inAmount = (*hexutil.Big)(amount)
				outAmount = (*hexutil.Big)(big.NewInt(0))
				return
			default:
				log.Warn(fmt.Sprintf("unexpected activity type %d", activityType))
			}
		} else {
			log.Warn(fmt.Sprintf("could not parse amount %s", trAmount.String))
		}
	} else {
		log.Warn(fmt.Sprintf("invalid transaction amount for type %d", activityType))
	}
	inAmount = (*hexutil.Big)(big.NewInt(0))
	outAmount = (*hexutil.Big)(big.NewInt(0))
	return
}

func getMtInAndOutAmounts(dbFromAmount sql.NullString, dbToAmount sql.NullString) (inAmount *hexutil.Big, outAmount *hexutil.Big) {
	if dbFromAmount.Valid && dbToAmount.Valid {
		fromHexStr := dbFromAmount.String
		toHexStr := dbToAmount.String
		if len(fromHexStr) > 2 && len(toHexStr) > 2 {
			fromAmount, frOk := new(big.Int).SetString(dbFromAmount.String[2:], 16)
			toAmount, toOk := new(big.Int).SetString(dbToAmount.String[2:], 16)
			if frOk && toOk {
				inAmount = (*hexutil.Big)(toAmount)
				outAmount = (*hexutil.Big)(fromAmount)
				return
			}
		}
		log.Warn(fmt.Sprintf("could not parse amounts %s %s", fromHexStr, toHexStr))
	} else {
		log.Warn("invalid transaction amounts")
	}
	inAmount = (*hexutil.Big)(big.NewInt(0))
	outAmount = (*hexutil.Big)(big.NewInt(0))
	return
}

func contractTypeFromDBType(dbType string) (transferType *TransferType) {
	transferType = new(TransferType)
	switch common.Type(dbType) {
	case common.EthTransfer:
		*transferType = TransferTypeEth
	case common.Erc20Transfer:
		*transferType = TransferTypeErc20
	case common.Erc721Transfer:
		*transferType = TransferTypeErc721
	default:
		return nil
	}
	return transferType
}

func updateKeypairsAccountsTable(accountsDb *accounts.Database, db *sql.DB) error {
	_, err := db.Exec(fmt.Sprintf("CREATE TEMP TABLE IF NOT EXISTS %s (address VARCHAR PRIMARY KEY)",
		keypairAccountsTable))
	if err != nil {
		log.Error("failed to create 'keypairs_accounts' table", "err", err)
		return err
	}

	addresses, err := accountsDb.GetWalletAddresses()
	if err != nil {
		log.Error("failed to get wallet addresses", "err", err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	for _, address := range addresses {
		_, err = tx.Exec(fmt.Sprintf("INSERT OR IGNORE INTO %s (address) VALUES (?)", keypairAccountsTable), address)
		if err != nil {
			log.Error("failed to insert wallet addresses", "err", err)
			return err
		}
	}

	return nil
}

func lookupAndFillInTokens(deps FilterDependencies, tokenOut *Token, tokenIn *Token) (symbolOut *string, symbolIn *string) {
	if tokenOut != nil {
		symbol := deps.tokenSymbol(*tokenOut)
		if len(symbol) > 0 {
			symbolOut = common.NewAndSet(symbol)
		}
	}
	if tokenIn != nil {
		symbol := deps.tokenSymbol(*tokenIn)
		if len(symbol) > 0 {
			symbolIn = common.NewAndSet(symbol)
		}
	}
	return symbolOut, symbolIn
}
