package bridge

import (
	"math"
	"testing"
	"time"
	"unsafe"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/p2p"

	"github.com/status-im/status-go/waku"
	wakucommon "github.com/status-im/status-go/waku/common"
	"github.com/status-im/status-go/whisper/v6"
)

func TestEnvelopesBeingIdentical(t *testing.T) {
	// whisper.Envelope --> wakucommon.Envelope
	whisperEnvelope, err := createWhisperEnvelope()
	require.NoError(t, err)
	wakuEnvelope := (*wakucommon.Envelope)(unsafe.Pointer(whisperEnvelope)) // nolint: gosec
	require.Equal(t, whisperEnvelope.Hash(), wakuEnvelope.Hash())

	// wakucommon.Envelope --> whisper.Envelope
	wakuEnvelope, err = createWakuEnvelope()
	require.NoError(t, err)
	whisperEnvelope = (*whisper.Envelope)(unsafe.Pointer(wakuEnvelope)) // nolint: gosec
	require.Equal(t, wakuEnvelope.Hash(), whisperEnvelope.Hash())
}

func TestBridgeWhisperToWaku(t *testing.T) {
	shh := whisper.New(nil)
	shh.SetTimeSource(time.Now)
	wak := waku.New(nil, nil)
	wak.SetTimeSource(time.Now)
	b := New(shh, wak, zap.NewNop())
	b.Start()
	defer b.Cancel()

	server1 := createServer()
	err := shh.Start(server1)
	require.NoError(t, err)
	server2 := createServer()
	err = wak.Start(server2)
	require.NoError(t, err)

	// Subscribe for envelope events in Waku.
	eventsWaku := make(chan wakucommon.EnvelopeEvent, 10)
	sub1 := wak.SubscribeEnvelopeEvents(eventsWaku)
	defer sub1.Unsubscribe()

	// Subscribe for envelope events in Whisper.
	eventsWhsiper := make(chan whisper.EnvelopeEvent, 10)
	sub2 := shh.SubscribeEnvelopeEvents(eventsWhsiper)
	defer sub2.Unsubscribe()

	// Send message to Whisper and receive in Waku.
	envelope, err := createWhisperEnvelope()
	require.NoError(t, err)
	err = shh.Send(envelope)
	require.NoError(t, err)
	<-eventsWhsiper // skip event resulting from calling Send()

	// Verify that the message was received by waku.
	select {
	case err := <-sub1.Err():
		require.NoError(t, err)
	case event := <-eventsWaku:
		require.Equal(t, envelope.Hash(), event.Hash)
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	// Verify that the message was NOT received by whisper.
	select {
	case err := <-sub1.Err():
		require.NoError(t, err)
	case event := <-eventsWhsiper:
		t.Fatalf("unexpected event: %v", event)
	case <-time.After(time.Second):
		// expect to time out; TODO: replace with a bridge event which should not be sent by Waku
	}
}

func TestBridgeWakuToWhisper(t *testing.T) {
	shh := whisper.New(nil)
	shh.SetTimeSource(time.Now)
	wak := waku.New(nil, nil)
	wak.SetTimeSource(time.Now)
	b := New(shh, wak, zap.NewNop())
	b.Start()
	defer b.Cancel()

	server1 := createServer()
	err := shh.Start(server1)
	require.NoError(t, err)
	server2 := createServer()
	err = wak.Start(server2)
	require.NoError(t, err)

	// Subscribe for envelope events in Whisper.
	eventsWhisper := make(chan whisper.EnvelopeEvent, 10)
	sub1 := shh.SubscribeEnvelopeEvents(eventsWhisper)
	defer sub1.Unsubscribe()

	// Subscribe for envelope events in Waku.
	eventsWaku := make(chan wakucommon.EnvelopeEvent, 10)
	sub2 := wak.SubscribeEnvelopeEvents(eventsWaku)
	defer sub2.Unsubscribe()

	// Send message to Waku and receive in Whisper.
	envelope, err := createWakuEnvelope()
	require.NoError(t, err)
	err = wak.Send(envelope)
	require.NoError(t, err)
	<-eventsWaku // skip event resulting from calling Send()

	// Verify that the message was received by Whisper.
	select {
	case err := <-sub1.Err():
		require.NoError(t, err)
	case event := <-eventsWhisper:
		require.Equal(t, envelope.Hash(), event.Hash)
	case <-time.After(time.Second):
		t.Fatal("timed out")
	}

	// Verify that the message was NOT received by Waku.
	select {
	case err := <-sub1.Err():
		require.NoError(t, err)
	case event := <-eventsWaku:
		t.Fatalf("unexpected event: %v", event)
	case <-time.After(time.Second):
		// expect to time out; TODO: replace with a bridge event which should not be sent by Waku
	}
}

func createServer() *p2p.Server {
	return &p2p.Server{
		Config: p2p.Config{
			MaxPeers:    math.MaxInt32,
			NoDiscovery: true,
		},
	}
}

func createWhisperEnvelope() (*whisper.Envelope, error) {
	messageParams := &whisper.MessageParams{
		TTL:      120,
		KeySym:   []byte{0xaa, 0xbb, 0xcc},
		Topic:    whisper.BytesToTopic([]byte{0x01}),
		WorkTime: 10,
		PoW:      2.0,
		Payload:  []byte("hello!"),
	}
	sentMessage, err := whisper.NewSentMessage(messageParams)
	if err != nil {
		return nil, err
	}
	envelope := whisper.NewEnvelope(120, whisper.BytesToTopic([]byte{0x01}), sentMessage, time.Now())
	if err := envelope.Seal(messageParams); err != nil {
		return nil, err
	}
	return envelope, nil
}

func createWakuEnvelope() (*wakucommon.Envelope, error) {
	messageParams := &wakucommon.MessageParams{
		TTL:      120,
		KeySym:   []byte{0xaa, 0xbb, 0xcc},
		Topic:    wakucommon.BytesToTopic([]byte{0x01}),
		WorkTime: 10,
		PoW:      2.0,
		Payload:  []byte("hello!"),
	}
	sentMessage, err := wakucommon.NewSentMessage(messageParams)
	if err != nil {
		return nil, err
	}
	envelope := wakucommon.NewEnvelope(120, wakucommon.BytesToTopic([]byte{0x01}), sentMessage, time.Now())
	if err := envelope.Seal(messageParams); err != nil {
		return nil, err
	}
	return envelope, nil
}
