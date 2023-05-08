package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
)

var (
	WakuVersion = stats.Int64("waku_version", "", stats.UnitDimensionless)
	Messages    = stats.Int64("node_messages", "Number of messages received", stats.UnitDimensionless)
	MessageSize = stats.Int64("waku_histogram_message_size", "message size histogram in kB", stats.UnitDimensionless)

	Peers = stats.Int64("peers", "Number of connected peers", stats.UnitDimensionless)
	Dials = stats.Int64("dials", "Number of peer dials", stats.UnitDimensionless)

	LegacyFilterMessages      = stats.Int64("legacy_filter_messages", "Number of legacy filter messages", stats.UnitDimensionless)
	LegacyFilterSubscribers   = stats.Int64("legacy_filter_subscribers", "Number of legacy filter subscribers", stats.UnitDimensionless)
	LegacyFilterSubscriptions = stats.Int64("legacy_filter_subscriptions", "Number of legacy filter subscriptions", stats.UnitDimensionless)
	LegacyFilterErrors        = stats.Int64("legacy_filter_errors", "Number of errors in legacy filter protocol", stats.UnitDimensionless)

	FilterMessages                     = stats.Int64("filter_messages", "Number of filter messages", stats.UnitDimensionless)
	FilterRequests                     = stats.Int64("filter_requests", "Number of filter requests", stats.UnitDimensionless)
	FilterSubscriptions                = stats.Int64("filter_subscriptions", "Number of filter subscriptions", stats.UnitDimensionless)
	FilterErrors                       = stats.Int64("filter_errors", "Number of errors in filter protocol", stats.UnitDimensionless)
	FilterRequestDurationSeconds       = stats.Int64("filter_request_duration_seconds", "Duration of Filter Subscribe Requests", stats.UnitSeconds)
	FilterHandleMessageDurationSeconds = stats.Int64("filter_handle_msessageduration_seconds", "Duration to Push Message to Filter Subscribers", stats.UnitSeconds)

	StoreErrors  = stats.Int64("errors", "Number of errors in store protocol", stats.UnitDimensionless)
	StoreQueries = stats.Int64("store_queries", "Number of store queries", stats.UnitDimensionless)

	ArchiveMessages              = stats.Int64("waku_archive_messages", "Number of historical messages", stats.UnitDimensionless)
	ArchiveErrors                = stats.Int64("waku_archive_errors", "Number of errors in archive protocol", stats.UnitDimensionless)
	ArchiveInsertDurationSeconds = stats.Int64("waku_archive_insert_duration_seconds", "Message insertion duration", stats.UnitSeconds)
	ArchiveQueryDurationSeconds  = stats.Int64("waku_archive_query_duration_seconds", "History query duration", stats.UnitSeconds)

	LightpushMessages = stats.Int64("lightpush_messages", "Number of messages sent via lightpush protocol", stats.UnitDimensionless)
	LightpushErrors   = stats.Int64("errors", "Number of errors in lightpush protocol", stats.UnitDimensionless)

	PeerExchangeError = stats.Int64("errors", "Number of errors in peer exchange protocol", stats.UnitDimensionless)

	DnsDiscoveryNodes  = stats.Int64("dnsdisc_nodes", "Number of discovered nodes in dns discovert", stats.UnitDimensionless)
	DnsDiscoveryErrors = stats.Int64("dnsdisc_errors", "Number of errors in dns discovery", stats.UnitDimensionless)

	DiscV5Errors = stats.Int64("discv5_errors", "Number of errors in discv5", stats.UnitDimensionless)
)

var (
	KeyType, _    = tag.NewKey("type")
	ErrorType, _  = tag.NewKey("error_type")
	GitVersion, _ = tag.NewKey("git_version")
)

var (
	PeersView = &view.View{
		Name:        "waku_connected_peers",
		Measure:     Peers,
		Description: "Number of connected peers",
		Aggregation: view.Sum(),
	}
	DialsView = &view.View{
		Name:        "waku_peers_dials",
		Measure:     Dials,
		Description: "Number of peer dials",
		Aggregation: view.Count(),
	}
	MessageView = &view.View{
		Name:        "waku_node_messages",
		Measure:     Messages,
		Description: "The number of the messages received",
		Aggregation: view.Count(),
	}
	MessageSizeView = &view.View{
		Name:        "waku_histogram_message_size",
		Measure:     MessageSize,
		Description: "message size histogram in kB",
		Aggregation: view.Distribution(0.0, 5.0, 15.0, 50.0, 100.0, 300.0, 700.0, 1000.0),
	}

	StoreQueriesView = &view.View{
		Name:        "waku_store_queries",
		Measure:     StoreQueries,
		Description: "The number of the store queries received",
		Aggregation: view.Count(),
	}
	StoreErrorTypesView = &view.View{
		Name:        "waku_store_errors",
		Measure:     StoreErrors,
		Description: "The distribution of the store protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}

	ArchiveMessagesView = &view.View{
		Name:        "waku_archive_messages",
		Measure:     ArchiveMessages,
		Description: "The distribution of the archive protocol messages",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyType},
	}
	ArchiveErrorTypesView = &view.View{
		Name:        "waku_archive_errors",
		Measure:     StoreErrors,
		Description: "Number of errors in archive protocol",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
	ArchiveInsertDurationView = &view.View{
		Name:        "waku_archive_insert_duration_seconds",
		Measure:     ArchiveInsertDurationSeconds,
		Description: "Message insertion duration",
		Aggregation: view.Count(),
	}
	ArchiveQueryDurationView = &view.View{
		Name:        "waku_archive_query_duration_seconds",
		Measure:     ArchiveQueryDurationSeconds,
		Description: "History query duration",
		Aggregation: view.Count(),
	}

	LegacyFilterSubscriptionsView = &view.View{
		Name:        "waku_legacy_filter_subscriptions",
		Measure:     LegacyFilterSubscriptions,
		Description: "The number of legacy filter subscriptions",
		Aggregation: view.Count(),
	}
	LegacyFilterSubscribersView = &view.View{
		Name:        "waku_legacy_filter_subscribers",
		Measure:     LegacyFilterSubscribers,
		Description: "The number of legacy filter subscribers",
		Aggregation: view.LastValue(),
	}
	LegacyFilterMessagesView = &view.View{
		Name:        "waku_legacy_filter_messages",
		Measure:     LegacyFilterMessages,
		Description: "The distribution of the legacy filter protocol messages received",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyType},
	}
	LegacyFilterErrorTypesView = &view.View{
		Name:        "waku_legacy_filter_errors",
		Measure:     LegacyFilterErrors,
		Description: "The distribution of the legacy filter protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}

	FilterSubscriptionsView = &view.View{
		Name:        "waku_filter_subscriptions",
		Measure:     FilterSubscriptions,
		Description: "The number of filter subscriptions",
		Aggregation: view.Count(),
	}
	FilterRequestsView = &view.View{
		Name:        "waku_filter_requests",
		Measure:     FilterRequests,
		Description: "The number of filter requests",
		Aggregation: view.Count(),
	}
	FilterMessagesView = &view.View{
		Name:        "waku_filter_messages",
		Measure:     FilterMessages,
		Description: "The distribution of the filter protocol messages received",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{KeyType},
	}
	FilterErrorTypesView = &view.View{
		Name:        "waku_filter_errors",
		Measure:     FilterErrors,
		Description: "The distribution of the filter protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}

	FilterRequestDurationView = &view.View{
		Name:        "waku_filter_request_duration_seconds",
		Measure:     FilterRequestDurationSeconds,
		Description: "Duration of Filter Subscribe Requests",
		Aggregation: view.Count(),
	}
	FilterHandleMessageDurationView = &view.View{
		Name:        "waku_filter_handle_msessageduration_seconds",
		Measure:     FilterHandleMessageDurationSeconds,
		Description: "Duration to Push Message to Filter Subscribers",
		Aggregation: view.Count(),
	}

	LightpushMessagesView = &view.View{
		Name:        "waku_lightpush_messages",
		Measure:     LightpushMessages,
		Description: "The distribution of the lightpush protocol messages",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{KeyType},
	}
	LightpushErrorTypesView = &view.View{
		Name:        "waku_lightpush_errors",
		Measure:     LightpushErrors,
		Description: "The distribution of the lightpush protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
	VersionView = &view.View{
		Name:        "waku_version",
		Measure:     WakuVersion,
		Description: "The gowaku version",
		Aggregation: view.LastValue(),
		TagKeys:     []tag.Key{GitVersion},
	}
	DnsDiscoveryNodesView = &view.View{
		Name:        "waku_dnsdisc_discovered",
		Measure:     DnsDiscoveryNodes,
		Description: "The number of nodes discovered via DNS discovery",
		Aggregation: view.Count(),
	}
	DnsDiscoveryErrorTypesView = &view.View{
		Name:        "waku_dnsdisc_errors",
		Measure:     DnsDiscoveryErrors,
		Description: "The distribution of the dns discovery protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
	DiscV5ErrorTypesView = &view.View{
		Name:        "waku_discv5_errors",
		Measure:     DiscV5Errors,
		Description: "The distribution of the discv5 protocol errors",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{ErrorType},
	}
)

func recordWithTags(ctx context.Context, tagKey tag.Key, tagType string, ms stats.Measurement) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(tagKey, tagType)}, ms); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordLightpushMessage(ctx context.Context, tagType string) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, LightpushMessages.M(1)); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordLightpushError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, LightpushErrors.M(1))
}

func RecordLegacyFilterError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, LegacyFilterErrors.M(1))
}

func RecordArchiveError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, ArchiveErrors.M(1))
}

func RecordFilterError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, FilterErrors.M(1))
}

func RecordFilterRequest(ctx context.Context, tagType string, duration time.Duration) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, FilterRequests.M(1)); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
	FilterRequestDurationSeconds.M(int64(duration.Seconds()))
}

func RecordFilterMessage(ctx context.Context, tagType string, len int) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, FilterMessages.M(int64(len))); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordLegacyFilterMessage(ctx context.Context, tagType string, len int) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, LegacyFilterMessages.M(int64(len))); err != nil {
		utils.Logger().Error("failed to record with tags", zap.Error(err))
	}
}

func RecordPeerExchangeError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, PeerExchangeError.M(1))
}

func RecordDnsDiscoveryError(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, DnsDiscoveryErrors.M(1))
}

func RecordDiscV5Error(ctx context.Context, tagType string) {
	recordWithTags(ctx, ErrorType, tagType, DiscV5Errors.M(1))
}

func RecordArchiveMessage(ctx context.Context, tagType string, len int) {
	if err := stats.RecordWithTags(ctx, []tag.Mutator{tag.Insert(KeyType, tagType)}, ArchiveMessages.M(int64(len))); err != nil {
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
