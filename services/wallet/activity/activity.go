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

	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/transfer"

	"golang.org/x/exp/constraints"
)

type PayloadType = int

// Beware: pleas update multiTransactionTypeToActivityType if changing this enum
const (
	MultiTransactionPT PayloadType = iota + 1
	SimpleTransactionPT
	PendingTransactionPT
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
	e.amountOut = aux.AmountOut
	e.amountIn = aux.AmountIn
	return nil
}

func newActivityEntryWithPendingTransaction(transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status, amountIn *hexutil.Big, amountOut *hexutil.Big, tokenOut *Token, tokenIn *Token) Entry {
	return newActivityEntryWithTransaction(true, transaction, timestamp, activityType, activityStatus, amountIn, amountOut, tokenOut, tokenIn)
}

func newActivityEntryWithSimpleTransaction(transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status, amountIn *hexutil.Big, amountOut *hexutil.Big, tokenOut *Token, tokenIn *Token) Entry {
	return newActivityEntryWithTransaction(false, transaction, timestamp, activityType, activityStatus, amountIn, amountOut, tokenOut, tokenIn)
}

func newActivityEntryWithTransaction(pending bool, transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status, amountIn *hexutil.Big, amountOut *hexutil.Big, tokenOut *Token, tokenIn *Token) Entry {
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
		amountIn:       amountIn,
		amountOut:      amountOut,
		tokenOut:       tokenOut,
		tokenIn:        tokenIn,
	}
}

func NewActivityEntryWithMultiTransaction(id transfer.MultiTransactionIDType, timestamp int64, activityType Type, activityStatus Status, amountIn *hexutil.Big, amountOut *hexutil.Big, tokenOut *Token, tokenIn *Token) Entry {
	return Entry{
		payloadType:    MultiTransactionPT,
		id:             id,
		timestamp:      timestamp,
		activityType:   activityType,
		activityStatus: activityStatus,
		amountIn:       amountIn,
		amountOut:      amountOut,
		tokenOut:       tokenOut,
		tokenIn:        tokenIn,
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

	// TODO: Multi-transaction network information is missing in filtering
	// TODO: extract token code for non transfer type eth
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

            ? AS includeAllNetworks
        ),
        filter_addresses(address) AS (
            SELECT HEX(address) FROM keypairs_accounts WHERE (SELECT filterAllAddresses FROM filter_conditions) != 0
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
              COUNT(*) AS count
            FROM
              transfers
            WHERE transfers.multi_transaction_id != 0
            GROUP BY
                transfers.multi_transaction_id
        ),
        pending_status AS (
            SELECT
              multi_transaction_id,
              COUNT(*) AS count
            FROM
              pending_transactions
            WHERE pending_transactions.multi_transaction_id != 0
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
                    WHEN from_join.address < to_join.address THEN 1
                    ELSE 2
                END
            ELSE NULL
        END as tr_type,

        transfers.sender AS from_address,
        transfers.address AS to_address,
        transfers.amount_padded128hex AS tr_amount,
        NULL AS mt_from_amount,
        NULL AS mt_to_amount,

        CASE
            WHEN transfers.status IS 1 THEN statusSuccess
            ELSE statusFailed
        END AS agg_status,

        1 AS agg_count,

        transfers.token_address AS token_address,
        NULL AS token_code,
        NULL AS from_token_code,
        NULL AS to_token_code
    FROM transfers, filter_conditions
    LEFT JOIN
        filter_addresses from_join ON HEX(transfers.sender) = from_join.address
    LEFT JOIN
        filter_addresses to_join ON HEX(transfers.address) = to_join.address
    WHERE transfers.multi_transaction_id = 0
        AND ((startFilterDisabled OR timestamp >= startTimestamp)
            AND (endFilterDisabled OR timestamp <= endTimestamp)
        )
        AND (filterActivityTypeAll
            OR (filterActivityTypeSend
                AND (filterAllAddresses
                    OR (HEX(transfers.sender) IN filter_addresses)
                )
            )
            OR (filterActivityTypeReceive
                AND (filterAllAddresses OR (HEX(transfers.address) IN filter_addresses))
            )
        )
        AND (filterAllAddresses
            OR (HEX(transfers.sender) IN filter_addresses)
            OR (HEX(transfers.address) IN filter_addresses)
        )
        AND (filterAllToAddresses
            OR (HEX(transfers.address) IN filter_to_addresses)
        )
        AND (includeAllTokenTypeAssets OR (transfers.type = "eth" AND ("ETH" IN assets_token_codes))
            OR (transfers.type = "erc20" AND ((transfers.network_id, transfers.token_address) IN assets_erc20)))
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
            WHEN from_join.address IS NOT NULL AND to_join.address IS NULL THEN 1
            WHEN to_join.address IS NOT NULL AND from_join.address IS NULL THEN 2
            WHEN from_join.address IS NOT NULL AND to_join.address IS NOT NULL THEN
                CASE
                    WHEN from_join.address < to_join.address THEN 1
                    ELSE 2
                END
            ELSE NULL
        END as tr_type,

        pending_transactions.from_address AS from_address,
        pending_transactions.to_address AS to_address,
        pending_transactions.value AS tr_amount,
        NULL AS mt_from_amount,
        NULL AS mt_to_amount,

        statusPending AS agg_status,
        1 AS agg_count,

        NULL AS token_address,
        pending_transactions.symbol AS token_code,
        NULL AS from_token_code,
        NULL AS to_token_code
    FROM pending_transactions, filter_conditions
    LEFT JOIN
        filter_addresses from_join ON HEX(pending_transactions.from_address) = from_join.address
    LEFT JOIN
        filter_addresses to_join ON HEX(pending_transactions.to_address) = to_join.address
    WHERE pending_transactions.multi_transaction_id = 0
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
        NULL AS tr_amount,
        multi_transactions.from_amount AS mt_from_amount,
        multi_transactions.to_amount AS mt_to_amount,

        CASE
            WHEN tr_status.min_status = 1 AND pending_status.count IS NULL THEN statusSuccess
            WHEN tr_status.min_status = 0 THEN statusFailed
            ELSE statusPending
        END AS agg_status,

        COALESCE(tr_status.count, 0) + COALESCE(pending_status.count, 0) AS agg_count,

        NULL AS token_address,
        NULL AS token_code,
        multi_transactions.from_asset AS from_token_code,
        multi_transactions.to_asset AS to_token_code
    FROM multi_transactions, filter_conditions
    JOIN tr_status ON multi_transactions.ROWID = tr_status.multi_transaction_id
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
        AND (filterAllActivityStatus OR ((filterStatusCompleted OR filterStatusFinalized) AND agg_status = statusSuccess)
        OR (filterStatusFailed AND agg_status = statusFailed) OR (filterStatusPending AND agg_status = statusPending))

    ORDER BY timestamp DESC
    LIMIT ? OFFSET ?`

	noEntriesInTmpTableSQLValues           = "(NULL)"
	noEntriesInTwoColumnsTmpTableSQLValues = "(NULL, NULL)"
)

type FilterDependencies struct {
	db              *sql.DB
	tokenSymbol     func(token Token) string
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
					return fmt.Sprintf("%d, '%s'", item.ChainID, item.Address.Hex())
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

	queryString := fmt.Sprintf(queryFormatString, involvedAddresses, toAddresses, assetsTokenCodes, assetsERC20, networks,
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
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var transferHash, pendingHash []byte
		var chainID, multiTxID, aggregatedCount sql.NullInt64
		var timestamp int64
		var dbMtType, dbTrType sql.NullByte
		var toAddress, fromAddress eth.Address
		var aggregatedStatus int
		var dbTrAmount sql.NullString
		var dbMtFromAmount, dbMtToAmount sql.NullString
		var tokenAddress, tokenCode, fromTokenCode, toTokenCode sql.NullString
		err := rows.Scan(&transferHash, &pendingHash, &chainID, &multiTxID, &timestamp, &dbMtType, &dbTrType, &fromAddress,
			&toAddress, &dbTrAmount, &dbMtFromAmount, &dbMtToAmount, &aggregatedStatus, &aggregatedCount,
			&tokenAddress, &tokenCode, &fromTokenCode, &toTokenCode)
		if err != nil {
			return nil, err
		}

		getActivityType := func(trType sql.NullByte) (activityType Type, filteredAddress eth.Address) {
			if trType.Valid {
				if trType.Byte == fromTrType {
					return SendAT, fromAddress
				} else if trType.Byte == toTrType {
					return ReceiveAT, toAddress
				}
			}
			log.Warn(fmt.Sprintf("unexpected activity type. Missing [%s, %s] in the addresses table?", fromAddress, toAddress))
			return ReceiveAT, toAddress
		}

		// Can be mapped directly because the values are injected into the query
		activityStatus := Status(aggregatedStatus)
		var tokenOut, tokenIn *Token

		var entry Entry
		if transferHash != nil && chainID.Valid {
			// Extract activity type: SendAT/ReceiveAT
			activityType, filteredAddress := getActivityType(dbTrType)

			inAmount, outAmount := getTrInAndOutAmounts(activityType, dbTrAmount)

			// Extract tokens
			var involvedToken *Token
			if tokenAddress.Valid && eth.HexToAddress(tokenAddress.String) != eth.HexToAddress("0x") {
				involvedToken = &Token{TokenType: Erc20, ChainID: common.ChainID(chainID.Int64), Address: eth.HexToAddress(tokenAddress.String)}
			} else {
				involvedToken = &Token{TokenType: Native, ChainID: common.ChainID(chainID.Int64)}
			}
			if activityType == SendAT {
				tokenOut = involvedToken
			} else {
				tokenIn = involvedToken
			}

			entry = newActivityEntryWithSimpleTransaction(
				&transfer.TransactionIdentity{ChainID: common.ChainID(chainID.Int64), Hash: eth.BytesToHash(transferHash), Address: filteredAddress},
				timestamp, activityType, activityStatus, inAmount, outAmount, tokenOut, tokenIn)
		} else if pendingHash != nil && chainID.Valid {
			// Extract activity type: PendingAT
			activityType, _ := getActivityType(dbTrType)

			inAmount, outAmount := getTrInAndOutAmounts(activityType, dbTrAmount)

			// Extract tokens
			if tokenCode.Valid {
				cID := common.ChainID(chainID.Int64)
				tokenOut = deps.tokenFromSymbol(&cID, tokenCode.String)
			}

			entry = newActivityEntryWithPendingTransaction(&transfer.TransactionIdentity{ChainID: common.ChainID(chainID.Int64), Hash: eth.BytesToHash(pendingHash)},
				timestamp, activityType, activityStatus, inAmount, outAmount, tokenOut, tokenIn)
		} else if multiTxID.Valid {
			mtInAmount, mtOutAmount := getMtInAndOutAmounts(dbMtFromAmount, dbMtToAmount)

			// Extract activity type: SendAT/SwapAT/BridgeAT
			activityType := multiTransactionTypeToActivityType(transfer.MultiTransactionType(dbMtType.Byte))

			// Extract tokens
			if fromTokenCode.Valid {
				tokenOut = deps.tokenFromSymbol(nil, fromTokenCode.String)
			}
			if toTokenCode.Valid {
				tokenIn = deps.tokenFromSymbol(nil, toTokenCode.String)
			}

			entry = NewActivityEntryWithMultiTransaction(transfer.MultiTransactionIDType(multiTxID.Int64),
				timestamp, activityType, activityStatus, mtInAmount, mtOutAmount, tokenOut, tokenIn)
		} else {
			return nil, errors.New("invalid row data")
		}
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
