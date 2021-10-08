package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	Messages            = stats.Int64("node_messages", "Number of messages received", stats.UnitDimensionless)
	Peers               = stats.Int64("peers", "Number of connected peers", stats.UnitDimensionless)
	Dials               = stats.Int64("dials", "Number of peer dials", stats.UnitDimensionless)
	StoreMessages       = stats.Int64("store_messages", "Number of historical messages", stats.UnitDimensionless)
	FilterSubscriptions = stats.Int64("filter_subscriptions", "Number of filter subscriptions", stats.UnitDimensionless)
	Errors              = stats.Int64("errors", "Number of errors", stats.UnitDimensionless)
)

var (
	KeyType, _           = tag.NewKey("type")
	KeyStoreErrorType, _ = tag.NewKey("store_error_type")
)

var (
	PeersView = &view.View{
		Name:        "gowaku_connected_peers",
		Measure:     Peers,
		Description: "Number of connected peers",
		Aggregation: view.Sum(),
	}
	DialsView = &view.View{
		Name:        "gowaku_peers_dials",
		Measure:     Dials,
		Description: "Number of peer dials",
		Aggregation: view.Count(),
	}
	MessageView = &view.View{
		Name:        "gowaku_node_messages",
		Measure:     Messages,
		Description: "The number of the messages received",
		Aggregation: view.Count(),
	}
	StoreMessagesView = &view.View{
		Name:        "gowaku_store_messages",
		Measure:     StoreMessages,
		Description: "The distribution of the store protocol messages",
		Aggregation: view.LastValue(),
	}
	FilterSubscriptionsView = &view.View{
		Name:        "gowaku_filter_subscriptions",
		Measure:     FilterSubscriptions,
		Description: "The number of content filter subscriptions",
		Aggregation: view.LastValue(),
	}
	StoreErrorTypesView = &view.View{
		Name:        "gowaku_store_errors",
		Measure:     Errors,
		Description: "The distribution of the store protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyType},
	}
)
