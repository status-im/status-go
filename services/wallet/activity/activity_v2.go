package activity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/logutils"
	ac "github.com/status-im/status-go/services/wallet/activity/common"
	wCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/requests"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/thirdparty/activity/alchemy"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/sqlite"
	"go.uber.org/zap"
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

// getSentActivitiesEntries queries the route_* and tracked_transactions based on filter parameters and arguments
// it returns metadata for all entries ordered by timestamp column
func getSentActivitiesEntries(ctx context.Context, deps FilterDependencies, addresses []eth.Address, chainIDs []wCommon.ChainID, offset int, limit int) ([]Entry, error) {
	if len(addresses) == 0 {
		return nil, ErrNoAddressesProvided
	}
	if len(chainIDs) == 0 {
		return nil, ErrNoChainIDsProvided
	}

	q := sq.Select(`
		st.tx_json,
		rpt.tx_args_json,
		rpt.is_approval,
		rp.path_json,
		rip.route_input_params_json,
		tt.tx_status,
		tt.timestamp
		`).Distinct()
	q = q.From("sent_transactions st").
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
	q = q.OrderBy("tt.timestamp DESC", "rpt.is_approval ASC")

	qConditions := sq.And{}

	qConditions = append(qConditions, sq.Eq{"rpt.chain_id": chainIDs})
	qConditions = append(qConditions, sq.Eq{"rip.from_address": addresses})

	q = q.Where(qConditions)

	if limit != ac.NoLimit {
		q = q.Limit(uint64(limit))
		q = q.Offset(uint64(offset))
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

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

	data, err := rowsToSentEntryData(rows)
	if err != nil {
		return nil, err
	}

	return sentEntryDataToEntriesV2(deps, data)
}

func getFetchedActivitiesEntries(ctx context.Context, deps FilterDependencies, addresses []eth.Address, chainIDs []wCommon.ChainID, offset int, limit int) ([]Entry, error) {
	if len(addresses) == 0 {
		return nil, ErrNoAddressesProvided
	}
	if len(chainIDs) == 0 {
		return nil, ErrNoChainIDsProvided
	}

	q := sq.Select("e.transfer").
		From("fetched_alchemy_transfers e").
		Where(sq.And{
			sq.Eq{"e.chain_id": chainIDs},
			sq.Eq{"e.address": addresses}})

	if limit != ac.NoLimit {
		q = q.Limit(uint64(limit))
		q = q.Offset(uint64(offset))
	}

	query, args, err := q.ToSql()
	if err != nil {
		return nil, err
	}

	stmt, err := deps.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alchemyTransfers, err := rowsToTransfers(rows)
	if err != nil {
		return nil, err
	}

	return alchemy.TransfersToWalletActivityEntries(alchemyTransfers, uint64(chainIDs[0]), addresses[0])
}

func rowsToTransfers(rows *sql.Rows) ([]alchemy.Transfer, error) {
	var transfers []alchemy.Transfer
	for rows.Next() {
		var transfer alchemy.Transfer
		var transferJSON = sqlite.ToJSONBlob(&transfer)
		err := rows.Scan(transferJSON)
		fmt.Println("Scanned json:", transferJSON)
		if err != nil {
			return nil, err
		}
		if !transferJSON.Valid {
			return nil, errors.New("invalid entry")
		}
		transfers = append(transfers, transfer)
	}
	return transfers, nil
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

func rowsToSentEntryData(rows *sql.Rows) ([]*sentEntryDataV2, error) {
	var ret []*sentEntryDataV2
	for rows.Next() {
		data := newSentEntryDataV2()

		nullableTx := sqlite.JSONBlob{Data: data.Tx}
		nullableTxArgs := sqlite.JSONBlob{Data: data.TxArgs}
		nullableIsApproval := sql.NullBool{}
		nullablePath := sqlite.JSONBlob{Data: data.Path}
		nullableRouteInputParams := sqlite.JSONBlob{Data: data.RouteInputParams}
		nullableStatus := sql.NullString{}
		nullableTimestamp := sql.NullInt64{}

		err := rows.Scan(
			&nullableTx,
			&nullableTxArgs,
			&nullableIsApproval,
			&nullablePath,
			&nullableRouteInputParams,
			&nullableStatus,
			&nullableTimestamp,
		)
		if err != nil {
			return nil, err
		}

		// Check all necessary fields are not null
		if !nullableTxArgs.Valid ||
			!nullableTx.Valid ||
			!nullableIsApproval.Valid ||
			!nullableStatus.Valid ||
			!nullableTimestamp.Valid ||
			!nullablePath.Valid ||
			!nullableRouteInputParams.Valid {
			logutils.ZapLogger().Warn("some fields missing in entryData")
			continue
		}

		data.IsApproval = nullableIsApproval.Bool
		data.Status = nullableStatus.String
		data.Timestamp = nullableTimestamp.Int64

		ret = append(ret, data)
	}

	return ret, nil
}

func sentEntryDataToEntriesV2(deps FilterDependencies, data []*sentEntryDataV2) ([]Entry, error) {
	var ret []Entry

	now := time.Now().Unix()

	for _, d := range data {
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
			activityType:   getSentActivityType(d.Path.ProcessorName, d.IsApproval),
			activityStatus: getSentActivityStatus(d.Status, d.Timestamp, now, getFinalizationPeriod(chainID)),
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
	}

	return ret, nil
}

func getSentActivityType(processorName string, isApproval bool) ac.Type {
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

func getSentActivityStatus(status ac.TxStatus, timestamp int64, now int64, finalizationDuration int64) ac.Status {
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
