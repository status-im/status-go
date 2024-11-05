package routeexecution

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/logutils"

	status_common "github.com/status-im/status-go/common"
	statusErrors "github.com/status-im/status-go/errors"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/router"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/signal"
)

type Manager struct {
	router             *router.Router
	transactionManager *transfer.TransactionManager
	transferController *transfer.Controller
	db                 *DB

	// Local data used for storage purposes
	buildInputParams *requests.RouterBuildTransactionsParams
}

func NewManager(walletDB *sql.DB, router *router.Router, transactionManager *transfer.TransactionManager, transferController *transfer.Controller) *Manager {
	return &Manager{
		router:             router,
		transactionManager: transactionManager,
		transferController: transferController,
		db:                 NewDB(walletDB),
	}
}

func (m *Manager) clearLocalRouteData() {
	m.buildInputParams = nil
	m.transactionManager.ClearLocalRouterTransactionsData()
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
				m.clearLocalRouteData()
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

		m.buildInputParams = buildInputParams

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
				if m.transactionManager.ApprovalRequiredForPath(walletCommon.ProcessorSwapParaswapName) &&
					m.transactionManager.ApprovalPlacedForPath(walletCommon.ProcessorSwapParaswapName) &&
					!m.transactionManager.TxPlacedForPath(walletCommon.ProcessorSwapParaswapName) {
					clearLocalData = false
				}
			}

			if clearLocalData {
				m.clearLocalRouteData()
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
		if err != nil {
			log.Error("Error sending router transactions", "error", err)
			// TODO #16556: Handle partially successful Tx sends?
			// Don't return, store whichever transactions were successfully sent
		}

		// don't overwrite err since we want to process it in the deferred function
		var tmpErr error
		routerTransactions := m.transactionManager.GetRouterTransactions()
		routeData := NewRouteData(&routeInputParams, m.buildInputParams, routerTransactions)
		tmpErr = m.db.PutRouteData(routeData)
		if tmpErr != nil {
			log.Error("Error storing route data", "error", tmpErr)
		}

		var (
			chainIDs  []uint64
			addresses []common.Address
		)
		for _, tx := range response.SentTransactions {
			chainIDs = append(chainIDs, tx.FromChain)
			addresses = append(addresses, common.Address(tx.FromAddress))
			go func(chainId uint64, txHash common.Hash) {
				defer status_common.LogOnPanic()
				tmpErr = m.transactionManager.WatchTransaction(context.Background(), chainId, txHash)
				if tmpErr != nil {
					logutils.ZapLogger().Error("Error watching transaction", zap.Error(tmpErr))
					return
				}
			}(tx.FromChain, common.Hash(tx.Hash))
		}
		tmpErr = m.transferController.CheckRecentHistory(chainIDs, addresses)
		if tmpErr != nil {
			logutils.ZapLogger().Error("Error checking recent history", zap.Error(tmpErr))
		}
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
