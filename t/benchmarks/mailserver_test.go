package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/node"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/services/shhext"
	"github.com/stretchr/testify/require"
)

const (
	mailServerPass = "status-offline-inbox"
)

// TestConcurrentMailserverPeers runs `ccyPeers` tests in parallel
// that require messages from a MailServer.
//
// It can be used to test the maximum number of concurrent MailServer peers.
//
// Messages stored by the MailServer must be generated separately.
// Take a look at TestSendMessages test.
func TestConcurrentMailserverPeers(t *testing.T) {
	// Request for messages from mail server
	for i := 0; i < *ccyPeers; i++ {
		t.Run(fmt.Sprintf("Peer #%d", i), testMailserverPeer)
	}
}

func testMailserverPeer(t *testing.T) {
	t.Parallel()

	shhService := createWhisperService()
	shhAPI := whisper.NewPublicWhisperAPI(shhService)
	mailService := shhext.New(shhService, nil, nil)
	shhextAPI := shhext.NewPublicAPI(mailService)

	// create node with services
	n, err := createNode()
	require.NoError(t, err)
	err = n.Register(func(_ *node.ServiceContext) (node.Service, error) {
		return shhService, nil
	})
	require.NoError(t, err)
	// register mail service as well
	err = n.Register(func(_ *node.ServiceContext) (node.Service, error) {
		return mailService, nil
	})
	require.NoError(t, err)

	// start node
	require.NoError(t, n.Start())
	defer func() { require.NoError(t, n.Stop()) }()

	// add mail server as a peer
	require.NoError(t, addPeerWithConfirmation(n.Server(), peerEnode))

	// sym key to decrypt messages
	msgSymKeyID, err := shhService.AddSymKeyFromPassword(*msgPass)
	require.NoError(t, err)

	// prepare new filter for messages from mail server
	filterID, err := shhAPI.NewMessageFilter(whisper.Criteria{
		SymKeyID: msgSymKeyID,
		Topics:   []whisper.TopicType{topic},
		AllowP2P: true,
	})
	require.NoError(t, err)
	messages, err := shhAPI.GetFilterMessages(filterID)
	require.NoError(t, err)
	require.Len(t, messages, 0)

	// request messages from mail server
	symKeyID, err := shhService.AddSymKeyFromPassword(mailServerPass)
	require.NoError(t, err)
	ok, err := shhAPI.MarkTrustedPeer(context.TODO(), *peerURL)
	require.NoError(t, err)
	require.True(t, ok)
	hash, err := shhextAPI.RequestMessages(context.TODO(), shhext.MessagesRequest{
		MailServerPeer: *peerURL,
		SymKeyID:       symKeyID,
		Topic:          topic,
	})
	require.NoError(t, err)
	require.NotNil(t, hash)
	// wait for all messages
	require.NoError(t, waitForMessages(t, *msgCount, shhAPI, filterID))
}

func waitForMessages(t *testing.T, messagesCount int64, shhAPI *whisper.PublicWhisperAPI, filterID string) error {
	received := int64(0)
	for range time.After(time.Second) {
		messages, err := shhAPI.GetFilterMessages(filterID)
		if err != nil {
			return err
		}

		received += int64(len(messages))

		fmt.Printf("Received %d messages so far\n", received)

		if received >= messagesCount {
			return nil
		}
	}

	return nil
}
