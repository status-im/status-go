package routeexecution

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"

	status_common "github.com/status-im/status-go/common"
	statusErrors "github.com/status-im/status-go/errors"
	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/routeexecution/storage"
	"github.com/status-im/status-go/services/wallet/router"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/transfer"
	"github.com/status-im/status-go/services/wallet/walletevent"
	"github.com/status-im/status-go/services/wallet/wallettypes"
	"github.com/status-im/status-go/signal"
)

const (
	EventRouteExecutionTransactionSent walletevent.EventType = walletevent.InternalEventTypePrefix + "wallet-route-execution-transaction-sent"
)

type Manager struct {
	router             *router.Router
	transactionManager *transfer.TransactionManager
	transferController *transfer.Controller
	db                 *storage.DB
	eventFeed          *event.Feed
}

func NewManager(walletDB *sql.DB, eventFeed *event.Feed, router *router.Router, transactionManager *transfer.TransactionManager, transferController *transfer.Controller) *Manager {
	return &Manager{
		router:             router,
		transactionManager: transactionManager,
		transferController: transferController,
		db:                 storage.NewDB(walletDB),
		eventFeed:          eventFeed,
	}
}

func (m *Manager) ClearLocalRouteData() {
	m.transactionManager.ClearLocalRouterTransactionsData()
}

func (m *Manager) ReevaluateRouterPath(ctx context.Context, pathTxIdentity *requests.PathTxIdentity) error {
	return m.router.ReevaluateRouterPath(ctx, pathTxIdentity)
}

func (m *Manager) BuildTransactionsFromRoute(ctx context.Context, uuid string) {
	go func() {
		defer status_common.LogOnPanic()

		m.router.StopSuggestedRoutesAsyncCalculation()

		var err error
		response := &responses.RouterTransactionsForSigning{
			SendDetails: &responses.SendDetails{
				Uuid: uuid,
			},
		}

		defer func() {
			if err != nil {
				m.ClearLocalRouteData()
				err = statusErrors.CreateErrorResponseFromError(err)
				response.SendDetails.ErrorResponse = err.(*statusErrors.ErrorResponse)
			}
			signal.SendWalletEvent(signal.SignRouterTransactions, response)
		}()

		route, routeInputParams := m.router.GetBestRouteAndAssociatedInputParams()
		if routeInputParams.Uuid != uuid {
			// should never be here
			err = ErrCannotResolveRouteId
			return
		}

		// re-use path processor input params structure to pass extra params to transaction manager
		var extraParams pathprocessor.ProcessorInputParams
		extraParams, err = m.router.CreateProcessorInputParams(&routeInputParams, nil, nil, nil, nil, 0)
		if err != nil {
			return
		}

		fromChainID, toChainID := route.GetFirstPathChains()

		response.SendDetails.UpdateFields(routeInputParams, fromChainID, toChainID)

		// notify client that sending transactions started (has 3 steps, building txs, signing txs, sending txs)
		signal.SendWalletEvent(signal.RouterSendingTransactionsStarted, response.SendDetails)

		response.SigningDetails, fromChainID, toChainID, err = m.transactionManager.BuildTransactionsFromRoute(
			route,
			m.router.GetPathProcessors(),
			&extraParams,
		)
		if err != nil {
			response.SendDetails.UpdateFields(routeInputParams, fromChainID, toChainID)
		}
	}()
}

func (m *Manager) SendRouterTransactionsWithSignatures(ctx context.Context, sendInputParams *requests.RouterSendTransactionsParams) {
	go func() {
		defer status_common.LogOnPanic()

		var (
			err              error
			route            routes.Route
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
				if m.transactionManager.ApprovalRequiredForPath(pathProcessorCommon.ProcessorSwapParaswapName) &&
					m.transactionManager.ApprovalPlacedForPath(pathProcessorCommon.ProcessorSwapParaswapName) &&
					!m.transactionManager.TxPlacedForPath(pathProcessorCommon.ProcessorSwapParaswapName) {
					clearLocalData = false
				}
			}

			if clearLocalData {
				m.ClearLocalRouteData()
			}

			if err != nil {
				err = statusErrors.CreateErrorResponseFromError(err)
				response.SendDetails.ErrorResponse = err.(*statusErrors.ErrorResponse)
			}
			signal.SendWalletEvent(signal.RouterTransactionsSent, response)

			event := walletevent.Event{
				Type:        EventRouteExecutionTransactionSent,
				EventParams: response,
			}
			m.eventFeed.Send(event)
		}()

		route, routeInputParams = m.router.GetBestRouteAndAssociatedInputParams()
		if routeInputParams.Uuid != sendInputParams.Uuid {
			err = ErrCannotResolveRouteId
			return
		}

		fromChainID, toChainID := route.GetFirstPathChains()

		response.SendDetails.UpdateFields(routeInputParams, fromChainID, toChainID)

		fromChainID, toChainID, err = m.transactionManager.ValidateAndAddSignaturesToRouterTransactions(sendInputParams.Signatures)
		if err != nil {
			response.SendDetails.UpdateFields(routeInputParams, fromChainID, toChainID)
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

		response.SentTransactions, fromChainID, toChainID, err = m.transactionManager.SendRouterTransactions(ctx, multiTx)
		if err != nil {
			response.SendDetails.UpdateFields(routeInputParams, fromChainID, toChainID)
			logutils.ZapLogger().Error("Error sending router transactions", zap.Error(err))
			// TODO #16556: Handle partially successful Tx sends?
			// Don't return, store whichever transactions were successfully sent
		}

		// don't overwrite err since we want to process it in the deferred function
		var tmpErr error
		routerTransactions := m.transactionManager.GetRouterTransactions()
		routeData := wallettypes.NewRouteData(&routeInputParams, routerTransactions)
		tmpErr = m.db.PutRouteData(routeData)
		if tmpErr != nil {
			logutils.ZapLogger().Error("Error storing route data", zap.Error(tmpErr))
		}
	}()
}
