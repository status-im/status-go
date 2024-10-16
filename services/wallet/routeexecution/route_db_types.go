package routeexecution

import (
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/transactions"
)

// These structs oontain all route execution data
// that's stored to the DB
type RouteData struct {
	RouteInputParams *requests.RouteInputParams
	BuildInputParams *requests.RouterBuildTransactionsParams
	PathsData        []*PathData
}

type PathData struct {
	Path             *routes.Path
	TransactionsData []*TransactionData
}

type TransactionData struct {
	ChainID    uint64
	TxHash     types.Hash
	IsApproval bool
	TxArgs     *transactions.SendTxArgs
	Tx         *ethTypes.Transaction
}

func NewRouteData(routeInputParams *requests.RouteInputParams,
	buildInputParams *requests.RouterBuildTransactionsParams,
	transactionDetails []*transfer.RouterTransactionDetails) *RouteData {

	pathDataPerProcessorName := make(map[string]*PathData)
	pathsData := make([]*PathData, 0, len(transactionDetails))
	for _, td := range transactionDetails {
		transactionsData := make([]*TransactionData, 0, 2)
		if td.IsApprovalPlaced() {
			transactionsData = append(transactionsData, &TransactionData{
				ChainID:    td.RouterPath.FromChain.ChainID,
				TxHash:     td.ApprovalTxSentHash,
				IsApproval: true,
				TxArgs:     td.ApprovalTxArgs,
				Tx:         td.ApprovalTx,
			})
		}
		if td.IsTxPlaced() {
			transactionsData = append(transactionsData, &TransactionData{
				ChainID:    td.RouterPath.FromChain.ChainID,
				TxHash:     td.TxSentHash,
				IsApproval: false,
				TxArgs:     td.TxArgs,
				Tx:         td.Tx,
			})
		}

		var pathData *PathData
		var ok bool
		if pathData, ok = pathDataPerProcessorName[td.RouterPath.ProcessorName]; !ok {
			pathData = &PathData{
				Path:             td.RouterPath,
				TransactionsData: make([]*TransactionData, 0, 2),
			}
			pathsData = append(pathsData, pathData)
			pathDataPerProcessorName[td.RouterPath.ProcessorName] = pathData
		}
		pathData.TransactionsData = append(pathData.TransactionsData, transactionsData...)
	}

	return &RouteData{
		RouteInputParams: routeInputParams,
		BuildInputParams: buildInputParams,
		PathsData:        pathsData,
	}
}
