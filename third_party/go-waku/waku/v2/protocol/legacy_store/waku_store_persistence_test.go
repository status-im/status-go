package legacy_store

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/timesource"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"google.golang.org/protobuf/proto"
)

func TestStorePersistence(t *testing.T) {
	db := MemoryDB(t)

	s1 := NewWakuStore(db, nil, timesource.NewDefaultClock(), prometheus.DefaultRegisterer, utils.Logger())

	defaultPubSubTopic := "test"
	defaultContentTopic := "1"
	msg := &pb.WakuMessage{
		Payload:      []byte{1, 2, 3},
		ContentTopic: defaultContentTopic,
		Timestamp:    utils.GetUnixEpoch(),
	}
	err := s1.storeMessage(protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), defaultPubSubTopic))
	require.NoError(t, err)

	msg2 := &pb.WakuMessage{
		Payload:      []byte{4, 5, 6},
		ContentTopic: defaultContentTopic,
		Timestamp:    utils.GetUnixEpoch(),
		Ephemeral:    proto.Bool(true), // Should not insert this message
	}
	err = s1.storeMessage(protocol.NewEnvelope(msg2, *utils.GetUnixEpoch(), defaultPubSubTopic))
	require.NoError(t, err)

	allMsgs, err := db.GetAll()
	require.NoError(t, err)
	require.Len(t, allMsgs, 1)
	require.True(t, proto.Equal(msg, allMsgs[0].Message))

	// Storing a duplicated message should not crash. It's okay to generate an error log in this case
	err = s1.storeMessage(protocol.NewEnvelope(msg, *utils.GetUnixEpoch(), defaultPubSubTopic))
	require.Error(t, err)
}
