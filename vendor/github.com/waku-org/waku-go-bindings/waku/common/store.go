package common

type StoreQueryRequest struct {
	RequestId         string         `json:"request_id"`
	IncludeData       bool           `json:"include_data"`
	PubsubTopic       string         `json:"pubsub_topic,omitempty"`
	ContentTopics     *[]string      `json:"content_topics,omitempty"`
	TimeStart         *int64         `json:"time_start,omitempty"`
	TimeEnd           *int64         `json:"time_end,omitempty"`
	MessageHashes     *[]MessageHash `json:"message_hashes,omitempty"`
	PaginationCursor  *MessageHash   `json:"pagination_cursor,omitempty"`
	PaginationForward bool           `json:"pagination_forward"`
	PaginationLimit   *uint64        `json:"pagination_limit,omitempty"`
}

type StoreMessageResponse struct {
	WakuMessage *tmpWakuMessageJson `json:"message"`
	PubsubTopic string              `json:"pubsubTopic"`
	MessageHash MessageHash         `json:"messageHash"`
}

type StoreQueryResponse struct {
	RequestId        string                  `json:"requestId,omitempty"`
	StatusCode       *uint32                 `json:"statusCode,omitempty"`
	StatusDesc       string                  `json:"statusDesc,omitempty"`
	Messages         *[]StoreMessageResponse `json:"messages,omitempty"`
	PaginationCursor MessageHash             `json:"paginationCursor,omitempty"`
}
