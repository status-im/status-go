package wallettypes

import (
	"github.com/status-im/status-go/services/wallet/requests"
)

// These structs oontain all route execution data
// that's stored to the DB
type RouteData struct {
	RouteInputParams *requests.RouteInputParams
	PathsData        []*RouterTransactionDetails
}

func NewRouteData(routeInputParams *requests.RouteInputParams,
	pathsData []*RouterTransactionDetails) *RouteData {
	return &RouteData{
		RouteInputParams: routeInputParams,
		PathsData:        pathsData,
	}
}
