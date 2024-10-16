package routeexecution_test

import (
	"testing"

	"github.com/status-im/status-go/services/wallet/routeexecution"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"

	"github.com/stretchr/testify/require"
)

func Test_PutRouteData(t *testing.T) {
	testData := []dbTestData{
		createDBTestData("USDTSwapApprove", getUSDTSwapApproveDBTestData(), getUSDTSwapTxDBTestData()), // After placing the Swap Tx, we expect to get info for both txs
		createDBTestData("USDTSwapTx", getUSDTSwapTxDBTestData(), getUSDTSwapTxDBTestData()),
		createDBTestData("ETHSwapTx", getETHSwapTxDBTestData(), getETHSwapTxDBTestData()),
		createDBTestData("ETHBridgeTx", getETHBridgeTxDBTestData(), getETHBridgeTxDBTestData()),
		createDBTestData("USDTSendTx", getUSDTSendTxDBTestData(), getUSDTSendTxDBTestData()),
	}

	walletDB, closeFn, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "routeexecution-tests")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, closeFn())
	}()

	routeDB := routeexecution.NewDB(walletDB)

	for _, tt := range testData {
		t.Run("Put_"+tt.name, func(t *testing.T) {
			insertedParams := tt.insertedParams
			routeData := routeexecution.NewRouteData(&insertedParams.routeInputParams, insertedParams.buildInputParams, insertedParams.transactionDetails)
			err := routeDB.PutRouteData(routeData)
			require.NoError(t, err)
		})
	}

	for _, tt := range testData {
		t.Run("Get_"+tt.name, func(t *testing.T) {
			expectedParams := tt.expectedParams
			routeData := routeexecution.NewRouteData(&expectedParams.routeInputParams, expectedParams.buildInputParams, expectedParams.transactionDetails)
			readRouteData, err := routeDB.GetRouteData(routeData.RouteInputParams.Uuid)
			require.NoError(t, err)
			require.EqualExportedValues(t, routeData, readRouteData)
		})
	}
}
