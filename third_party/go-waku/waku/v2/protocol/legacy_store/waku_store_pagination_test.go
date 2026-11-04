package legacy_store

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/persistence"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/legacy_store/pb"
	wpb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

func TestIndexComputation(t *testing.T) {
	testContentTopic := "/waku/2/default-content/proto"
	msg := &wpb.WakuMessage{
		Payload:   []byte{1, 2, 3},
		Timestamp: utils.GetUnixEpoch(),
	}

	idx := protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), "test").Index()
	require.NotZero(t, idx.ReceiverTime)
	require.Equal(t, msg.GetTimestamp(), idx.SenderTime)
	require.NotZero(t, idx.Digest)
	require.Len(t, idx.Digest, 32)

	msg1 := &wpb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    proto.Int64(123),
		ContentTopic: testContentTopic,
	}
	idx1 := protocol.NewEnvelope(msg1, *utils.GetUnixEpoch(), "test").Index()

	msg2 := &wpb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		Timestamp:    proto.Int64(123),
		ContentTopic: testContentTopic,
	}
	idx2 := protocol.NewEnvelope(msg2, *utils.GetUnixEpoch(), "test").Index()

	require.Equal(t, idx1.Digest, idx2.Digest)
}

func createSampleList(s int) []*protocol.Envelope {
	var result []*protocol.Envelope
	for i := 1; i <= s; i++ {
		msg :=
			&wpb.WakuMessage{
				Payload:   []byte{byte(i)},
				Timestamp: proto.Int64(int64(i)),
			}
		result = append(result, protocol.NewEnvelope(msg, int64(i), "abc"))
	}
	return result
}

func TestForwardPagination(t *testing.T) {
	msgList := createSampleList(10)
	db := MemoryDB(t)
	for _, m := range msgList {
		err := db.Put(m)
		require.NoError(t, err)
	}

	// test for a normal pagination
	pagingInfo := &pb.PagingInfo{PageSize: 2, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err := findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.True(t, proto.Equal(msgList[4].Message(), messages[0]))
	require.True(t, proto.Equal(msgList[5].Message(), messages[1]))
	require.Equal(t, msgList[5].Index(), newPagingInfo.Cursor)

	// test for an initial pagination request with an empty cursor
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.Equal(t, msgList[0].Message(), messages[0])

	require.True(t, proto.Equal(msgList[0].Message(), messages[0]))
	require.True(t, proto.Equal(msgList[1].Message(), messages[1]))
	require.Equal(t, msgList[1].Index(), newPagingInfo.Cursor)

	// test for an initial pagination request with an empty cursor to fetch the entire history
	pagingInfo = &pb.PagingInfo{PageSize: 13, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 10)
	require.True(t, proto.Equal(msgList[9].Message(), messages[9]))
	require.Nil(t, newPagingInfo.Cursor)

	// test for an empty msgList
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, MemoryDB(t))
	require.NoError(t, err)
	require.Len(t, messages, 0)
	require.Nil(t, newPagingInfo.Cursor)

	// test for a page size larger than the remaining messages
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 6)
	require.True(t, proto.Equal(msgList[4].Message(), messages[0]))
	require.True(t, proto.Equal(msgList[9].Message(), messages[5]))
	require.Nil(t, newPagingInfo.Cursor)

	// test for a page size larger than the maximum allowed page size
	pagingInfo = &pb.PagingInfo{PageSize: MaxPageSize + 1, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.True(t, len(messages) <= MaxPageSize)
	require.Nil(t, newPagingInfo.Cursor)

	// test for a cursor pointing to the end of the message list
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: msgList[9].Index(), Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 0)
	require.Nil(t, newPagingInfo.Cursor)

	// test for an invalid cursor
	invalidIndex := protocol.NewEnvelope(&wpb.WakuMessage{Payload: []byte{255, 255, 255}}, *utils.GetUnixEpoch(), "test").Index()
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: invalidIndex, Direction: pb.PagingInfo_FORWARD}
	_, _, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.ErrorIs(t, err, persistence.ErrInvalidCursor)
	require.Len(t, messages, 0)

	// test initial paging query over a message list with one message
	singleItemDB := MemoryDB(t)
	err = singleItemDB.Put(msgList[0])
	require.NoError(t, err)
	pagingInfo = &pb.PagingInfo{PageSize: 10, Direction: pb.PagingInfo_FORWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, singleItemDB)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Nil(t, newPagingInfo.Cursor)
}

func TestBackwardPagination(t *testing.T) {
	msgList := createSampleList(10)
	db := MemoryDB(t)
	for _, m := range msgList {
		err := db.Put(m)
		require.NoError(t, err)
	}

	// test for a normal pagination
	pagingInfo := &pb.PagingInfo{PageSize: 2, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err := findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 2)

	require.True(t, proto.Equal(msgList[1].Message(), messages[0]))
	require.True(t, proto.Equal(msgList[2].Message(), messages[1]))
	require.Equal(t, msgList[1].Index(), newPagingInfo.Cursor)

	// test for an initial pagination request with an empty cursor
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 2)
	require.True(t, proto.Equal(msgList[8].Message(), messages[0]))
	require.True(t, proto.Equal(msgList[9].Message(), messages[1]))
	require.Equal(t, msgList[8].Index(), newPagingInfo.Cursor)

	// test for an initial pagination request with an empty cursor to fetch the entire history
	pagingInfo = &pb.PagingInfo{PageSize: 13, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 10)
	require.True(t, proto.Equal(msgList[0].Message(), messages[0]))
	require.True(t, proto.Equal(msgList[9].Message(), messages[9]))
	require.Nil(t, newPagingInfo.Cursor)

	// test for an empty msgList
	pagingInfo = &pb.PagingInfo{PageSize: 2, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, MemoryDB(t))
	require.NoError(t, err)
	require.Len(t, messages, 0)
	require.Equal(t, pagingInfo.Cursor, newPagingInfo.Cursor)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.Equal(t, uint64(0), newPagingInfo.PageSize)

	// test for a page size larger than the remaining messages
	pagingInfo = &pb.PagingInfo{PageSize: 5, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 3)
	require.True(t, proto.Equal(msgList[0].Message(), messages[0]))
	require.True(t, proto.Equal(msgList[2].Message(), messages[2]))
	require.Nil(t, newPagingInfo.Cursor)

	// test for a page size larger than the maximum allowed page size
	pagingInfo = &pb.PagingInfo{PageSize: MaxPageSize + 1, Cursor: msgList[3].Index(), Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.True(t, len(messages) <= MaxPageSize)
	require.Equal(t, pagingInfo.Direction, newPagingInfo.Direction)
	require.True(t, newPagingInfo.PageSize <= MaxPageSize)

	// test for a cursor pointing to the beginning of the message list
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: msgList[0].Index(), Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.NoError(t, err)
	require.Len(t, messages, 0)
	require.Nil(t, newPagingInfo.Cursor)

	// test for an invalid cursor
	invalidIndex := protocol.NewEnvelope(&wpb.WakuMessage{Payload: []byte{255, 255, 255}}, *utils.GetUnixEpoch(), "test").Index()
	pagingInfo = &pb.PagingInfo{PageSize: 10, Cursor: invalidIndex, Direction: pb.PagingInfo_BACKWARD}
	_, _, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, db)
	require.ErrorIs(t, err, persistence.ErrInvalidCursor)
	require.Len(t, messages, 0)

	// test initial paging query over a message list with one message
	singleItemDB := MemoryDB(t)
	err = singleItemDB.Put(msgList[0])
	require.NoError(t, err)
	pagingInfo = &pb.PagingInfo{PageSize: 10, Direction: pb.PagingInfo_BACKWARD}
	messages, newPagingInfo, err = findMessages(&pb.HistoryQuery{PagingInfo: pagingInfo}, singleItemDB)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Nil(t, newPagingInfo.Cursor)
}
