package common

type StoreQueryRequest struct {
	RequestId         string         `json:"requestId"`
	IncludeData       bool           `json:"includeData"`
	PubsubTopic       string         `json:"pubsubTopic,omitempty"`
	ContentTopics     *[]string      `json:"contentTopics,omitempty"`
	TimeStart         *int64         `json:"timeStart,omitempty"`
	TimeEnd           *int64         `json:"timeEnd,omitempty"`
	MessageHashes     *[]MessageHash `json:"messageHashes,omitempty"`
	PaginationCursor  *MessageHash   `json:"paginationCursor,omitempty"`
	PaginationForward bool           `json:"paginationForward"`
	PaginationLimit   *uint64        `json:"paginationLimit,omitempty"`
}

type StoreMessageResponse struct {
	WakuMessage *wakuMessage `json:"message"`
	PubsubTopic string       `json:"pubsubTopic"`
	MessageHash MessageHash  `json:"messageHash"`
}

type StoreQueryResponse struct {
	RequestId        string                  `json:"requestId,omitempty"`
	StatusCode       *uint32                 `json:"statusCode,omitempty"`
	StatusDesc       string                  `json:"statusDesc,omitempty"`
	Messages         *[]StoreMessageResponse `json:"messages,omitempty"`
	PaginationCursor MessageHash             `json:"paginationCursor,omitempty"`
}
