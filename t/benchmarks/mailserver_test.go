package benchmarks

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/status-im/status-go/services/shhext"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/node"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/ext"
	"github.com/status-im/status-go/services/nodebridge"
	"github.com/status-im/status-go/whisper"
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
	config := params.ShhextConfig{
		BackupDisabledDataDir: os.TempDir(),
		InstallationID:        "1",
	}

	// create node with services
	n, err := createNode()
	require.NoError(t, err)
	err = n.Register(func(_ *node.ServiceContext) (node.Service, error) {
		return shhService, nil
	})
	require.NoError(t, err)
	// Register status-eth-node node bridge
	err = n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return &nodebridge.NodeService{Node: gethbridge.NewNodeBridge(n)}, nil
	})
	require.NoError(t, err)
	err = n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		var ethnode *nodebridge.NodeService
		if err := ctx.Service(&ethnode); err != nil {
			return nil, err
		}
		w, err := ethnode.Node.GetWhisper(ctx)
		if err != nil {
			return nil, err
		}
		return &nodebridge.WhisperService{Whisper: w}, nil
	})
	require.NoError(t, err)
	// register mail service as well
	err = n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return shhext.New(config, gethbridge.NewNodeBridge(n), ctx, nil, nil), nil
	})
	require.NoError(t, err)
	var mailService *shhext.Service
	require.NoError(t, n.Service(&mailService))
	shhextAPI := shhext.NewPublicAPI(mailService)

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
	requestID, err := shhextAPI.RequestMessages(context.TODO(), ext.MessagesRequest{
		MailServerPeer: *peerURL,
		SymKeyID:       symKeyID,
		Topic:          types.TopicType(topic),
	})
	require.NoError(t, err)
	require.NotNil(t, requestID)
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
