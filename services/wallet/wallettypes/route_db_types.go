package wallettypes

import (
	"github.com/status-im/status-go/services/wallet/requests"
)

// These structs oontain all route execution data
// that's stored to the DB
type RouteData struct {
	RouteInputParams *requests.RouteInputParams
	BuildInputParams *requests.RouterBuildTransactionsParams
	PathsData        []*RouterTransactionDetails
}

func NewRouteData(routeInputParams *requests.RouteInputParams,
	buildInputParams *requests.RouterBuildTransactionsParams,
	pathsData []*RouterTransactionDetails) *RouteData {
	return &RouteData{
		RouteInputParams: routeInputParams,
		BuildInputParams: buildInputParams,
		PathsData:        pathsData,
	}
}
