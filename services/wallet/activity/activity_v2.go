package activity

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	ac "github.com/status-im/status-go/services/wallet/activity/common"
	wCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/requests"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/thirdparty"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/sqlite"
)

type EntryType string

const (
	EntryTypeSentTransaction    EntryType = "sent_transaction"
	EntryTypeFetchedTransaction EntryType = "fetched_transaction"
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

// getActivityEntriesV2 queries the route_* and tracked_transactions based on filter parameters and arguments
// it returns metadata for all entries ordered by timestamp column
func getActivityEntriesV2(ctx context.Context, deps FilterDependencies, addresses []eth.Address, allAddresses bool, chainIDs []wCommon.ChainID, filter Filter, offset int, limit int) ([]Entry, error) {
	if len(addresses) == 0 {
		return nil, ErrNoAddressesProvided
	}
	if len(chainIDs) == 0 {
		return nil, ErrNoChainIDsProvided
	}

	// Get all sent transactions
	sentTxQ := sq.Select(`
		'sent_transaction' as entry_type,
		st.tx_json as sent_tx_json,
		rpt.tx_args_json as sent_tx_args_json,
		rpt.is_approval as sent_is_approval,
		rp.path_json as sent_path_json,
		rip.route_input_params_json as sent_route_input_params_json,
		tt.tx_status as tx_status,
		tt.timestamp as timestamp,
		NULL as fetched_entry,
		NULL as fetch_params
	`)
	sentTxQ = sentTxQ.From("sent_transactions st").
		LeftJoin(`route_path_transactions rpt ON
			st.chain_id = rpt.chain_id AND
			st.tx_hash = rpt.tx_hash`).
		LeftJoin(`tracked_transactions tt ON
			st.chain_id = tt.chain_id AND
			st.tx_hash = tt.tx_hash`).
		LeftJoin(`route_paths rp ON
			rpt.uuid = rp.uuid AND
			rpt.path_idx = rp.path_idx`).
		LeftJoin(`route_input_parameters rip ON
			rpt.uuid = rip.uuid`)
	sentTxCond := sq.And{}
	sentTxCond = append(sentTxCond, sq.Eq{"rpt.chain_id": chainIDs})
	sentTxCond = append(sentTxCond, sq.Or{
		sq.Eq{"rip.from_address": addresses},
		sq.Eq{"rip.to_address": addresses},
	})
	sentTxQ = sentTxQ.Where(sentTxCond)

	sentTxQuery, sentTxArgs, err := sentTxQ.ToSql()
	if err != nil {
		return nil, err
	}
	fmt.Println("sentTxQuery")
	fmt.Println(sentTxQuery)
	fmt.Println(sentTxArgs)

	// Get all fetched transactions
	fetchedTxQ := sq.Select(`
		'fetched_transaction' as entry_type,
		NULL as sent_tx_json,
		NULL as sent_tx_args_json,
		NULL as sent_is_approval,
		NULL as sent_path_json,
		NULL as sent_route_input_params_json,
		fae.entry ->> '$.txStatus' as tx_status,
		fae.timestamp as timestamp,
		fae.entry as fetched_entry,
		fafp.parameters as fetch_params
	`)
	fetchedTxQ = fetchedTxQ.From("fetched_activity_entries fae").
		LeftJoin(`fetched_activity_fetch_parameters fafp ON fae.fetch_parameters_id = fafp.id`)
	fetchedTxCond := sq.And{}
	fetchedTxCond = append(fetchedTxCond, sq.Or{
		sq.Eq{"fae.chain_id_out": chainIDs},
		sq.Eq{"fae.chain_id_in": chainIDs},
	})
	fetchedTxCond = append(fetchedTxCond, sq.Eq{"fafp.address": addresses})

	// Subquery to check if the transaction is already in the sent_transactions table
	distinctTxCond := sq.And{}
	distinctTxCond = append(distinctTxCond, sq.Expr("st.tx_hash = fae.tx_hash"))
	distinctTxCond = append(distinctTxCond, sq.Expr("st.chain_id = COALESCE(fae.chain_id_out, fae.chain_id_in)"))
	distinctTxSubQ := sq.Select("1").
		Prefix("NOT EXISTS (").
		From("sent_transactions st").
		Where(distinctTxCond).
		Suffix(")")

	fetchedTxCond = append(fetchedTxCond, distinctTxSubQ)
	fetchedTxQ = fetchedTxQ.Where(fetchedTxCond)

	fetchedTxQuery, fetchedTxArgs, err := fetchedTxQ.ToSql()
	if err != nil {
		return nil, err
	}
	fmt.Println("fetchedTxQuery")
	fmt.Println(fetchedTxQuery)
	fmt.Println(fetchedTxArgs)

	// Merge the two queries:
	// - distinct by tx hash
	// - prioritize sent transactions
	//q := sentTxQ.SuffixExpr(fetchedTxQ.Prefix("UNION ALL"))

	// Due to non-native support for UNION/UNION ALL, these need to be added ub a bit of
	// a hacky way
	order := "timestamp DESC, sent_is_approval ASC"
	q := sentTxQ.Suffix(" UNION "+fetchedTxQuery+" ORDER BY "+order+" LIMIT "+strconv.Itoa(limit)+" OFFSET "+strconv.Itoa(offset), fetchedTxArgs...)

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	fmt.Println("Final query")
	fmt.Println(query)
	fmt.Println(args)

	stmt, err := deps.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data, err := rowsToDataV2(rows)
	if err != nil {
		return nil, err
	}

	return dataToEntriesV2(deps, data, addresses)
}

type sentEntryDataV2 struct {
	TxArgs           *wallettypes.SendTxArgs
	Tx               *ethTypes.Transaction
	IsApproval       bool
	Status           ac.TxStatus
	Timestamp        int64
	Path             *routes.Path
	RouteInputParams *requests.RouteInputParams
}

func newSentEntryDataV2() *sentEntryDataV2 {
	return &sentEntryDataV2{
		TxArgs:           new(wallettypes.SendTxArgs),
		Tx:               new(ethTypes.Transaction),
		Path:             new(routes.Path),
		RouteInputParams: new(requests.RouteInputParams),
	}
}

type fetchedEntryDataV2 struct {
	Status       ac.TxStatus
	Timestamp    int64
	FetchedEntry *thirdparty.ActivityEntry
	FetchParams  *thirdparty.ActivityFetchParameters
}

func newFetchedEntryDataV2() *fetchedEntryDataV2 {
	return &fetchedEntryDataV2{
		FetchedEntry: new(thirdparty.ActivityEntry),
		FetchParams:  new(thirdparty.ActivityFetchParameters),
	}
}

type entryDataV2 struct {
	Type             EntryType
	SentEntryData    *sentEntryDataV2
	FetchedEntryData *fetchedEntryDataV2
}

func rowsToDataV2(rows *sql.Rows) ([]*entryDataV2, error) {
	var ret []*entryDataV2
	for rows.Next() {
		data := &entryDataV2{}
		sentData := newSentEntryDataV2()
		fetchedData := newFetchedEntryDataV2()

		var nullableEntryType sql.NullString
		nullableTx := sqlite.JSONBlob{Data: sentData.Tx}
		nullableTxArgs := sqlite.JSONBlob{Data: sentData.TxArgs}
		nullableIsApproval := sql.NullBool{}
		nullablePath := sqlite.JSONBlob{Data: sentData.Path}
		nullableRouteInputParams := sqlite.JSONBlob{Data: sentData.RouteInputParams}
		nullableStatus := sql.NullString{}
		nullableTimestamp := sql.NullInt64{}
		nullableFetchedEntry := sqlite.JSONBlob{Data: fetchedData.FetchedEntry}
		nullableFetchParams := sqlite.JSONBlob{Data: fetchedData.FetchParams}

		err := rows.Scan(
			&nullableEntryType,
			&nullableTx,
			&nullableTxArgs,
			&nullableIsApproval,
			&nullablePath,
			&nullableRouteInputParams,
			&nullableStatus,
			&nullableTimestamp,
			&nullableFetchedEntry,
			&nullableFetchParams,
		)
		if err != nil {
			return nil, err
		}

		// Check all necessary fields are not null
		if !nullableEntryType.Valid {
			logutils.ZapLogger().Warn("entry type missing in entryData")
			continue
		}
		data.Type = EntryType(nullableEntryType.String)

		switch data.Type {
		case EntryTypeSentTransaction:
			if !nullableTxArgs.Valid ||
				!nullableTx.Valid ||
				!nullableIsApproval.Valid ||
				!nullableStatus.Valid ||
				!nullableTimestamp.Valid ||
				!nullablePath.Valid ||
				!nullableRouteInputParams.Valid {
				logutils.ZapLogger().Warn("some fields missing in sent transaction entryData")
				continue
			}
			sentData.Status = nullableStatus.String
			sentData.Timestamp = nullableTimestamp.Int64
			sentData.IsApproval = nullableIsApproval.Bool

			data.SentEntryData = sentData

		case EntryTypeFetchedTransaction:
			if !nullableFetchedEntry.Valid ||
				!nullableFetchParams.Valid ||
				!nullableStatus.Valid ||
				!nullableTimestamp.Valid {
				logutils.ZapLogger().Warn("some fields missing in fetched transaction entryData")
				continue
			}
			fetchedData.Status = nullableStatus.String
			fetchedData.Timestamp = nullableTimestamp.Int64
			data.FetchedEntryData = fetchedData
		}

		ret = append(ret, data)
	}

	return ret, nil
}

func dataToEntriesV2(deps FilterDependencies, data []*entryDataV2, addresses []eth.Address) ([]Entry, error) {
	var ret []Entry

	now := time.Now().Unix()

	for _, data := range data {
		switch data.Type {
		case EntryTypeSentTransaction:
			d := data.SentEntryData
			chainID := wCommon.ChainID(d.Path.FromChain.ChainID)

			entry := Entry{
				payloadType: ac.MultiTransactionPT, // Temporary, to keep compatibility with clients
				id:          d.TxArgs.MultiTransactionID,
				transactions: []*ac.TransactionIdentity{
					{
						ChainID: chainID,
						Hash:    d.Tx.Hash(),
						Address: d.RouteInputParams.AddrFrom,
					},
				},
				timestamp:      d.Timestamp,
				activityType:   getActivityTypeV2(d.Path.ProcessorName, d.IsApproval),
				activityStatus: getActivityStatusV2(d.Status, d.Timestamp, now, getFinalizationPeriod(chainID)),
				amountOut:      d.Path.AmountIn,  // Path and Activity have inverse perspective for amountIn and amountOut
				amountIn:       d.Path.AmountOut, // Path has the Tx perspective, Activity has the Account perspective
				tokenOut:       getToken(d.Path.FromToken, d.Path.ProcessorName),
				tokenIn:        getToken(d.Path.ToToken, d.Path.ProcessorName),
				sender:         &d.RouteInputParams.AddrFrom,
				recipient:      &d.RouteInputParams.AddrTo,
				transferType:   getTransferTypeFromSentTx(d.Path.FromToken, d.Path.ProcessorName),
				//contractAddress:  // TODO: Handle community contract deployment
				//communityID:
			}

			if entry.activityType == ac.SendAT {
				if !slices.Contains(addresses, *entry.sender) {
					entry.activityType = ac.ReceiveAT
				}
			}

			if d.Path.FromChain != nil {
				chainID := wCommon.ChainID(d.Path.FromChain.ChainID)
				entry.chainIDOut = &chainID
			}
			if d.Path.ToChain != nil {
				chainID := wCommon.ChainID(d.Path.ToChain.ChainID)
				entry.chainIDIn = &chainID
			}

			entry.symbolOut, entry.symbolIn = lookupAndFillInTokens(deps, entry.tokenOut, entry.tokenIn)

			if entry.transferType == nil || ac.TokenType(*entry.transferType) != ac.Native {
				var interactedAddress eth.Address
				if d.Tx.To() != nil {
					interactedAddress = eth.BytesToAddress(d.Tx.To().Bytes())
				}
				entry.interactedContractAddress = &interactedAddress
			}

			if entry.activityType == ac.ApproveAT {
				entry.approvalSpender = d.Path.ApprovalContractAddress
			}

			ret = append(ret, entry)
		case EntryTypeFetchedTransaction:
			d := data.FetchedEntryData
			fmt.Printf("=== Fetched EntryTypeFetchedTransaction: %v\n", data.FetchedEntryData.FetchedEntry)
			uChainID := wCommon.UnknownChainID
			var chainIDOut *wCommon.ChainID
			var chainIDIn *wCommon.ChainID
			if d.FetchedEntry.ChainIDOut != nil {
				uChainID = *d.FetchedEntry.ChainIDOut
				chainIDOut = new(wCommon.ChainID)
				*chainIDOut = wCommon.ChainID(uChainID)
			} else if d.FetchedEntry.ChainIDIn != nil {
				uChainID = *d.FetchedEntry.ChainIDIn
				chainIDIn = new(wCommon.ChainID)
				*chainIDIn = wCommon.ChainID(uChainID)
			}
			chainID := wCommon.ChainID(uChainID)

			entry := Entry{
				payloadType: ac.MultiTransactionPT, // Temporary, to keep compatibility with clients
				id:          0,
				transactions: []*ac.TransactionIdentity{
					{
						ChainID: chainID,
						Hash:    d.FetchedEntry.TxHash,
						Address: d.FetchParams.Address,
					},
				},
				timestamp:                 d.Timestamp,
				activityType:              d.FetchedEntry.ActivityType,
				activityStatus:            getActivityStatusV2(d.Status, d.Timestamp, now, getFinalizationPeriod(chainID)),
				amountOut:                 d.FetchedEntry.AmountOut,
				amountIn:                  d.FetchedEntry.AmountIn,
				tokenOut:                  d.FetchedEntry.TokenOut,
				tokenIn:                   d.FetchedEntry.TokenIn,
				sender:                    &d.FetchedEntry.Sender,
				recipient:                 d.FetchedEntry.Recipient,
				chainIDOut:                chainIDOut,
				chainIDIn:                 chainIDIn,
				transferType:              getTransferTypeFromFetchedTx(d.FetchedEntry.TokenIn, d.FetchedEntry.TokenOut),
				interactedContractAddress: d.FetchedEntry.ContractAddress,
			}

			entry.symbolOut, entry.symbolIn = lookupAndFillInTokens(deps, entry.tokenOut, entry.tokenIn)
			fmt.Printf("=== Returned Entry: %v\n", entry)

			ret = append(ret, entry)
		}
	}

	return ret, nil
}

func getActivityTypeV2(processorName string, isApproval bool) ac.Type {
	if isApproval {
		return ac.ApproveAT
	}

	switch processorName {
	case pathProcessorCommon.ProcessorTransferName, pathProcessorCommon.ProcessorERC721Name, pathProcessorCommon.ProcessorERC1155Name:
		return ac.SendAT
	case pathProcessorCommon.ProcessorBridgeHopName, pathProcessorCommon.ProcessorBridgeCelerName:
		return ac.BridgeAT
	case pathProcessorCommon.ProcessorSwapParaswapName:
		return ac.SwapAT
	}
	return ac.UnknownAT
}

func getActivityStatusV2(status ac.TxStatus, timestamp int64, now int64, finalizationDuration int64) ac.Status {
	switch status {
	case ac.Pending:
		return ac.PendingAS
	case ac.Success:
		if timestamp+finalizationDuration > now {
			return ac.FinalizedAS
		}
		return ac.CompleteAS
	case ac.Failed:
		return ac.FailedAS
	}

	logutils.ZapLogger().Error("unhandled transaction status value")
	return ac.FailedAS
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

func getTransferTypeFromSentTx(fromToken *tokenTypes.Token, processorName string) *ac.TransferType {
	ret := new(ac.TransferType)

	switch processorName {
	case pathProcessorCommon.ProcessorTransferName:
		if fromToken.IsNative() {
			*ret = ac.TransferTypeEth
			break
		}
		*ret = ac.TransferTypeErc20
	case pathProcessorCommon.ProcessorERC721Name:
		*ret = ac.TransferTypeErc721
	case pathProcessorCommon.ProcessorERC1155Name:
		*ret = ac.TransferTypeErc1155
	default:
		ret = nil
	}

	return ret
}

func getTransferTypeFromFetchedTx(tokenIn, tokenOut *ac.Token) *ac.TransferType {
	ret := new(ac.TransferType)

	var token *ac.Token
	if tokenIn != nil {
		token = tokenIn
	} else if tokenOut != nil {
		token = tokenOut
	} else {
		return nil
	}

	switch token.TokenType {
	case ac.Erc20:
		*ret = ac.TransferTypeErc20
	case ac.Erc721:
		*ret = ac.TransferTypeErc721
	case ac.Erc1155:
		*ret = ac.TransferTypeErc1155
	default:
		*ret = ac.TransferTypeEth
	}

	return ret
}

func getToken(token *tokenTypes.Token, processorName string) *ac.Token {
	if token == nil {
		return nil
	}

	ret := new(ac.Token)
	ret.ChainID = wCommon.ChainID(token.ChainID)
	if token.IsNative() {
		ret.TokenType = ac.Native
	} else {
		ret.Address = token.Address
		switch processorName {
		case pathProcessorCommon.ProcessorERC721Name, pathProcessorCommon.ProcessorERC1155Name:
			id, err := wCommon.GetTokenIdFromSymbol(token.Symbol)
			if err != nil {
				logutils.ZapLogger().Warn("malformed token symbol", zap.Error(err))
				return nil
			}
			ret.TokenID = (*hexutil.Big)(id)
			if processorName == pathProcessorCommon.ProcessorERC721Name {
				ret.TokenType = ac.Erc721
			} else {
				ret.TokenType = ac.Erc1155
			}
		default:
			ret.TokenType = ac.Erc20
		}
	}
	return ret
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
