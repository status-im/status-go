package metrics

import (
	"context"
	"fmt"

	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
)

var (
	WakuVersion         = stats.Int64("waku_version", "", stats.UnitDimensionless)
	Messages            = stats.Int64("node_messages", "Number of messages received", stats.UnitDimensionless)
	Peers               = stats.Int64("peers", "Number of connected peers", stats.UnitDimensionless)
	Dials               = stats.Int64("dials", "Number of peer dials", stats.UnitDimensionless)
	StoreMessages       = stats.Int64("store_messages", "Number of historical messages", stats.UnitDimensionless)
	FilterSubscriptions = stats.Int64("filter_subscriptions", "Number of filter subscriptions", stats.UnitDimensionless)
	StoreErrors         = stats.Int64("errors", "Number of errors in store protocol", stats.UnitDimensionless)
	StoreQueries        = stats.Int64("store_queries", "Number of store queries", stats.UnitDimensionless)
	LightpushErrors     = stats.Int64("errors", "Number of errors in lightpush protocol", stats.UnitDimensionless)
	PeerExchangeError   = stats.Int64("errors", "Number of errors in peer exchange protocol", stats.UnitDimensionless)
)

var (
	KeyType, _    = tag.NewKey("type")
	ErrorType, _  = tag.NewKey("error_type")
	GitVersion, _ = tag.NewKey("git_version")
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
	StoreQueriesView = &view.View{
		Name:        "gowaku_store_queries",
		Measure:     StoreQueries,
		Description: "The number of the store queries received",
		Aggregation: view.Count(),
	}
	StoreMessagesView = &view.View{
		Name:        "gowaku_store_messages",
		Measure:     StoreMessages,
		Description: "The distribution of the store protocol messages",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyType},
	}
	FilterSubscriptionsView = &view.View{
		Name:        "gowaku_filter_subscriptions",
		Measure:     FilterSubscriptions,
		Description: "The number of content filter subscriptions",
		Aggregation: view.LastValue(),
	}
	StoreErrorTypesView = &view.View{
		Name:        "gowaku_store_errors",
		Measure:     StoreErrors,
		Description: "The distribution of the store protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
	LightpushErrorTypesView = &view.View{
		Name:        "gowaku_lightpush_errors",
		Measure:     LightpushErrors,
		Description: "The distribution of the lightpush protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
	VersionView = &view.View{
		Name:        "gowaku_version",
		Measure:     WakuVersion,
		Description: "The gowaku version",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{GitVersion},
	}
)

func recordWithTags(ctx context.Context, tagKey tag.Key, tagType string, ms stats.Measurement) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(tagKey, tagType)}, ms); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordLightpushError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, LightpushErrors.M(1))
}

func RecordPeerExchangeError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, PeerExchangeError.M(1))
}

func RecordMessage(ctx context.Context, tagType string, len int) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, StoreMessages.M(int64(len))); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordStoreQuery(ctx context.Context) {
	stats.Record(ctx, StoreQueries.M(1))
}

func RecordStoreError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, StoreErrors.M(1))
}

func RecordVersion(ctx context.Context, version string, commit string) {
	v := fmt.Sprintf("%s-%s", version, commit)
	recordWithTags(ctx, GitVersion, v, WakuVersion.M(1))
}
