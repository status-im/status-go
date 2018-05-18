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
)

var (
	mailServerEnode     = discover.MustParseNode(mailServerRawURL)
	payload             = make([]byte, whisper.DefaultMaxMessageSize/1024/8)
	sentMessagesCounter = 1000
)

func init() {
	rand.Read(payload)
}

func TestConcurrentMailserverPeers(t *testing.T) {
	// sent Whisper messages first
	n, err := createNode()
	require.NoError(t, err)

	shhService := createWhisperService()

	err = n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return shhService, nil
	})
	require.NoError(t, err)

	// start node and add mail server as peer
	require.NoError(t, n.Start())
	require.NoError(t, addPeerWithConfirmation(n.Server(), mailServerEnode))
	// wait until peer is handled by Whisper service
	time.Sleep(time.Second)

	symKeyID, err := shhService.AddSymKeyFromPassword("message-pass")
	require.NoError(t, err)

	shhAPI := whisper.NewPublicWhisperAPI(shhService)

	for i := 0; i < sentMessagesCounter; i++ {
		_, err := shhAPI.Post(nil, whisper.NewMessage{
			SymKeyID:  symKeyID,
			TTL:       30,
			Topic:     whisper.TopicType{0x01, 0x02, 0x03, 0x04},
			Payload:   payload,
			PowTime:   10,
			PowTarget: 0.005,
		})
		require.NoError(t, err)
	}

	// wait untill Whisper messages are propagated
	time.Sleep(time.Second * 5)
	require.NoError(t, n.Stop())

	// Request for messages from mail server
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("Peer #%d", i), testMailserverPeer)
	}
}

func testMailserverPeer(t *testing.T) {
	t.Parallel()

	n, err := createNode()
	require.NoError(t, err)

	shhService := createWhisperService()

	err = n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return shhService, nil
	})
	require.NoError(t, err)

	mailService := shhext.New(shhService, nil, nil)

	err = n.Register(func(_ *node.ServiceContext) (node.Service, error) {
		return mailService, nil
	})
	require.NoError(t, err)

	require.NoError(t, n.Start())
	defer func() { require.NoError(t, n.Stop()) }()

	server := n.Server()

	ch := make(chan *p2p.PeerEvent, server.MaxPeers)
	subscription := n.Server().SubscribeEvents(ch)
	defer subscription.Unsubscribe()

	server.AddPeer(mailServerEnode)

	select {
	case ev := <-ch:
		// t.Logf("received event: %+v", ev)
		if ev.Type == p2p.PeerEventTypeAdd && ev.Peer == mailServerEnode.ID {
			break
		} else {
			t.Errorf("got unexpected event: %+v", ev)
		}
	}

	symKeyID, err := shhService.AddSymKeyFromPassword("status-offline-inbox")
	require.NoError(t, err)

	err = shhService.AllowP2PMessagesFromPeer(mailServerEnode.ID[:])
	require.NoError(t, err)

	shhAPI := whisper.NewPublicWhisperAPI(shhService)

	ok, err := shhAPI.MarkTrustedPeer(nil, mailServerRawURL)
	require.NoError(t, err)
	require.True(t, ok)

	msgSymKeyID, err := shhService.AddSymKeyFromPassword("message-pass")
	require.NoError(t, err)

	filterID, err := shhAPI.NewMessageFilter(whisper.Criteria{
		SymKeyID: msgSymKeyID,
		Topics:   []whisper.TopicType{whisper.TopicType{0x01, 0x02, 0x03, 0x04}},
	})
	require.NoError(t, err)
	messages, err := shhAPI.GetFilterMessages(filterID)
	require.NoError(t, err)
	require.Len(t, messages, 0)

	shhextAPI := shhext.NewPublicAPI(mailService)

	counter := 0
FOR_LOOP:
	for {
		select {
		case <-time.After(time.Second):
			messages, err := shhAPI.GetFilterMessages(filterID)
			require.NoError(t, err)

			fmt.Println("received messages", len(messages))

			counter += len(messages)
			if counter >= sentMessagesCounter {
				break FOR_LOOP
			}
		}
	}

	ok, err = shhAPI.DeleteMessageFilter(filterID)
	require.NoError(t, err)
	require.True(t, ok)

	filterID, err = shhAPI.NewMessageFilter(whisper.Criteria{
		SymKeyID: msgSymKeyID,
		Topics:   []whisper.TopicType{whisper.TopicType{0x01, 0x02, 0x03, 0x04}},
		AllowP2P: true,
	})
	require.NoError(t, err)
	messages, err = shhAPI.GetFilterMessages(filterID)
	require.NoError(t, err)
	require.Len(t, messages, 0)

	ok, err = shhextAPI.RequestMessages(nil, shhext.MessagesRequest{
		MailServerPeer: mailServerRawURL,
		SymKeyID:       symKeyID,
		Topic:          whisper.TopicType{0x01, 0x02, 0x03, 0x04},
	})
	require.NoError(t, err)
	require.True(t, ok)

	counter = 0
	for {
		select {
		case <-time.After(time.Second):
			messages, err := shhAPI.GetFilterMessages(filterID)
			require.NoError(t, err)

			fmt.Println("received messages #2", len(messages))

			counter += len(messages)
			if counter >= sentMessagesCounter {
				return
			}
		}
	}
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
