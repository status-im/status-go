package routeexecution

import (
	"github.com/status-im/status-go/services/wallet/requests"
	"github.com/status-im/status-go/services/wallet/transfer"
)

// These structs oontain all route execution data
// that's stored to the DB
type RouteData struct {
	RouteInputParams *requests.RouteInputParams
	BuildInputParams *requests.RouterBuildTransactionsParams
	PathsData        []*transfer.RouterTransactionDetails
}

func NewRouteData(routeInputParams *requests.RouteInputParams,
	buildInputParams *requests.RouterBuildTransactionsParams,
	pathsData []*transfer.RouterTransactionDetails) *RouteData {

	return &RouteData{
		RouteInputParams: routeInputParams,
		BuildInputParams: buildInputParams,
		PathsData:        pathsData,
	}
}
