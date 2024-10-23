package routeexecution_test

import (
	"testing"

	"github.com/status-im/status-go/services/wallet/routeexecution"
	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
	"github.com/stretchr/testify/require"
)

func Test_PutRouteData(t *testing.T) {
	testData := getDBTestData()

	walletDB, closeFn, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "routeexecution-tests")
	require.NoError(t, err)
	defer closeFn()

	routeDB := routeexecution.NewDB(walletDB)

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			routeData := routeexecution.NewRouteData(&tt.routeInputParams, tt.buildInputParams, tt.transactionDetails)
			err := routeDB.PutRouteData(routeData)
			require.NoError(t, err)

			readRouteData, err := routeDB.GetRouteData(routeData.RouteInputParams.Uuid)
			require.NoError(t, err)
			require.EqualExportedValues(t, routeData, readRouteData)
		})
	}
}
