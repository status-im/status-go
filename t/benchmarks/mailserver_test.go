package benchmarks

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/services/shhext"
	// _ "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/require"
)

const (
	mailServerRawURL = "enode://767808076b264cda39fb28156e1c6b92d3d527be41a5af8cd24c809f44137fadd4b9e4397f6a1e621582ce7b951d85a9ff8e7c53aca52168df696f9436b9dadc@127.0.0.1:30303"
	messagesCount    = 1000           // number of messages to send to Mail Server first
	messagePassword  = "message-pass" // password to decrypt a message
	numberOfPeers    = 10             // number of peers requesting messages concurrently
)

var (
	mailServerEnode = discover.MustParseNode(mailServerRawURL)
	payload         = make([]byte, whisper.DefaultMaxMessageSize/1024/8)
	topic           = whisper.TopicType{0x01, 0x02, 0x03, 0x04}
)

func init() {
	rand.Read(payload)
}

func TestConcurrentMailserverPeers(t *testing.T) {
	testSendingMessages(t)

	// Request for messages from mail server
	for i := 0; i < numberOfPeers; i++ {
		t.Run(fmt.Sprintf("Peer #%d", i), testMailserverPeer)
	}
}

func testSendingMessages(t *testing.T) {
	shhService := createWhisperService()
	shhAPI := whisper.NewPublicWhisperAPI(shhService)

	// create node with services
	n, err := createNode()
	require.NoError(t, err)
	err = n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return shhService, nil
	})
	require.NoError(t, err)

	// start node
	require.NoError(t, n.Start())

	// add mail server as a peer
	require.NoError(t, addPeerWithConfirmation(n.Server(), mailServerEnode))
	// wait until peer is handled by Whisper service
	time.Sleep(time.Second)

	symKeyID, err := shhService.AddSymKeyFromPassword(messagePassword)
	require.NoError(t, err)

	for i := 0; i < messagesCount; i++ {
		_, err := shhAPI.Post(nil, whisper.NewMessage{
			SymKeyID:  symKeyID,
			TTL:       30,
			Topic:     topic,
			Payload:   payload,
			PowTime:   10,
			PowTarget: 0.005,
		})
		require.NoError(t, err)
	}

	// wait untill Whisper messages are propagated
	time.Sleep(time.Second * 5)
	require.NoError(t, n.Stop())
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
	err = n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
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
	require.NoError(t, addPeerWithConfirmation(n.Server(), mailServerEnode))

	// sym key to decrypt messages
	msgSymKeyID, err := shhService.AddSymKeyFromPassword(messagePassword)
	require.NoError(t, err)

	// load messages to cache
	filterID, err := shhAPI.NewMessageFilter(whisper.Criteria{
		SymKeyID: msgSymKeyID,
		Topics:   []whisper.TopicType{topic},
	})
	require.NoError(t, err)
	messages, err := shhAPI.GetFilterMessages(filterID)
	require.NoError(t, err)
	require.Len(t, messages, 0)
	// wait for messages
	require.NoError(t, waitForMessages(messagesCount, shhAPI, filterID))

	// clean up old filter
	ok, err := shhAPI.DeleteMessageFilter(filterID)
	require.NoError(t, err)
	require.True(t, ok)

	// prepare new filter for messages from mail server
	filterID, err = shhAPI.NewMessageFilter(whisper.Criteria{
		SymKeyID: msgSymKeyID,
		Topics:   []whisper.TopicType{topic},
		AllowP2P: true,
	})
	require.NoError(t, err)
	messages, err = shhAPI.GetFilterMessages(filterID)
	require.NoError(t, err)
	require.Len(t, messages, 0)

	// request messages from mail server
	symKeyID, err := shhService.AddSymKeyFromPassword("status-offline-inbox")
	require.NoError(t, err)
	ok, err = shhAPI.MarkTrustedPeer(nil, mailServerRawURL)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = shhextAPI.RequestMessages(nil, shhext.MessagesRequest{
		MailServerPeer: mailServerRawURL,
		SymKeyID:       symKeyID,
		Topic:          whisper.TopicType{0x01, 0x02, 0x03, 0x04},
	})
	require.NoError(t, err)
	require.True(t, ok)
	// wait for all messages
	require.NoError(t, waitForMessages(messagesCount, shhAPI, filterID))
}

func createNode() (*node.Node, error) {
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return node.New(&node.Config{
		DataDir: "",
		P2P: p2p.Config{
			PrivateKey:  key,
			DiscoveryV5: false,
			NoDiscovery: true,
			MaxPeers:    1,
			NAT:         nat.Any(),
		},
	})
}

func createWhisperService() *whisper.Whisper {
	whisperServiceConfig := &whisper.Config{
		MaxMessageSize:     whisper.DefaultMaxMessageSize,
		MinimumAcceptedPOW: 0.005,
		TimeSource:         func() time.Time { return time.Now().UTC() },
	}
	return whisper.New(whisperServiceConfig)
}

func addPeerWithConfirmation(server *p2p.Server, node *discover.Node) error {
	ch := make(chan *p2p.PeerEvent, server.MaxPeers)
	subscription := server.SubscribeEvents(ch)
	defer subscription.Unsubscribe()

	server.AddPeer(node)

	select {
	case ev := <-ch:
		if ev.Type == p2p.PeerEventTypeAdd && ev.Peer == node.ID {
			return nil
		}

		return fmt.Errorf("got unexpected event: %+v", ev)
	}
}

func waitForMessages(messagesCount int, shhAPI *whisper.PublicWhisperAPI, filterID string) error {
	received := 0

	for {
		select {
		case <-time.After(time.Second):
			messages, err := shhAPI.GetFilterMessages(filterID)
			if err != nil {
				return err
			}

			received += len(messages)
			if received >= messagesCount {
				return nil
			}
		}
	}
}
