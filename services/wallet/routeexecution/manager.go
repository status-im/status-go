package routeexecution

import (
	"context"
	"database/sql"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/event"

	status_common "github.com/status-im/status-go/common"
	statusErrors "github.com/status-im/status-go/internal/errors"
	"github.com/status-im/status-go/internal/logutils"
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/responses"
	"github.com/status-im/status-go/services/wallet/routeexecution/storage"
	"github.com/status-im/status-go/services/wallet/router"
	"github.com/status-im/status-go/services/wallet/router/pathprocessor"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/sendtype"
	"github.com/status-im/status-go/services/wallet/token"
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
	tokenManager       *token.Manager
	transactionManager *transfer.TransactionManager
	db                 *storage.DB
	eventFeed          *event.Feed
}

func NewManager(walletDB *sql.DB, eventFeed *event.Feed, router *router.Router, tokenManager *token.Manager, transactionManager *transfer.TransactionManager) *Manager {
	return &Manager{
		router:             router,
		tokenManager:       tokenManager,
		transactionManager: transactionManager,
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

		tokenFrom, _ := m.tokenManager.GetTokenByKey(routeInputParams.TokenKey)
		tokenTo, _ := m.tokenManager.GetTokenByKey(routeInputParams.ToTokenKey)

		// re-use path processor input params structure to pass extra params to transaction manager
		var extraParams pathprocessor.ProcessorInputParams
		extraParams, err = m.router.CreateProcessorInputParams(&routeInputParams, tokenFrom, tokenTo, 0)
		if err != nil {
			return
		}

		response.SendDetails.UpdateFields(routeInputParams, routeInputParams.FromChainID, routeInputParams.ToChainID)

		// notify client that sending transactions started (has 3 steps, building txs, signing txs, sending txs)
		signal.SendWalletEvent(signal.RouterSendingTransactionsStarted, response.SendDetails)

		var fromChainID, toChainID uint64
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

		_, routeInputParams = m.router.GetBestRouteAndAssociatedInputParams()
		if routeInputParams.Uuid != sendInputParams.Uuid {
			err = ErrCannotResolveRouteId
			return
		}

		response.SendDetails.UpdateFields(routeInputParams, routeInputParams.FromChainID, routeInputParams.ToChainID)

		fromChainID, toChainID, err := m.transactionManager.ValidateAndAddSignaturesToRouterTransactions(sendInputParams.Signatures)
		if err != nil {
			response.SendDetails.UpdateFields(routeInputParams, fromChainID, toChainID)
			return
		}

		response.SentTransactions, fromChainID, toChainID, err = m.transactionManager.SendRouterTransactions(ctx)
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
