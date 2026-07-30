package sender

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
)

type sdsWrapperStub struct {
	called         bool
	wantPayload    []byte
	wantChannelID  string
	wrappedPayload []byte
	messageID      []byte
	err            error
}

func (s *sdsWrapperStub) WrapPayloadForSDS(payload []byte, channelID string) ([]byte, []byte, error) {
	s.called = true
	s.wantPayload = payload
	s.wantChannelID = channelID
	return s.wrappedPayload, s.messageID, s.err
}

func TestWrapPayloadForPublicSDS_MissingCommunityID(t *testing.T) {
	logger := zap.NewNop()
	payload := []byte("payload")
	stub := &sdsWrapperStub{}

	wrapped, messageID, err := wrapPayloadForPublicSDS(logger, stub, payload, "")

	require.NoError(t, err)
	require.Equal(t, payload, wrapped)
	require.Nil(t, messageID)
	require.False(t, stub.called)
}

func TestWrapPayloadForPublicSDS_OversizedPayload(t *testing.T) {
	logger := zap.NewNop()
	payload := []byte("payload")
	communityID := "community-id"
	stub := &sdsWrapperStub{
		err: errors.New("reMessageTooLarge: message exceeds allowed size"),
	}

	wrapped, messageID, err := wrapPayloadForPublicSDS(logger, stub, payload, communityID)

	require.NoError(t, err)
	require.Equal(t, payload, wrapped)
	require.Nil(t, messageID)
	require.True(t, stub.called)
	require.Equal(t, payload, stub.wantPayload)
	require.Equal(t, reliability.BuildChannelID(communityID), stub.wantChannelID)
}

func TestWrapPayloadForPublicSDS_WrapError(t *testing.T) {
	logger := zap.NewNop()
	payload := []byte("payload")
	stub := &sdsWrapperStub{
		err: errors.New("boom"),
	}

	wrapped, messageID, err := wrapPayloadForPublicSDS(logger, stub, payload, "community-id")

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to wrap payload for SDS")
	require.Contains(t, err.Error(), "boom")
	require.Equal(t, payload, wrapped)
	require.Nil(t, messageID)
	require.True(t, stub.called)
}

func TestWrapPayloadForPublicSDS_Success(t *testing.T) {
	logger := zap.NewNop()
	payload := []byte("payload")
	wantedWrapped := []byte("wrapped-payload")
	wantedMessageID := []byte{0x01, 0x02, 0x03}
	stub := &sdsWrapperStub{
		wrappedPayload: wantedWrapped,
		messageID:      wantedMessageID,
	}

	wrapped, messageID, err := wrapPayloadForPublicSDS(logger, stub, payload, "community-id")

	require.NoError(t, err)
	require.Equal(t, wantedWrapped, wrapped)
	require.Equal(t, wantedMessageID, messageID)
	require.True(t, stub.called)
}
