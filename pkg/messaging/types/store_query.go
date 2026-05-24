package types

import "time"

// StoreQueryRequest describes an explicit historical message query against a
// store node. It is intentionally decoupled from chat filters: the caller is
// responsible for resolving filters into the criteria below (content topics +
// pubsub topic + time range).
//
// Peer/store selection is delegated to the underlying store ("at own risk"),
// reflecting that logos-delivery owns store/peer selection and recv-side
// catch-up. There is therefore no storenode argument on this seam.
type StoreQueryRequest struct {
	// PubsubTopic is the pubsub (shard) topic to query.
	PubsubTopic string
	// ContentTopics are the content topics to query within PubsubTopic.
	ContentTopics []ContentTopic
	// From and To bound the query time range.
	From time.Time
	To   time.Time

	// PageSize is the pagination limit for each page of the query.
	PageSize uint64

	// ProcessEnvelopes controls whether fetched envelopes are pushed into the
	// regular message-processing pipeline (true) or only surfaced to the
	// underlying store result (false).
	ProcessEnvelopes bool

	// ShouldProcessNextPage, when set, is invoked after each page with the
	// number of envelopes fetched in that page. Returning false stops
	// pagination; the returned page size is used for the next page.
	ShouldProcessNextPage func(envelopesFetched int) (next bool, pageSize uint64)
}
