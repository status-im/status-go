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
	tokenType      TokenType
	amountOut      *hexutil.Big // Used for activityType SendAT, SwapAT, BridgeAT
	amountIn       *hexutil.Big // Used for activityType ReceiveAT, BuyAT, SwapAT, BridgeAT
}

type jsonSerializationTemplate struct {
	PayloadType    PayloadType                     `json:"payloadType"`
	Transaction    *transfer.TransactionIdentity   `json:"transaction"`
	ID             transfer.MultiTransactionIDType `json:"id"`
	Timestamp      int64                           `json:"timestamp"`
	ActivityType   Type                            `json:"activityType"`
	ActivityStatus Status                          `json:"activityStatus"`
	TokenType      TokenType                       `json:"tokenType"`
	AmountOut      *hexutil.Big                    `json:"amountOut"`
	AmountIn       *hexutil.Big                    `json:"amountIn"`
}

func (e *Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonSerializationTemplate{
		PayloadType:    e.payloadType,
		Transaction:    e.transaction,
		ID:             e.id,
		Timestamp:      e.timestamp,
		ActivityType:   e.activityType,
		ActivityStatus: e.activityStatus,
		TokenType:      e.tokenType,
		AmountOut:      e.amountOut,
		AmountIn:       e.amountIn,
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

func newActivityEntryWithPendingTransaction(transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status, amountIn *hexutil.Big, amountOut *hexutil.Big) Entry {
	return newActivityEntryWithTransaction(true, transaction, timestamp, activityType, activityStatus, amountIn, amountOut)
}

func newActivityEntryWithSimpleTransaction(transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status, amountIn *hexutil.Big, amountOut *hexutil.Big) Entry {
	return newActivityEntryWithTransaction(false, transaction, timestamp, activityType, activityStatus, amountIn, amountOut)
}

func newActivityEntryWithTransaction(pending bool, transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status, amountIn *hexutil.Big, amountOut *hexutil.Big) Entry {
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
		tokenType:      AssetTT,
		amountIn:       amountIn,
		amountOut:      amountOut,
	}
}

func NewActivityEntryWithMultiTransaction(id transfer.MultiTransactionIDType, timestamp int64, activityType Type, activityStatus Status, amountIn *hexutil.Big, amountOut *hexutil.Big) Entry {
	return Entry{
		payloadType:    MultiTransactionPT,
		id:             id,
		timestamp:      timestamp,
		activityType:   activityType,
		activityStatus: activityStatus,
		tokenType:      AssetTT,
		amountIn:       amountIn,
		amountOut:      amountOut,
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
		filter_assets(token_code) AS (
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

		1 AS agg_count
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
		AND (includeAllTokenTypeAssets OR (transfers.type = "eth" AND ("ETH" IN filter_assets)))
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
		1 AS agg_count
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
		AND (includeAllTokenTypeAssets OR (UPPER(pending_transactions.symbol) IN filter_assets))
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

		COALESCE(tr_status.count, 0) + COALESCE(pending_status.count, 0) AS agg_count

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
		AND (includeAllTokenTypeAssets OR (UPPER(multi_transactions.from_asset) IN filter_assets)
			OR (UPPER(multi_transactions.to_asset) IN filter_assets)
		)
		AND (filterAllActivityStatus OR ((filterStatusCompleted OR filterStatusFinalized) AND agg_status = statusSuccess)
		OR (filterStatusFailed AND agg_status = statusFailed) OR (filterStatusPending AND agg_status = statusPending))

	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?`

	noEntriesInTmpTableSQLValues = "(NULL)"
)

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
		fromAmount, frOk := new(big.Int).SetString(dbFromAmount.String, 16)
		toAmount, toOk := new(big.Int).SetString(dbToAmount.String, 16)
		if frOk && toOk {
			inAmount = (*hexutil.Big)(toAmount)
			outAmount = (*hexutil.Big)(fromAmount)
			return
		}
		log.Warn(fmt.Sprintf("could not parse amounts %s %s", dbFromAmount.String, dbToAmount.String))
	} else {
		log.Warn("invalid transaction amounts")
	}
	inAmount = (*hexutil.Big)(big.NewInt(0))
	outAmount = (*hexutil.Big)(big.NewInt(0))
	return
}

// getActivityEntries queries the transfers, pending_transactions, and multi_transactions tables
// based on filter parameters and arguments
// it returns metadata for all entries ordered by timestamp column
//
// Adding a no-limit option was never considered or required.
func getActivityEntries(ctx context.Context, db *sql.DB, addresses []eth.Address, chainIDs []common.ChainID, filter Filter, offset int, limit int) ([]Entry, error) {
	// TODO: filter collectibles after they are added to multi_transactions table
	if len(filter.Tokens.EnabledTypes) > 0 && !sliceContains(filter.Tokens.EnabledTypes, AssetTT) {
		// For now we deal only with assets so return empty result
		return []Entry{}, nil
	}

	includeAllTokenTypeAssets := (len(filter.Tokens.EnabledTypes) == 0 ||
		sliceContains(filter.Tokens.EnabledTypes, AssetTT)) && len(filter.Tokens.Assets) == 0

	assets := noEntriesInTmpTableSQLValues
	if !includeAllTokenTypeAssets {
		assets = joinItems(filter.Tokens.Assets, func(item TokenCode) string { return fmt.Sprintf("'%v'", item) })
	}

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

	queryString := fmt.Sprintf(queryFormatString, involvedAddresses, toAddresses, assets, networks,
		joinedMTTypes)

	rows, err := db.QueryContext(ctx, queryString,
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
		err := rows.Scan(&transferHash, &pendingHash, &chainID, &multiTxID, &timestamp, &dbMtType, &dbTrType, &fromAddress, &toAddress, &dbTrAmount, &dbMtFromAmount, &dbMtToAmount, &aggregatedStatus, &aggregatedCount)
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

		var entry Entry
		if transferHash != nil && chainID.Valid {
			activityType, filteredAddress := getActivityType(dbTrType)
			inAmount, outAmount := getTrInAndOutAmounts(activityType, dbTrAmount)
			entry = newActivityEntryWithSimpleTransaction(
				&transfer.TransactionIdentity{ChainID: common.ChainID(chainID.Int64), Hash: eth.BytesToHash(transferHash), Address: filteredAddress},
				timestamp, activityType, activityStatus, inAmount, outAmount)
		} else if pendingHash != nil && chainID.Valid {
			activityType, _ := getActivityType(dbTrType)
			inAmount, outAmount := getTrInAndOutAmounts(activityType, dbTrAmount)
			entry = newActivityEntryWithPendingTransaction(&transfer.TransactionIdentity{ChainID: common.ChainID(chainID.Int64), Hash: eth.BytesToHash(pendingHash)},
				timestamp, activityType, activityStatus, inAmount, outAmount)
		} else if multiTxID.Valid {
			mtInAmount, mtOutAmount := getMtInAndOutAmounts(dbMtFromAmount, dbMtToAmount)
			activityType := multiTransactionTypeToActivityType(transfer.MultiTransactionType(dbMtType.Byte))
			entry = NewActivityEntryWithMultiTransaction(transfer.MultiTransactionIDType(multiTxID.Int64),
				timestamp, activityType, activityStatus, mtInAmount, mtOutAmount)
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
