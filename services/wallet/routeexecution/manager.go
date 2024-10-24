package routeexecution

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/status-im/status-go/eth-node/types"

	status_common "github.com/status-im/status-go/common"
	statusErrors "github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/signal"
)

type Manager struct {
	router             *router.Router
	transactionManager *transfer.TransactionManager
	transferController *transfer.Controller
}

func NewManager(router *router.Router, transactionManager *transfer.TransactionManager, transferController *transfer.Controller) *Manager {
	return &Manager{
		router:             router,
		transactionManager: transactionManager,
		transferController: transferController,
	}
}

func (m *Manager) BuildTransactionsFromRoute(ctx context.Context, buildInputParams *requests.RouterBuildTransactionsParams) {
	go func() {
		defer status_common.LogOnPanic()

		m.router.StopSuggestedRoutesAsyncCalculation()

		var err error
		response := &responses.RouterTransactionsForSigning{
			SendDetails: &responses.SendDetails{
				Uuid: buildInputParams.Uuid,
			},
		}

		defer func() {
			if err != nil {
				m.transactionManager.ClearLocalRouterTransactionsData()
				err = statusErrors.CreateErrorResponseFromError(err)
				response.SendDetails.ErrorResponse = err.(*statusErrors.ErrorResponse)
			}
			signal.SendWalletEvent(signal.SignRouterTransactions, response)
		}()

		route, routeInputParams := m.router.GetBestRouteAndAssociatedInputParams()
		if routeInputParams.Uuid != buildInputParams.Uuid {
			// should never be here
			err = ErrCannotResolveRouteId
			return
		}

		updateFields(response.SendDetails, routeInputParams)

		// notify client that sending transactions started (has 3 steps, building txs, signing txs, sending txs)
		signal.SendWalletEvent(signal.RouterSendingTransactionsStarted, response.SendDetails)

		response.SigningDetails, err = m.transactionManager.BuildTransactionsFromRoute(
			route,
			m.router.GetPathProcessors(),
			transfer.BuildRouteExtraParams{
				AddressFrom:        routeInputParams.AddrFrom,
				AddressTo:          routeInputParams.AddrTo,
				Username:           routeInputParams.Username,
				PublicKey:          routeInputParams.PublicKey,
				PackID:             routeInputParams.PackID.ToInt(),
				SlippagePercentage: buildInputParams.SlippagePercentage,
			},
		)
	}()
}

func (m *Manager) SendRouterTransactionsWithSignatures(ctx context.Context, sendInputParams *requests.RouterSendTransactionsParams) {
	go func() {
		defer status_common.LogOnPanic()

		var (
			err              error
			routeInputParams requests.RouteInputParams
		)
		response := &responses.RouterSentTransactions{
			SendDetails: &responses.SendDetails{
				Uuid: sendInputParams.Uuid,
			},
		}

		defer func() {
			clearLocalData := true
			if routeInputParams.SendType == sendtype.Swap {
				// in case of swap don't clear local data if an approval is placed, but swap tx is not sent yet
				if m.transactionManager.ApprovalRequiredForPath(pathprocessor.ProcessorSwapParaswapName) &&
					m.transactionManager.ApprovalPlacedForPath(pathprocessor.ProcessorSwapParaswapName) &&
					!m.transactionManager.TxPlacedForPath(pathprocessor.ProcessorSwapParaswapName) {
					clearLocalData = false
				}
			}

			if clearLocalData {
				m.transactionManager.ClearLocalRouterTransactionsData()
			}

			if err != nil {
				err = statusErrors.CreateErrorResponseFromError(err)
				response.SendDetails.ErrorResponse = err.(*statusErrors.ErrorResponse)
			}
			signal.SendWalletEvent(signal.RouterTransactionsSent, response)
		}()

		_, routeInputParams = m.router.GetBestRouteAndAssociatedInputParams()
		if routeInputParams.Uuid != sendInputParams.Uuid {
			err = ErrCannotResolveRouteId
			return
		}

		updateFields(response.SendDetails, routeInputParams)

		err = m.transactionManager.ValidateAndAddSignaturesToRouterTransactions(sendInputParams.Signatures)
		if err != nil {
			return
		}

		//////////////////////////////////////////////////////////////////////////////
		// prepare multitx
		var mtType transfer.MultiTransactionType = transfer.MultiTransactionSend
		if routeInputParams.SendType == sendtype.Bridge {
			mtType = transfer.MultiTransactionBridge
		} else if routeInputParams.SendType == sendtype.Swap {
			mtType = transfer.MultiTransactionSwap
		}

		multiTx := transfer.NewMultiTransaction(
			/* Timestamp:     */ uint64(time.Now().Unix()),
			/* FromNetworkID: */ 0,
			/* ToNetworkID:	  */ 0,
			/* FromTxHash:    */ common.Hash{},
			/* ToTxHash:      */ common.Hash{},
			/* FromAddress:   */ routeInputParams.AddrFrom,
			/* ToAddress:     */ routeInputParams.AddrTo,
			/* FromAsset:     */ routeInputParams.TokenID,
			/* ToAsset:       */ routeInputParams.ToTokenID,
			/* FromAmount:    */ routeInputParams.AmountIn,
			/* ToAmount:      */ routeInputParams.AmountOut,
			/* Type:		  */ mtType,
			/* CrossTxID:	  */ "",
		)

		_, err = m.transactionManager.InsertMultiTransaction(multiTx)
		if err != nil {
			return
		}
		//////////////////////////////////////////////////////////////////////////////

		response.SentTransactions, err = m.transactionManager.SendRouterTransactions(ctx, multiTx)

		var (
			chainIDs  []uint64
			addresses []common.Address
		)
		for _, tx := range response.SentTransactions {
			chainIDs = append(chainIDs, tx.FromChain)
			addresses = append(addresses, common.Address(tx.FromAddress))
			go func(chainId uint64, txHash common.Hash) {
				defer status_common.LogOnPanic()
				err = m.transactionManager.WatchTransaction(context.Background(), chainId, txHash)
				if err != nil {
					return
				}
			}(tx.FromChain, common.Hash(tx.Hash))
		}
		err = m.transferController.CheckRecentHistory(chainIDs, addresses)
	}()
}

func updateFields(sd *responses.SendDetails, inputParams requests.RouteInputParams) {
	sd.SendType = int(inputParams.SendType)
	sd.FromAddress = types.Address(inputParams.AddrFrom)
	sd.ToAddress = types.Address(inputParams.AddrTo)
	sd.FromToken = inputParams.TokenID
	sd.ToToken = inputParams.ToTokenID
	if inputParams.AmountIn != nil {
		sd.FromAmount = inputParams.AmountIn.String()
	}
	if inputParams.AmountOut != nil {
		sd.ToAmount = inputParams.AmountOut.String()
	}
	sd.OwnerTokenBeingSent = inputParams.TokenIDIsOwnerToken
	sd.Username = inputParams.Username
	sd.PublicKey = inputParams.PublicKey
	if inputParams.PackID != nil {
		sd.PackID = inputParams.PackID.String()
	}
}
