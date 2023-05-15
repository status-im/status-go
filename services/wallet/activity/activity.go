package activity

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/services/wallet/transfer"
)

type PayloadType = int

const (
	MultiTransactionPT PayloadType = iota + 1
	SimpleTransactionPT
	PendingTransactionPT
)

type Entry struct {
	// TODO: rename in payloadType
	transactionType PayloadType
	transaction     *transfer.TransactionIdentity
	id              transfer.MultiTransactionIDType
	timestamp       int64
	activityType    Type
}

type jsonSerializationTemplate struct {
	TransactionType PayloadType                     `json:"transactionType"`
	Transaction     *transfer.TransactionIdentity   `json:"transaction"`
	ID              transfer.MultiTransactionIDType `json:"id"`
	Timestamp       int64                           `json:"timestamp"`
	ActivityType    Type                            `json:"activityType"`
}

func (e *Entry) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonSerializationTemplate{
		TransactionType: e.transactionType,
		Transaction:     e.transaction,
		ID:              e.id,
		Timestamp:       e.timestamp,
		ActivityType:    e.activityType,
	})
}

func (e *Entry) UnmarshalJSON(data []byte) error {
	aux := jsonSerializationTemplate{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	e.transactionType = aux.TransactionType
	e.transaction = aux.Transaction
	e.id = aux.ID
	e.timestamp = aux.Timestamp
	e.activityType = aux.ActivityType
	return nil
}

func NewActivityEntryWithTransaction(transactionType PayloadType, transaction *transfer.TransactionIdentity, timestamp int64, activityType Type) Entry {
	if transactionType != SimpleTransactionPT && transactionType != PendingTransactionPT {
		panic("invalid transaction type")
	}

	return Entry{
		transactionType: transactionType,
		transaction:     transaction,
		id:              0,
		timestamp:       timestamp,
		activityType:    activityType,
	}
}

func NewActivityEntryWithMultiTransaction(id transfer.MultiTransactionIDType, timestamp int64, activityType Type) Entry {
	return Entry{
		transactionType: MultiTransactionPT,
		id:              id,
		timestamp:       timestamp,
		activityType:    activityType,
	}
}

func (e *Entry) TransactionType() PayloadType {
	return e.transactionType
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

func typesContain(slice []Type, item Type) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func joinMTTypes(types []transfer.MultiTransactionType) string {
	var sb strings.Builder
	for i, val := range types {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(strconv.Itoa(int(val)))
	}

	return sb.String()
}

func joinAddresses(addresses []common.Address) string {
	var sb strings.Builder
	for i, address := range addresses {
		if i == 0 {
			sb.WriteString("('")
		} else {
			sb.WriteString("'),('")
		}
		sb.WriteString(strings.ToUpper(hex.EncodeToString(address[:])))
	}
	sb.WriteString("')")

	return sb.String()
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

// TODO: extend with SEND/RECEIVE for transfers and pending_transactions
// TODO: clarify if we include sender and receiver in pending_transactions as we do for transfers
// TODO optimization: consider implementing nullable []byte instead of using strings for addresses
// Query includes duplicates, will return multiple rows for the same transaction
const queryFormatString = `
	WITH filter_conditions AS (
		SELECT
			? AS startFilterDisabled,
			? AS startTimestamp,
			? AS endFilterDisabled,
			? AS endTimestamp,

			? AS filterActivityTypeAll,
			? AS filterActivityTypeSend,
			? AS filterActivityTypeReceive,

			? AS filterAllAddresses
		),
		filter_addresses(address) AS (
			VALUES %s
		)
	SELECT
		transfers.hash AS transfer_hash,
		NULL AS pending_hash,
		transfers.network_id AS network_id,
		0 AS multi_tx_id,
		transfers.timestamp AS timestamp,
		NULL AS mt_type,
		HEX(transfers.address) AS owner_address
	FROM transfers, filter_conditions
	WHERE transfers.multi_transaction_id = 0
		AND ((startFilterDisabled OR timestamp >= startTimestamp) AND (endFilterDisabled OR timestamp <= endTimestamp))
		AND (filterActivityTypeAll OR (filterActivityTypeSend AND (filterAllAddresses OR (HEX(transfers.sender) IN filter_addresses))) OR (filterActivityTypeReceive AND (filterAllAddresses OR (HEX(transfers.address) IN filter_addresses))))
		AND (filterAllAddresses OR (HEX(transfers.sender) IN filter_addresses) OR (HEX(transfers.address) IN filter_addresses))

	UNION ALL

	SELECT
		NULL AS transfer_hash,
		pending_transactions.hash AS pending_hash,
		pending_transactions.network_id AS network_id,
		0 AS multi_tx_id,
		pending_transactions.timestamp AS timestamp,
		NULL AS mt_type,
		NULL AS owner_address
	FROM pending_transactions, filter_conditions
	WHERE pending_transactions.multi_transaction_id = 0
		AND ((startFilterDisabled OR timestamp >= startTimestamp) AND (endFilterDisabled OR timestamp <= endTimestamp))
		AND (filterActivityTypeAll OR filterActivityTypeSend)
		AND (filterAllAddresses OR (HEX(pending_transactions.from_address) IN filter_addresses) OR (HEX(pending_transactions.to_address) IN filter_addresses))

	UNION ALL

	SELECT
		NULL AS transfer_hash,
		NULL AS pending_hash,
		NULL AS network_id,
		multi_transactions.ROWID AS multi_tx_id,
		multi_transactions.timestamp AS timestamp,
		multi_transactions.type AS mt_type,
		NULL AS owner_address
	FROM multi_transactions, filter_conditions
	WHERE ((startFilterDisabled OR timestamp >= startTimestamp) AND (endFilterDisabled OR timestamp <= endTimestamp))
		AND (filterActivityTypeAll OR (multi_transactions.type IN (%s)))
		AND (filterAllAddresses OR (HEX(multi_transactions.from_address) IN filter_addresses) OR (HEX(multi_transactions.to_address) IN filter_addresses))

	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?`

func GetActivityEntries(db *sql.DB, addresses []common.Address, chainIDs []uint64, filter Filter, offset int, limit int) ([]Entry, error) {
	// Query the transfers, pending_transactions, and multi_transactions tables ordered by timestamp column

	// TODO: finish filter: chainIDs, statuses, tokenTypes, counterpartyAddresses
	// TODO: use all accounts list for detecting SEND/RECEIVE instead of the current addresses list; also change activityType detection in transfer part
	startFilterDisabled := !(filter.Period.StartTimestamp > 0)
	endFilterDisabled := !(filter.Period.EndTimestamp > 0)
	filterActivityTypeAll := typesContain(filter.Types, AllAT) || len(filter.Types) == 0
	filterAllAddresses := len(addresses) == 0

	//fmt.Println("@dd filter: timeEnabled", filter.Period.StartTimestamp, filter.Period.EndTimestamp, "; type", filter.Types, "offset", offset, "limit", limit)

	joinedAddresses := "(NULL)"
	if !filterAllAddresses {
		joinedAddresses = joinAddresses(addresses)
	}

	mtTypes := activityTypesToMultiTransactionTypes(filter.Types)
	joinedMTTypes := joinMTTypes(mtTypes)

	queryString := fmt.Sprintf(queryFormatString, joinedAddresses, joinedMTTypes)

	rows, err := db.Query(queryString,
		startFilterDisabled, filter.Period.StartTimestamp, endFilterDisabled, filter.Period.EndTimestamp,
		filterActivityTypeAll, typesContain(filter.Types, SendAT), typesContain(filter.Types, ReceiveAT),
		filterAllAddresses,
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
		var dbActivityType sql.NullByte
		var dbAddress sql.NullString
		err := rows.Scan(&transferHash, &pendingHash, &chainID, &multiTxID, &timestamp, &dbActivityType, &dbAddress)
		if err != nil {
			return nil, err
		}

		var entry Entry
		if transferHash != nil && chainID.Valid {
			var activityType Type = SendAT
			thisAddress := common.HexToAddress(dbAddress.String)
			for _, address := range addresses {
				if address == thisAddress {
					activityType = ReceiveAT
				}
			}
			entry = NewActivityEntryWithTransaction(SimpleTransactionPT, &transfer.TransactionIdentity{ChainID: uint64(chainID.Int64), Hash: common.BytesToHash(transferHash), Address: thisAddress}, timestamp, activityType)
		} else if pendingHash != nil && chainID.Valid {
			var activityType Type = SendAT
			entry = NewActivityEntryWithTransaction(PendingTransactionPT, &transfer.TransactionIdentity{ChainID: uint64(chainID.Int64), Hash: common.BytesToHash(pendingHash)}, timestamp, activityType)
		} else if multiTxID.Valid {
			activityType := multiTransactionTypeToActivityType(transfer.MultiTransactionType(dbActivityType.Byte))
			entry = NewActivityEntryWithMultiTransaction(transfer.MultiTransactionIDType(multiTxID.Int64),
				timestamp, activityType)
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
