package metrics

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	Messages            = stats.Int64("messages", "Number of messages received", stats.UnitDimensionless)
	StoreMessages       = stats.Int64("store_messages", "Number of historical messages", stats.UnitDimensionless)
	FilterSubscriptions = stats.Int64("filter_subscriptions", "Number of filter subscriptions", stats.UnitDimensionless)
	Errors              = stats.Int64("errors", "Number of errors", stats.UnitDimensionless)
)

var (
	KeyType, _           = tag.NewKey("type")
	KeyStoreErrorType, _ = tag.NewKey("store_error_type")
)

var (
	MessageTypeView = &view.View{
		Name:        "messages",
		Measure:     Messages,
		Description: "The distribution of the messages received",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyType},
	}
	StoreMessageTypeView = &view.View{
		Name:        "store_messages",
		Measure:     StoreMessages,
		Description: "The distribution of the store protocol messages",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyType},
	}
	FilterSubscriptionsView = &view.View{
		Name:        "filter_subscriptions",
		Measure:     FilterSubscriptions,
		Description: "The number of content filter subscriptions",
		Aggregation: view.LastValue(),
	}
	StoreErrorTypesView = &view.View{
		Name:        "store_errors",
		Measure:     Errors,
		Description: "The distribution of the store protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyType},
	}
)
