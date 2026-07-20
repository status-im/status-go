package messaging

import (
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	reliabilitypb "github.com/status-im/status-go/pkg/messaging/layers/reliability/protobuf"
)

type sdsEnvelopeHashesTrackerStub struct {
	called         bool
	receivedID     []byte
	trackedHashes  []string
	trackedHashErr error
}

func (s *sdsEnvelopeHashesTrackerStub) TrackedEnvelopeHashes(identifier []byte) ([]string, error) {
	s.called = true
	s.receivedID = identifier
	return s.trackedHashes, s.trackedHashErr
}

func TestBuildSDSRetrievalHint_InvalidMessageID(t *testing.T) {
	tracker := &sdsEnvelopeHashesTrackerStub{}

	hint := buildSDSRetrievalHint(zap.NewNop(), tracker, "not-hex")

	require.Nil(t, hint)
	require.False(t, tracker.called)
}

func TestBuildSDSRetrievalHint_NoTrackedEnvelopeHashes(t *testing.T) {
	tracker := &sdsEnvelopeHashesTrackerStub{
		trackedHashErr: errors.New("not found"),
	}

	hint := buildSDSRetrievalHint(zap.NewNop(), tracker, "0x0102")

	require.Nil(t, hint)
	require.True(t, tracker.called)
	require.Equal(t, []byte{0x01, 0x02}, tracker.receivedID)
}

func TestBuildSDSRetrievalHint_Success(t *testing.T) {
	tracker := &sdsEnvelopeHashesTrackerStub{
		trackedHashes: []string{"0xabc", "0xdef"},
	}

	hintBytes := buildSDSRetrievalHint(zap.NewNop(), tracker, "0x0102")

	require.True(t, tracker.called)
	require.Equal(t, []byte{0x01, 0x02}, tracker.receivedID)
	require.NotNil(t, hintBytes)

	var hint reliabilitypb.RetrievalHint
	require.NoError(t, proto.Unmarshal(hintBytes, &hint))
	require.Equal(t, [][]byte{[]byte("0xabc"), []byte("0xdef")}, hint.EnvelopeHashes)
}
