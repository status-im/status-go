package activity

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	eth "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/transfer"

	"golang.org/x/exp/constraints"
)

type PayloadType = int

// Beware if adding/removing please check if affected and update the functions below
// - NewActivityEntryWithTransaction
// - multiTransactionTypeToActivityType
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
}

type jsonSerializationTemplate struct {
	PayloadType    PayloadType                     `json:"payloadType"`
	Transaction    *transfer.TransactionIdentity   `json:"transaction"`
	ID             transfer.MultiTransactionIDType `json:"id"`
	Timestamp      int64                           `json:"timestamp"`
	ActivityType   Type                            `json:"activityType"`
	ActivityStatus Status                          `json:"activityStatus"`
	TokenType      TokenType                       `json:"tokenType"`
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
	return nil
}

func NewActivityEntryWithTransaction(payloadType PayloadType, transaction *transfer.TransactionIdentity, timestamp int64, activityType Type, activityStatus Status) Entry {
	if payloadType != SimpleTransactionPT && payloadType != PendingTransactionPT {
		panic("invalid transaction type")
	}

	return Entry{
		payloadType:    payloadType,
		transaction:    transaction,
		id:             0,
		timestamp:      timestamp,
		activityType:   activityType,
		activityStatus: activityStatus,
		tokenType:      AssetTT,
	}
}

func NewActivityEntryWithMultiTransaction(id transfer.MultiTransactionIDType, timestamp int64, activityType Type, activityStatus Status) Entry {
	return Entry{
		payloadType:    MultiTransactionPT,
		id:             id,
		timestamp:      timestamp,
		activityType:   activityType,
		activityStatus: activityStatus,
		tokenType:      AssetTT,
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
	//toTrType   = byte(2)

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

			? AS filterAllAddresses,
			? AS filterAllToAddresses,
			? AS filterAllActivityStatus,
			? AS includeAllTokenTypeAssets,
			? AS statusIsPending,

			? AS includeAllNetworks
		),
		filter_addresses(address) AS (
			SELECT HEX(address) FROM accounts WHERE (SELECT filterAllAddresses FROM filter_conditions) != 0
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
		)
	SELECT
		transfers.hash AS transfer_hash,
		NULL AS pending_hash,
		transfers.network_id AS network_id,
		0 AS multi_tx_id,
		transfers.timestamp AS timestamp,
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

		transfers.sender AS from_address,
		transfers.address AS to_address
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
		pending_transactions.to_address AS to_address
	FROM pending_transactions, filter_conditions
	LEFT JOIN
		filter_addresses from_join ON HEX(pending_transactions.from_address) = from_join.address
	LEFT JOIN
		filter_addresses to_join ON HEX(pending_transactions.to_address) = to_join.address
	WHERE pending_transactions.multi_transaction_id = 0
		AND (filterAllActivityStatus OR statusIsPending)
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
		multi_transactions.to_address AS to_address
	FROM multi_transactions, filter_conditions
	WHERE ((startFilterDisabled OR timestamp >= startTimestamp)
			AND (endFilterDisabled OR timestamp <= endTimestamp)
		)
		AND (filterActivityTypeAll OR (multi_transactions.type IN (%s)))
		AND (filterAllAddresses
			OR (HEX(multi_transactions.from_address) IN filter_addresses)
			OR (HEX(multi_transactions.to_address) IN filter_addresses)
		)
		AND (filterAllToAddresses
			OR (HEX(multi_transactions.to_address) IN filter_to_addresses)
		)
		AND (includeAllTokenTypeAssets OR (UPPER(multi_transactions.from_asset) IN filter_assets) OR (UPPER(multi_transactions.to_asset) IN filter_assets))

	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?`

	noEntriesInTmpTableSQLValues = "(NULL)"
)

// GetActivityEntries returns query the transfers, pending_transactions, and multi_transactions tables
// based on filter parameters and arguments
// it returns metadata for all entries ordered by timestamp column
//
// Adding a no-limit option was never considered or required.
func GetActivityEntries(db *sql.DB, addresses []eth.Address, chainIDs []common.ChainID, filter Filter, offset int, limit int) ([]Entry, error) {
	// TODO: filter collectibles after  they are added to multi_transactions table
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

	statusIsPending := false
	if !includeAllStatuses {
		statusIsPending = sliceContains(filter.Statuses, PendingAS)
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

	rows, err := db.Query(queryString,
		startFilterDisabled, filter.Period.StartTimestamp, endFilterDisabled, filter.Period.EndTimestamp,
		filterActivityTypeAll, sliceContains(filter.Types, SendAT), sliceContains(filter.Types, ReceiveAT),
		filterAllAddresses, filterAllToAddresses, includeAllStatuses, includeAllTokenTypeAssets, statusIsPending,
		includeAllNetworks,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var transferHash, pendingHash []byte
		var chainID, multiTxID sql.NullInt64
		var timestamp int64
		var dbMtType, dbTrType sql.NullByte
		var toAddress, fromAddress eth.Address
		err := rows.Scan(&transferHash, &pendingHash, &chainID, &multiTxID, &timestamp, &dbMtType, &dbTrType, &fromAddress, &toAddress)
		if err != nil {
			return nil, err
		}

		getActivityType := func(trType sql.NullByte) (activityType Type, filteredAddress eth.Address) {
			if trType.Valid && trType.Byte == fromTrType {
				return SendAT, fromAddress
			}
			// Don't expect this to happen due to trType = NULL outside of tests
			return ReceiveAT, toAddress
		}

		var entry Entry
		if transferHash != nil && chainID.Valid {
			// TODO: extend DB with status in order to filter by status. The status has to be extracted from the receipt upon downloading
			activityStatus := FinalizedAS
			activityType, filteredAddress := getActivityType(dbTrType)
			entry = NewActivityEntryWithTransaction(SimpleTransactionPT,
				&transfer.TransactionIdentity{ChainID: common.ChainID(chainID.Int64), Hash: eth.BytesToHash(transferHash), Address: filteredAddress},
				timestamp, activityType, activityStatus)
		} else if pendingHash != nil && chainID.Valid {
			activityStatus := PendingAS
			activityType, _ := getActivityType(dbTrType)
			entry = NewActivityEntryWithTransaction(PendingTransactionPT,
				&transfer.TransactionIdentity{ChainID: common.ChainID(chainID.Int64), Hash: eth.BytesToHash(pendingHash)},
				timestamp, activityType, activityStatus)
		} else if multiTxID.Valid {
			activityType := multiTransactionTypeToActivityType(transfer.MultiTransactionType(dbMtType.Byte))
			// TODO: aggregate status from all sub-transactions
			activityStatus := FinalizedAS
			entry = NewActivityEntryWithMultiTransaction(transfer.MultiTransactionIDType(multiTxID.Int64),
				timestamp, activityType, activityStatus)
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
