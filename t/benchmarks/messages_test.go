package benchmarks

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/stretchr/testify/require"
)

// TestSendMessages sends messages to a peer.
//
// Because of batching outgoing messages in Whisper V6,
// we need to pause and wait until the pending queue
// is emptied in Whisper API. Otherwise, the batch
// will be too large for the peer to consume it.
// It's a potential bug that Whisper code performs
//     packet.Size > whisper.MaxMessageSize()
// check instead of checking the size of each individual message.
func TestSendMessages(t *testing.T) {
	shhService := createWhisperService()
	shhAPI := whisper.NewPublicWhisperAPI(shhService)

	n, err := createNode()
	require.NoError(t, err)

	err = n.Register(func(_ *node.ServiceContext) (node.Service, error) {
		return shhService, nil
	})
	require.NoError(t, err)

	err = n.Start()
	require.NoError(t, err)
	defer func() { require.NoError(t, n.Stop()) }()

	err = addPeerWithConfirmation(n.Server(), peerEnode)
	require.NoError(t, err)

	symKeyID, err := shhService.AddSymKeyFromPassword(*msgPass)
	require.NoError(t, err)

	payload := make([]byte, *msgSize)
	rand.Read(payload)

	envelopeEvents := make(chan whisper.EnvelopeEvent, 100)
	sub := shhService.SubscribeEnvelopeEvents(envelopeEvents)
	defer sub.Unsubscribe()

	batchSent := make(chan struct{})
	envelopesSent := int64(0)
	go func() {
		for {
			select {
			case ev := <-envelopeEvents:
				if ev.Event == whisper.EventEnvelopeSent {
					envelopesSent++
				}

				if envelopesSent%(*msgBatchSize) == 0 {
					t.Logf("Sent a batch")
					batchSent <- struct{}{}
				}

				if envelopesSent == *msgCount {
					t.Logf("Sent all messages")
					close(batchSent)
					return
				}
			}
		}
	}()

	for i := int64(1); i <= *msgCount; i++ {
		_, err := shhAPI.Post(nil, whisper.NewMessage{
			SymKeyID:  symKeyID,
			TTL:       30,
			Topic:     topic,
			Payload:   payload,
			PowTime:   10,
			PowTarget: 0.005,
		})
		require.NoError(t, err)

		if i%(*msgBatchSize) == 0 {
			t.Logf("Waiting for a batch")
			<-batchSent
			time.Sleep(time.Second)
		}
	}

	t.Logf("Waiting for all messages to be sent")
	<-batchSent
	require.Equal(t, *msgCount, envelopesSent)
}
