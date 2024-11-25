package activity

import (
	"context"
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"

	eth "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	wCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/requests"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/token"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/sqlite"
	"github.com/status-im/status-go/transactions"
)

// getActivityEntriesV2 queries the route_* and tracked_transactions based on filter parameters and arguments
// it returns metadata for all entries ordered by timestamp column
func getActivityEntriesV2(ctx context.Context, deps FilterDependencies, addresses []eth.Address, allAddresses bool, chainIDs []wCommon.ChainID, filter Filter, offset int, limit int) ([]Entry, error) {
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
		rbtp.route_build_tx_params_json,
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
		LeftJoin(`route_build_tx_parameters rbtp ON 
			rpt.uuid = rbtp.uuid`).
		LeftJoin(`route_input_parameters rip ON 
			rpt.uuid = rip.uuid`)
	q = q.OrderBy("tt.timestamp DESC")

	qConditions := sq.And{}

	qConditions = append(qConditions, sq.Eq{"rpt.chain_id": chainIDs})
	qConditions = append(qConditions, sq.Eq{"rip.from_address": addresses})

	q = q.Where(qConditions)

	if limit != NoLimit {
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

	data, err := rowsToDataV2(rows)
	if err != nil {
		return nil, err
	}

	return dataToEntriesV2(deps, data)
}

type entryDataV2 struct {
	TxArgs           *wallettypes.SendTxArgs
	Tx               *ethTypes.Transaction
	IsApproval       bool
	Status           transactions.TxStatus
	Timestamp        int64
	Path             *routes.Path
	RouteInputParams *requests.RouteInputParams
	BuildInputParams *requests.RouterBuildTransactionsParams
}

func newEntryDataV2() *entryDataV2 {
	return &entryDataV2{
		TxArgs:           new(wallettypes.SendTxArgs),
		Tx:               new(ethTypes.Transaction),
		Path:             new(routes.Path),
		RouteInputParams: new(requests.RouteInputParams),
		BuildInputParams: new(requests.RouterBuildTransactionsParams),
	}
}

func rowsToDataV2(rows *sql.Rows) ([]*entryDataV2, error) {
	var ret []*entryDataV2
	for rows.Next() {
		data := newEntryDataV2()

		nullableTx := sqlite.JSONBlob{Data: data.Tx}
		nullableTxArgs := sqlite.JSONBlob{Data: data.TxArgs}
		nullableIsApproval := sql.NullBool{}
		nullablePath := sqlite.JSONBlob{Data: data.Path}
		nullableRouteInputParams := sqlite.JSONBlob{Data: data.RouteInputParams}
		nullableBuildInputParams := sqlite.JSONBlob{Data: data.BuildInputParams}
		nullableStatus := sql.NullString{}
		nullableTimestamp := sql.NullInt64{}

		err := rows.Scan(
			&nullableTx,
			&nullableTxArgs,
			&nullableIsApproval,
			&nullablePath,
			&nullableRouteInputParams,
			&nullableBuildInputParams,
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
			!nullableRouteInputParams.Valid ||
			!nullableBuildInputParams.Valid {
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

func dataToEntriesV2(deps FilterDependencies, data []*entryDataV2) ([]Entry, error) {
	var ret []Entry

	now := time.Now().Unix()

	for _, d := range data {
		chainID := wCommon.ChainID(d.Path.FromChain.ChainID)

		entry := Entry{
			payloadType: MultiTransactionPT, // Temporary, to keep compatibility with clients
			id:          d.TxArgs.MultiTransactionID,
			transactions: []*transfer.TransactionIdentity{
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
			transferType:   getTransferType(d.Path.FromToken, d.Path.ProcessorName),
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

		if entry.transferType == nil || TokenType(*entry.transferType) != Native {
			interactedAddress := eth.BytesToAddress(d.Tx.To().Bytes())
			entry.interactedContractAddress = &interactedAddress
		}

		if entry.activityType == ApproveAT {
			entry.approvalSpender = d.Path.ApprovalContractAddress
		}

		ret = append(ret, entry)
	}

	return ret, nil
}

func getActivityTypeV2(processorName string, isApproval bool) Type {
	if isApproval {
		return ApproveAT
	}

	switch processorName {
	case pathProcessorCommon.ProcessorTransferName, pathProcessorCommon.ProcessorERC721Name, pathProcessorCommon.ProcessorERC1155Name:
		return SendAT
	case pathProcessorCommon.ProcessorBridgeHopName, pathProcessorCommon.ProcessorBridgeCelerName:
		return BridgeAT
	case pathProcessorCommon.ProcessorSwapParaswapName:
		return SwapAT
	}
	return UnknownAT
}

func getActivityStatusV2(status transactions.TxStatus, timestamp int64, now int64, finalizationDuration int64) Status {
	switch status {
	case transactions.Pending:
		return PendingAS
	case transactions.Success:
		if timestamp+finalizationDuration > now {
			return FinalizedAS
		}
		return CompleteAS
	case transactions.Failed:
		return FailedAS
	}

	logutils.ZapLogger().Error("unhandled transaction status value")
	return FailedAS
}

func getFinalizationPeriod(chainID wCommon.ChainID) int64 {
	switch uint64(chainID) {
	case wCommon.EthereumMainnet, wCommon.EthereumSepolia:
		return L1FinalizationDuration
	}

	return L2FinalizationDuration
}

func getTransferType(fromToken *token.Token, processorName string) *TransferType {
	ret := new(TransferType)

	switch processorName {
	case pathProcessorCommon.ProcessorTransferName:
		if fromToken.IsNative() {
			*ret = TransferTypeEth
			break
		}
		*ret = TransferTypeErc20
	case pathProcessorCommon.ProcessorERC721Name:
		*ret = TransferTypeErc721
	case pathProcessorCommon.ProcessorERC1155Name:
		*ret = TransferTypeErc1155
	default:
		ret = nil
	}

	return ret
}

func getToken(token *token.Token, processorName string) *Token {
	if token == nil {
		return nil
	}

	ret := new(Token)
	ret.ChainID = wCommon.ChainID(token.ChainID)
	if token.IsNative() {
		ret.TokenType = Native
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
				ret.TokenType = Erc721
			} else {
				ret.TokenType = Erc1155
			}
		default:
			ret.TokenType = Erc20
		}
	}
	return ret
}
