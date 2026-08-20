package activity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	eth "github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/internal/db/sqlite"
	"github.com/status-im/status-go/internal/logutils"
	ac "github.com/status-im/status-go/pkg/services/wallet/activity/common"
	wCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty"
	"github.com/status-im/status-go/pkg/services/wallet/thirdparty/activity/alchemy"
)

// getFetchedEntriesByIDs fetches Alchemy transaction details by IDs
// Returns a map keyed by "chainID-hash-address" string
func getFetchedEntriesByIDs(ctx context.Context, deps FilterDependencies, txIDs []OrderedTransactionID) (map[string]Entry, error) {

	if len(txIDs) == 0 {
		return make(map[string]Entry), nil
	}

	var chainIDs []wCommon.ChainID
	var hashStrings []string
	chainIDSet := make(map[wCommon.ChainID]bool)

	for _, txID := range txIDs {
		if txID.Source != FetchedTransaction {
			continue
		}
		hashStrings = append(hashStrings, txID.Hash.Hex())
		chainIDSet[txID.ChainID] = true
	}

	if len(hashStrings) == 0 {
		return make(map[string]Entry), nil
	}

	for chainID := range chainIDSet {
		chainIDs = append(chainIDs, chainID)
	}

	q := sq.Select("e.transfer", "e.chain_id", "e.address").
		From("fetched_alchemy_transfers e").
		Where(sq.And{
			sq.Eq{"e.chain_id": chainIDs},
			sq.Eq{"e.hash": hashStrings}})

	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build fetched entries query: %w", err)
	}

	stmt, err := deps.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare fetched entries query: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query fetched entries: %w", err)
	}
	defer rows.Close()

	transfersMap, err := rowsToTransfersGrouped(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to convert rows to transfers: %w", err)
	}

	result := make(map[string]Entry)
	for chainID, addressMap := range transfersMap {
		for address, transfers := range addressMap {
			var activityEntries []thirdparty.ActivityEntry = alchemy.TransfersToThirdpartyActivityEntries(transfers, uint64(chainID), address)
			entries := thirdpartyActivityEntriesToEntries(deps, activityEntries)

			for _, entry := range entries {
				if entry.transaction != nil {
					key := fmt.Sprintf("%d-%s-%s",
						entry.transaction.ChainID,
						entry.transaction.Hash.Hex(),
						address.Hex())
					result[key] = entry
				}
			}
		}
	}

	return result, nil
}

// rowsToTransfersGrouped returns transfers grouped by chainId and then by address
func rowsToTransfersGrouped(rows *sql.Rows) (map[wCommon.ChainID]map[eth.Address][]alchemy.Transfer, error) {
	transfersMap := make(map[wCommon.ChainID]map[eth.Address][]alchemy.Transfer)

	var counter int = 0

	for rows.Next() {
		var transfer alchemy.Transfer
		var chainID uint64
		var address eth.Address
		var transferJSON = sqlite.ToJSONBlob(&transfer)

		err := rows.Scan(transferJSON, &chainID, &address)
		if err != nil {
			return nil, err
		}
		if !transferJSON.Valid {
			return nil, errors.New("invalid entry")
		}

		// Initialize nested map if needed
		wChainID := wCommon.ChainID(chainID)
		if transfersMap[wChainID] == nil {
			transfersMap[wChainID] = make(map[eth.Address][]alchemy.Transfer)
		}
		transfersMap[wChainID][address] = append(transfersMap[wChainID][address], transfer)
		counter = counter + 1
	}

	return transfersMap, nil
}

func thirdpartyActivityEntriesToEntries(deps FilterDependencies, activityEntries []thirdparty.ActivityEntry) []Entry {
	entries := make([]Entry, 0, len(activityEntries))

	for _, ae := range activityEntries {
		// Determine the main chain ID and set chainIDOut/chainIDIn
		// uChainID := wCommon.UnknownChainID
		var chainIDOut *wCommon.ChainID
		var chainIDIn *wCommon.ChainID
		if ae.ChainIDOut != nil {
			chainIDOut = new(wCommon.ChainID)
			*chainIDOut = wCommon.ChainID(*ae.ChainIDOut)
		}
		if ae.ChainIDIn != nil {
			chainIDIn = new(wCommon.ChainID)
			*chainIDIn = wCommon.ChainID(*ae.ChainIDIn)
		}
		var chainID wCommon.ChainID
		if chainIDOut != nil {
			chainID = *chainIDOut
		} else if chainIDIn != nil {
			chainID = *chainIDIn
		} else {
			chainID = wCommon.ChainID(wCommon.UnknownChainID)
		}

		entry := Entry{
			payloadType: ac.SimpleTransactionPT,
			transaction: &ac.TransactionIdentity{
				ChainID: chainID,
				Hash:    ae.TxHash,
				Address: ae.Sender,
			},
			timestamp:                 ae.Timestamp,
			activityType:              ae.ActivityType,
			activityStatus:            getActivityStatusFromTxStatus(ae.TxStatus, ae.Timestamp, chainID),
			amountOut:                 ae.AmountOut,
			amountIn:                  ae.AmountIn,
			tokenOut:                  ae.TokenOut,
			tokenIn:                   ae.TokenIn,
			sender:                    &ae.Sender,
			recipient:                 ae.Recipient,
			chainIDOut:                chainIDOut,
			chainIDIn:                 chainIDIn,
			transferType:              getTransferTypeFromTokens(ae.TokenIn, ae.TokenOut),
			interactedContractAddress: ae.ContractAddress,
		}

		entry.symbolOut, entry.symbolIn = lookupAndFillInTokens(deps, entry.tokenOut, entry.tokenIn)
		entries = append(entries, entry)
	}

	return entries
}

func getActivityStatusFromTxStatus(status ac.TxStatus, timestamp int64, chainID wCommon.ChainID) ac.Status {
	switch status {
	case ac.Pending:
		return ac.PendingAS
	case ac.Success:
		now := time.Now().Unix()
		if timestamp+getFinalizationPeriod(chainID) > now {
			return ac.FinalizedAS
		}
		return ac.CompleteAS
	case ac.Failed:
		return ac.FailedAS
	}

	logutils.ZapLogger().Error("unhandled transaction status value")
	return ac.FailedAS
}

func getTransferTypeFromTokens(tokenIn, tokenOut *ac.Token) *ac.TransferType {
	var token *ac.Token
	if tokenIn != nil {
		token = tokenIn
	} else if tokenOut != nil {
		token = tokenOut
	} else {
		return nil
	}

	ret := new(ac.TransferType)
	switch token.TokenType {
	case ac.Native:
		*ret = ac.TransferTypeEth
	case ac.Erc20:
		*ret = ac.TransferTypeErc20
	case ac.Erc721:
		*ret = ac.TransferTypeErc721
	case ac.Erc1155:
		*ret = ac.TransferTypeErc1155
	default:
		return nil
	}

	return ret
}
