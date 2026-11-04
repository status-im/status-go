package store

import (
	"github.com/waku-org/go-waku/waku/v2/protocol"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/store/pb"
	"google.golang.org/protobuf/proto"
)

type Criteria interface {
	PopulateStoreRequest(request *pb.StoreQueryRequest)
}

type FilterCriteria struct {
	protocol.ContentFilter
	TimeStart *int64
	TimeEnd   *int64
}

func (f FilterCriteria) PopulateStoreRequest(request *pb.StoreQueryRequest) {
	request.ContentTopics = f.ContentTopicsList()
	request.PubsubTopic = proto.String(f.PubsubTopic)
	request.TimeStart = f.TimeStart
	request.TimeEnd = f.TimeEnd
}

type MessageHashCriteria struct {
	MessageHashes []wpb.MessageHash
}

func (m MessageHashCriteria) PopulateStoreRequest(request *pb.StoreQueryRequest) {
	request.MessageHashes = make([][]byte, len(m.MessageHashes))
	for i := range m.MessageHashes {
		request.MessageHashes[i] = m.MessageHashes[i][:]
	}
}
