package sender

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/status-im/status-go/internal/crypto"
	cryptotypes "github.com/status-im/status-go/internal/crypto/types"
	"github.com/status-im/status-go/pkg/messaging/layers/reliability"
	messagingtypes "github.com/status-im/status-go/pkg/messaging/types"
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

func TestWrapPublicParamsForSDS_SkipSDS(t *testing.T) {
	payload := []byte("payload")
	stub := &sdsWrapperStub{wrappedPayload: []byte("wrapped-payload")}

	wrapped, messageID, err := wrapPublicParamsForSDS(zap.NewNop(), stub, messagingtypes.SendPublicParams{
		Payload:     payload,
		CommunityID: "community-id",
		SkipSDS:     true,
	})

	require.NoError(t, err)
	require.Equal(t, payload, wrapped)
	require.Nil(t, messageID)
	require.False(t, stub.called)
}

func TestWrapPublicParamsForSDS_WrapsCommunityPayload(t *testing.T) {
	stub := &sdsWrapperStub{wrappedPayload: []byte("wrapped-payload"), messageID: []byte{0x01}}

	wrapped, messageID, err := wrapPublicParamsForSDS(zap.NewNop(), stub, messagingtypes.SendPublicParams{
		Payload:     []byte("payload"),
		CommunityID: "community-id",
	})

	require.NoError(t, err)
	require.Equal(t, stub.wrappedPayload, wrapped)
	require.Equal(t, stub.messageID, messageID)
	require.Equal(t, reliability.BuildChannelID("community-id"), stub.wantChannelID)
}

func TestWrapPublicParamsForSDS_DerivesCommunityIDFromPublicKey(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	stub := &sdsWrapperStub{wrappedPayload: []byte("wrapped-payload")}

	_, _, err = wrapPublicParamsForSDS(zap.NewNop(), stub, messagingtypes.SendPublicParams{
		Payload:            []byte("payload"),
		CommunityPublicKey: &key.PublicKey,
	})

	require.NoError(t, err)
	communityID := cryptotypes.EncodeHex(crypto.CompressPubkey(&key.PublicKey))
	require.Equal(t, reliability.BuildChannelID(communityID), stub.wantChannelID)
}
