package whisper

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/t/helpers"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

const mailboxPassword = "status-offline-inbox"

type WhisperMailboxSuite struct {
	suite.Suite
}

func TestWhisperMailboxTestSuite(t *testing.T) {
	suite.Run(t, new(WhisperMailboxSuite))
}

func (s *WhisperMailboxSuite) TestRequestMessageFromMailboxAsync() {
	var err error
	// Start mailbox and status node.
	mailboxBackend, stop := s.startMailboxBackend()
	defer stop()
	s.Require().True(mailboxBackend.IsNodeRunning())
	mailboxNode := mailboxBackend.StatusNode().GethNode()
	mailboxEnode := mailboxNode.Server().NodeInfo().Enode

	sender, stop := s.startBackend("sender")
	defer stop()
	s.Require().True(sender.IsNodeRunning())
	node := sender.StatusNode().GethNode()

	s.Require().NotEqual(mailboxEnode, node.Server().NodeInfo().Enode)

	err = sender.StatusNode().AddPeer(mailboxEnode)
	s.Require().NoError(err)

	waitErr := helpers.WaitForPeerAsync(node.Server(), mailboxEnode, p2p.PeerEventTypeAdd, time.Second)
	s.NoError(<-waitErr)

	senderWhisperService, err := sender.StatusNode().WhisperService()
	s.Require().NoError(err)

	// Mark mailbox node trusted.
	parsedNode, err := discover.ParseNode(mailboxNode.Server().NodeInfo().Enode)
	s.Require().NoError(err)
	mailboxPeer := parsedNode.ID[:]
	mailboxPeerStr := parsedNode.ID.String()
	err = senderWhisperService.AllowP2PMessagesFromPeer(mailboxPeer)
	s.Require().NoError(err)

	// Generate mailbox symkey.
	password := mailboxPassword
	MailServerKeyID, err := senderWhisperService.AddSymKeyFromPassword(password)
	s.Require().NoError(err)

	rpcClient := sender.StatusNode().RPCClient()
	s.Require().NotNil(rpcClient)

	mailboxWhisperService, err := mailboxBackend.StatusNode().WhisperService()
	s.Require().NoError(err)
	s.Require().NotNil(mailboxWhisperService)

	// watch envelopes to be archived on mailserver
	envelopeArchivedWatcher := make(chan whisper.EnvelopeEvent, 1024)
	mailboxWhisperService.SubscribeEnvelopeEvents(envelopeArchivedWatcher)

	// watch envelopes to be available for filters in the client
	envelopeAvailableWatcher := make(chan whisper.EnvelopeEvent, 1024)
	senderWhisperService.SubscribeEnvelopeEvents(envelopeAvailableWatcher)

	// watch mailserver responses in the client
	mailServerResponseWatcher := make(chan whisper.EnvelopeEvent, 1024)
	senderWhisperService.SubscribeEnvelopeEvents(mailServerResponseWatcher)

	// Create topic.
	topic := whisper.BytesToTopic([]byte("topic name"))

	// Add key pair to whisper.
	keyID, err := senderWhisperService.NewKeyPair()
	s.Require().NoError(err)
	key, err := senderWhisperService.GetPrivateKey(keyID)
	s.Require().NoError(err)
	pubkey := hexutil.Bytes(crypto.FromECDSAPub(&key.PublicKey))

	// Create message filter.
	messageFilterID := s.createPrivateChatMessageFilter(rpcClient, keyID, topic.String())

	// There are no messages at filter.
	messages := s.getMessagesByMessageFilterID(rpcClient, messageFilterID)
	s.Require().Empty(messages)

	// Post message matching with filter (key and topic).
	messageHash := s.postMessageToPrivate(rpcClient, pubkey.String(), topic.String(), hexutil.Encode([]byte("Hello world!")))

	// Get message to make sure that it will come from the mailbox later.
	s.waitForEnvelopeEvents(envelopeAvailableWatcher, []string{messageHash}, whisper.EventEnvelopeAvailable)
	messages = s.getMessagesByMessageFilterID(rpcClient, messageFilterID)
	s.Require().Equal(1, len(messages))

	// Act.

	// wait for mailserver to archive all the envelopes
	s.waitForEnvelopeEvents(envelopeArchivedWatcher, []string{messageHash}, whisper.EventMailServerEnvelopeArchived)

	// Request messages (including the previous one, expired) from mailbox.
	requestID := s.requestHistoricMessagesFromLast12Hours(senderWhisperService, rpcClient, mailboxPeerStr, MailServerKeyID, topic.String(), 0, "")

	// wait for mail server response
	resp := s.waitForMailServerResponse(mailServerResponseWatcher, requestID)
	s.Equal(messageHash, resp.LastEnvelopeHash.String())
	s.Empty(resp.Cursor)

	// wait for last envelope sent by the mailserver to be available for filters
	s.waitForEnvelopeEvents(envelopeAvailableWatcher, []string{resp.LastEnvelopeHash.String()}, whisper.EventEnvelopeAvailable)

	// And we receive message, it comes from mailbox.
	messages = s.getMessagesByMessageFilterID(rpcClient, messageFilterID)
	s.Require().Equal(1, len(messages))

	// Check that there are no messages.
	messages = s.getMessagesByMessageFilterID(rpcClient, messageFilterID)
	s.Require().Empty(messages)
}

func (s *WhisperMailboxSuite) TestRequestMessagesInGroupChat() {
	var err error

	// Start mailbox, alice, bob, charlie node.
	mailboxBackend, stop := s.startMailboxBackend()
	defer stop()

	aliceBackend, stop := s.startBackend("alice")
	defer stop()

	bobBackend, stop := s.startBackend("bob")
	defer stop()

	charlieBackend, stop := s.startBackend("charlie")
	defer stop()

	// Add mailbox to static peers.
	s.Require().True(mailboxBackend.IsNodeRunning())
	mailboxNode := mailboxBackend.StatusNode().GethNode()
	mailboxEnode := mailboxNode.Server().NodeInfo().Enode

	err = aliceBackend.StatusNode().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	waitErr := helpers.WaitForPeerAsync(aliceBackend.StatusNode().GethNode().Server(), mailboxEnode, p2p.PeerEventTypeAdd, time.Second)
	s.NoError(<-waitErr)

	err = bobBackend.StatusNode().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	waitErr = helpers.WaitForPeerAsync(bobBackend.StatusNode().GethNode().Server(), mailboxEnode, p2p.PeerEventTypeAdd, time.Second)
	s.NoError(<-waitErr)

	err = charlieBackend.StatusNode().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	waitErr = helpers.WaitForPeerAsync(charlieBackend.StatusNode().GethNode().Server(), mailboxEnode, p2p.PeerEventTypeAdd, time.Second)
	s.NoError(<-waitErr)

	// Get whisper service.
	mailboxWhisperService, err := mailboxBackend.StatusNode().WhisperService()
	s.Require().NoError(err)
	aliceWhisperService, err := aliceBackend.StatusNode().WhisperService()
	s.Require().NoError(err)
	bobWhisperService, err := bobBackend.StatusNode().WhisperService()
	s.Require().NoError(err)
	charlieWhisperService, err := charlieBackend.StatusNode().WhisperService()
	s.Require().NoError(err)
	// Get rpc client.
	aliceRPCClient := aliceBackend.StatusNode().RPCClient()
	bobRPCClient := bobBackend.StatusNode().RPCClient()
	charlieRPCClient := charlieBackend.StatusNode().RPCClient()

	// watchers
	envelopeArchivedWatcher := make(chan whisper.EnvelopeEvent, 1024)
	mailboxWhisperService.SubscribeEnvelopeEvents(envelopeArchivedWatcher)

	bobEnvelopeAvailableWatcher := make(chan whisper.EnvelopeEvent, 1024)
	bobWhisperService.SubscribeEnvelopeEvents(bobEnvelopeAvailableWatcher)

	charlieEnvelopeAvailableWatcher := make(chan whisper.EnvelopeEvent, 1024)
	charlieWhisperService.SubscribeEnvelopeEvents(charlieEnvelopeAvailableWatcher)

	// Bob and charlie add the mailserver key.
	password := mailboxPassword
	bobMailServerKeyID, err := bobWhisperService.AddSymKeyFromPassword(password)
	s.Require().NoError(err)
	charlieMailServerKeyID, err := charlieWhisperService.AddSymKeyFromPassword(password)
	s.Require().NoError(err)

	// Generate a group chat symkey and topic.
	groupChatKeyID, err := aliceWhisperService.GenerateSymKey()
	s.Require().NoError(err)
	groupChatKey, err := aliceWhisperService.GetSymKey(groupChatKeyID)
	s.Require().NoError(err)
	// Generate a group chat topic.
	groupChatTopic := whisper.BytesToTopic([]byte("groupChatTopic"))
	// sender must be subscribed to message topic it sends
	s.NotNil(s.createGroupChatMessageFilter(aliceRPCClient, groupChatKeyID, groupChatTopic.String()))
	groupChatPayload := newGroupChatParams(groupChatKey, groupChatTopic)
	payloadStr, err := groupChatPayload.Encode()
	s.Require().NoError(err)

	// Add Bob and Charlie's key pairs to receive the symmetric key for the group chat from Alice.
	bobKeyID, err := bobWhisperService.NewKeyPair()
	s.Require().NoError(err)
	bobKey, err := bobWhisperService.GetPrivateKey(bobKeyID)
	s.Require().NoError(err)
	bobPubkey := hexutil.Bytes(crypto.FromECDSAPub(&bobKey.PublicKey))
	bobAliceKeySendTopic := whisper.BytesToTopic([]byte("bobAliceKeySendTopic "))

	charlieKeyID, err := charlieWhisperService.NewKeyPair()
	s.Require().NoError(err)
	charlieKey, err := charlieWhisperService.GetPrivateKey(charlieKeyID)
	s.Require().NoError(err)
	charliePubkey := hexutil.Bytes(crypto.FromECDSAPub(&charlieKey.PublicKey))
	charlieAliceKeySendTopic := whisper.BytesToTopic([]byte("charlieAliceKeySendTopic "))

	// Alice must add peers topics into her own bloom filter.
	aliceKeyID, err := aliceWhisperService.NewKeyPair()
	s.Require().NoError(err)
	s.createPrivateChatMessageFilter(aliceRPCClient, aliceKeyID, bobAliceKeySendTopic.String())
	s.createPrivateChatMessageFilter(aliceRPCClient, aliceKeyID, charlieAliceKeySendTopic.String())

	// Bob and charlie create message filter.
	bobMessageFilterID := s.createPrivateChatMessageFilter(bobRPCClient, bobKeyID, bobAliceKeySendTopic.String())
	charlieMessageFilterID := s.createPrivateChatMessageFilter(charlieRPCClient, charlieKeyID, charlieAliceKeySendTopic.String())

	// Alice send message with symkey and topic to Bob and Charlie.
	aliceToBobMessageHash := s.postMessageToPrivate(aliceRPCClient, bobPubkey.String(), bobAliceKeySendTopic.String(), payloadStr)
	aliceToCharlieMessageHash := s.postMessageToPrivate(aliceRPCClient, charliePubkey.String(), charlieAliceKeySendTopic.String(), payloadStr)

	// Bob receive group chat data and add it to his node.
	// Bob get group chat details.
	s.waitForEnvelopeEvents(bobEnvelopeAvailableWatcher, []string{aliceToBobMessageHash}, whisper.EventEnvelopeAvailable)
	messages := s.getMessagesByMessageFilterID(bobRPCClient, bobMessageFilterID)
	s.Require().Equal(1, len(messages))
	bobGroupChatData := groupChatParams{}
	err = bobGroupChatData.Decode(messages[0]["payload"].(string))
	s.Require().NoError(err)
	s.EqualValues(groupChatPayload, bobGroupChatData)

	// Bob add symkey to his node.
	bobGroupChatSymkeyID := s.addSymKey(bobRPCClient, bobGroupChatData.Key)
	s.Require().NotEmpty(bobGroupChatSymkeyID)

	// Bob create message filter to node by group chat topic.
	bobGroupChatMessageFilterID := s.createGroupChatMessageFilter(bobRPCClient, bobGroupChatSymkeyID, bobGroupChatData.Topic)

	// Charlie receive group chat data and add it to his node.
	// Charlie get group chat details.
	s.waitForEnvelopeEvents(charlieEnvelopeAvailableWatcher, []string{aliceToCharlieMessageHash}, whisper.EventEnvelopeAvailable)
	messages = s.getMessagesByMessageFilterID(charlieRPCClient, charlieMessageFilterID)
	s.Require().Equal(1, len(messages))
	charlieGroupChatData := groupChatParams{}
	err = charlieGroupChatData.Decode(messages[0]["payload"].(string))
	s.Require().NoError(err)
	s.EqualValues(groupChatPayload, charlieGroupChatData)

	// Charlie add symkey to his node.
	charlieGroupChatSymkeyID := s.addSymKey(charlieRPCClient, charlieGroupChatData.Key)
	s.Require().NotEmpty(charlieGroupChatSymkeyID)

	// Charlie create message filter to node by group chat topic.
	charlieGroupChatMessageFilterID := s.createGroupChatMessageFilter(charlieRPCClient, charlieGroupChatSymkeyID, charlieGroupChatData.Topic)

	// Alice send message to group chat.
	helloWorldMessage := hexutil.Encode([]byte("Hello world!"))
	groupChatMessageHash := s.postMessageToGroup(aliceRPCClient, groupChatKeyID, groupChatTopic.String(), helloWorldMessage)

	// Bob receive group chat message.
	s.waitForEnvelopeEvents(bobEnvelopeAvailableWatcher, []string{groupChatMessageHash}, whisper.EventEnvelopeAvailable)
	messages = s.getMessagesByMessageFilterID(bobRPCClient, bobGroupChatMessageFilterID)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))

	// Charlie receive group chat message.
	s.waitForEnvelopeEvents(charlieEnvelopeAvailableWatcher, []string{groupChatMessageHash}, whisper.EventEnvelopeAvailable)
	messages = s.getMessagesByMessageFilterID(charlieRPCClient, charlieGroupChatMessageFilterID)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))

	// Check that we don't receive messages each one time.
	messages = s.getMessagesByMessageFilterID(bobRPCClient, bobGroupChatMessageFilterID)
	s.Require().Empty(messages)
	messages = s.getMessagesByMessageFilterID(charlieRPCClient, charlieGroupChatMessageFilterID)
	s.Require().Empty(messages)

	// be sure that message has been archived
	s.waitForEnvelopeEvents(envelopeArchivedWatcher, []string{groupChatMessageHash}, whisper.EventMailServerEnvelopeArchived)

	// Request each one messages from mailbox using enode.
	s.requestHistoricMessagesFromLast12Hours(bobWhisperService, bobRPCClient, mailboxEnode, bobMailServerKeyID, groupChatTopic.String(), 0, "")
	s.requestHistoricMessagesFromLast12Hours(charlieWhisperService, charlieRPCClient, mailboxEnode, charlieMailServerKeyID, groupChatTopic.String(), 0, "")

	// Bob receive p2p message from group chat filter.
	s.waitForEnvelopeEvents(bobEnvelopeAvailableWatcher, []string{groupChatMessageHash}, whisper.EventEnvelopeAvailable)
	messages = s.getMessagesByMessageFilterID(bobRPCClient, bobGroupChatMessageFilterID)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))

	// Charlie receive p2p message from group chat filter.
	s.waitForEnvelopeEvents(charlieEnvelopeAvailableWatcher, []string{groupChatMessageHash}, whisper.EventEnvelopeAvailable)
	messages = s.getMessagesByMessageFilterID(charlieRPCClient, charlieGroupChatMessageFilterID)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))
}

func (s *WhisperMailboxSuite) TestRequestMessagesWithPagination() {
	// Start mailbox
	mailbox, stop := s.startMailboxBackend()
	defer stop()
	s.Require().True(mailbox.IsNodeRunning())
	mailboxEnode := mailbox.StatusNode().GethNode().Server().NodeInfo().Enode

	// Start client
	client, stop := s.startBackend("client")
	defer stop()
	s.Require().True(client.IsNodeRunning())
	clientRPCClient := client.StatusNode().RPCClient()

	// Add mailbox to clients's peers
	s.addPeerAndWait(client.StatusNode(), mailbox.StatusNode())

	// Whisper services
	mailboxWhisperService, err := mailbox.StatusNode().WhisperService()
	s.Require().NoError(err)
	clientWhisperService, err := client.StatusNode().WhisperService()
	s.Require().NoError(err)
	// mailserver sym key
	mailServerKeyID, err := clientWhisperService.AddSymKeyFromPassword(mailboxPassword)
	s.Require().NoError(err)

	// public chat
	var (
		keyID    string
		topic    whisper.TopicType
		filterID string
	)
	publicChatName := "test public chat"
	keyID, topic, filterID = s.joinPublicChat(clientWhisperService, clientRPCClient, publicChatName)

	// envelopes to be sent
	envelopesCount := 5
	sentEnvelopesHashes := make([]string, 0)

	// watch envelopes to be archived on mailserver
	envelopeArchivedWatcher := make(chan whisper.EnvelopeEvent, 1024)
	mailboxWhisperService.SubscribeEnvelopeEvents(envelopeArchivedWatcher)

	// watch envelopes to be available for filters in the client
	envelopeAvailableWatcher := make(chan whisper.EnvelopeEvent, 1024)
	clientWhisperService.SubscribeEnvelopeEvents(envelopeAvailableWatcher)

	// watch mailserver responses in the client
	mailServerResponseWatcher := make(chan whisper.EnvelopeEvent, 1024)
	clientWhisperService.SubscribeEnvelopeEvents(mailServerResponseWatcher)

	// send envelopes
	for i := 0; i < envelopesCount; i++ {
		hash := s.postMessageToGroup(clientRPCClient, keyID, topic.String(), "")
		sentEnvelopesHashes = append(sentEnvelopesHashes, hash)
	}

	// get messages from filter before requesting them to mailserver
	lastEnvelopeHash := sentEnvelopesHashes[len(sentEnvelopesHashes)-1]
	s.waitForEnvelopeEvents(envelopeAvailableWatcher, []string{lastEnvelopeHash}, whisper.EventEnvelopeAvailable)
	messages := s.getMessagesByMessageFilterID(clientRPCClient, filterID)
	s.Equal(envelopesCount, len(messages))

	messages = s.getMessagesByMessageFilterID(clientRPCClient, filterID)
	s.Equal(0, len(messages))

	limit := 3

	getMessages := func() []string {
		envelopes := s.getMessagesByMessageFilterID(clientRPCClient, filterID)
		hashes := make([]string, 0)
		for _, e := range envelopes {
			hashes = append(hashes, e["hash"].(string))
		}

		return hashes
	}

	requestMessages := func(cursor string) common.Hash {
		return s.requestHistoricMessagesFromLast12Hours(clientWhisperService, clientRPCClient, mailboxEnode, mailServerKeyID, topic.String(), limit, cursor)
	}

	// wait for mailserver to archive all the envelopes
	s.waitForEnvelopeEvents(envelopeArchivedWatcher, sentEnvelopesHashes, whisper.EventMailServerEnvelopeArchived)

	// first page
	// send request
	requestID := requestMessages("")
	// wait for mail server response
	resp := s.waitForMailServerResponse(mailServerResponseWatcher, requestID)
	s.NotEmpty(resp.LastEnvelopeHash)
	s.NotEmpty(resp.Cursor)
	// wait for last envelope sent by the mailserver to be available for filters
	s.waitForEnvelopeEvents(envelopeAvailableWatcher, []string{resp.LastEnvelopeHash.String()}, whisper.EventEnvelopeAvailable)
	// get messages
	firstPageHashes := getMessages()
	s.Equal(3, len(firstPageHashes))

	// second page
	// send request
	requestID = requestMessages(fmt.Sprintf("%x", resp.Cursor))
	// wait for mail server response
	resp = s.waitForMailServerResponse(mailServerResponseWatcher, requestID)
	s.NotEmpty(resp.LastEnvelopeHash)
	// all messages have been sent, no more pages available
	s.Empty(resp.Cursor)
	// wait for last envelope sent by the mailserver to be available for filters
	s.waitForEnvelopeEvents(envelopeAvailableWatcher, []string{resp.LastEnvelopeHash.String()}, whisper.EventEnvelopeAvailable)
	// get messages
	secondPageHashes := getMessages()
	s.Equal(2, len(secondPageHashes))

	allReceivedHashes := append(firstPageHashes, secondPageHashes...)
	s.Equal(envelopesCount, len(allReceivedHashes))

	// check that all the envelopes have been received
	sort.Strings(sentEnvelopesHashes)
	sort.Strings(allReceivedHashes)
	s.Equal(sentEnvelopesHashes, allReceivedHashes)
}

func (s *WhisperMailboxSuite) waitForEnvelopeEvents(events chan whisper.EnvelopeEvent, hashes []string, event whisper.EventType) {
	check := make(map[string]struct{})
	for _, hash := range hashes {
		check[hash] = struct{}{}
	}

	timeout := time.NewTimer(time.Second * 5)
	for {
		select {
		case e := <-events:
			if e.Event == event {
				delete(check, e.Hash.String())
				if len(check) == 0 {
					timeout.Stop()
					return
				}
			}
		case <-timeout.C:
			s.FailNow("timed out while waiting for event on envelopes", "event: %s", event)
		}
	}
}

func (s *WhisperMailboxSuite) waitForMailServerResponse(events chan whisper.EnvelopeEvent, requestID common.Hash) *whisper.MailServerResponse {
	timeout := time.NewTimer(time.Second * 5)
	for {
		select {
		case event := <-events:
			if event.Event == whisper.EventMailServerRequestCompleted && event.Hash == requestID {
				timeout.Stop()
				resp, ok := event.Data.(*whisper.MailServerResponse)
				if !ok {
					s.FailNow("mailserver response error", "expected whisper.MailServerResponse, got: %+v", resp)
				}

				return resp
			}
		case <-timeout.C:
			s.FailNow("timed out while waiting for mailserver response")
		}
	}
}

func (s *WhisperMailboxSuite) addPeerAndWait(node, other *node.StatusNode) {
	nodeInfo := node.GethNode().Server().NodeInfo()
	nodeID := nodeInfo.ID
	nodeEnode := nodeInfo.Enode
	otherEnode := other.GethNode().Server().NodeInfo().Enode
	s.Require().NotEqual(nodeEnode, otherEnode)

	ch := make(chan *p2p.PeerEvent)
	subscription := other.GethNode().Server().SubscribeEvents(ch)
	defer subscription.Unsubscribe()

	err := node.AddPeer(otherEnode)
	s.Require().NoError(err)

	select {
	case event := <-ch:
		if event.Type == p2p.PeerEventTypeAdd && event.Peer.String() == nodeID {
			return
		}

		s.Failf("failed connecting to peer", "expected p2p.PeerEventTypeAdd with nodeID (%s), got: %+v", nodeID, event)
	case <-time.After(time.Second):
		s.Fail("timed out while waiting for a peer to be added")
	}
}

func newGroupChatParams(symkey []byte, topic whisper.TopicType) groupChatParams {
	groupChatKeyStr := hexutil.Bytes(symkey).String()
	return groupChatParams{
		Key:   groupChatKeyStr,
		Topic: topic.String(),
	}
}

type groupChatParams struct {
	Key   string
	Topic string
}

func (d *groupChatParams) Decode(i string) error {
	b, err := hexutil.Decode(i)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &d)
}

func (d *groupChatParams) Encode() (string, error) {
	payload, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return hexutil.Bytes(payload).String(), nil
}

// Start status node.
func (s *WhisperMailboxSuite) startBackend(name string) (*api.StatusBackend, func()) {
	datadir := filepath.Join(RootDir, ".ethereumtest/mailbox", name)
	backend := api.NewStatusBackend()
	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	nodeConfig.DataDir = datadir
	s.Require().NoError(err)
	s.Require().False(backend.IsNodeRunning())

	nodeConfig.WhisperConfig.LightClient = true

	if addr, err := GetRemoteURL(); err == nil {
		nodeConfig.UpstreamConfig.Enabled = true
		nodeConfig.UpstreamConfig.URL = addr
	}

	s.Require().NoError(backend.StartNode(nodeConfig))
	s.Require().True(backend.IsNodeRunning())

	return backend, func() {
		s.True(backend.IsNodeRunning())
		s.NoError(backend.StopNode())
		s.False(backend.IsNodeRunning())
		err = os.RemoveAll(datadir)
		s.Require().NoError(err)
	}

}

// Start mailbox node.
func (s *WhisperMailboxSuite) startMailboxBackend() (*api.StatusBackend, func()) {
	mailboxBackend := api.NewStatusBackend()
	mailboxConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.Require().NoError(err)
	datadir := filepath.Join(RootDir, ".ethereumtest/mailbox/mailserver")

	mailboxConfig.LightEthConfig.Enabled = false
	mailboxConfig.WhisperConfig.Enabled = true
	mailboxConfig.KeyStoreDir = datadir
	mailboxConfig.WhisperConfig.EnableMailServer = true
	mailboxConfig.WhisperConfig.PasswordFile = filepath.Join(RootDir, "/static/keys/wnodepassword")
	mailboxConfig.WhisperConfig.DataDir = filepath.Join(datadir, "data")
	mailboxConfig.DataDir = datadir

	s.Require().False(mailboxBackend.IsNodeRunning())
	s.Require().NoError(mailboxBackend.StartNode(mailboxConfig))
	s.Require().True(mailboxBackend.IsNodeRunning())
	return mailboxBackend, func() {
		s.True(mailboxBackend.IsNodeRunning())
		s.NoError(mailboxBackend.StopNode())
		s.False(mailboxBackend.IsNodeRunning())
		err = os.RemoveAll(datadir)
		s.Require().NoError(err)
	}
}

// createPrivateChatMessageFilter create message filter with asymmetric encryption.
func (s *WhisperMailboxSuite) createPrivateChatMessageFilter(rpcCli *rpc.Client, privateKeyID string, topic string) string {
	resp := rpcCli.CallRaw(`{
			"jsonrpc": "2.0",
			"method": "shh_newMessageFilter", "params": [
				{"privateKeyID": "` + privateKeyID + `", "topics": [ "` + topic + `"], "allowP2P":true}
			],
			"id": 1
		}`)

	msgFilterResp := returnedIDResponse{}
	err := json.Unmarshal([]byte(resp), &msgFilterResp)
	messageFilterID := msgFilterResp.Result
	s.Require().NoError(err)
	s.Require().Nil(msgFilterResp.Error)
	s.Require().NotEqual("", messageFilterID, resp)
	return messageFilterID
}

// createGroupChatMessageFilter create message filter with symmetric encryption.
func (s *WhisperMailboxSuite) createGroupChatMessageFilter(rpcCli *rpc.Client, symkeyID string, topic string) string {
	resp := rpcCli.CallRaw(`{
			"jsonrpc": "2.0",
			"method": "shh_newMessageFilter", "params": [
				{"symKeyID": "` + symkeyID + `", "topics": [ "` + topic + `"], "allowP2P":true}
			],
			"id": 1
		}`)

	msgFilterResp := returnedIDResponse{}
	err := json.Unmarshal([]byte(resp), &msgFilterResp)
	messageFilterID := msgFilterResp.Result
	s.Require().NoError(err)
	s.Require().Nil(msgFilterResp.Error)
	s.Require().NotEqual("", messageFilterID, resp)
	return messageFilterID
}

func (s *WhisperMailboxSuite) postMessageToPrivate(rpcCli *rpc.Client, recipientPubkey string, topic string, payload string) string {
	resp := rpcCli.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"pubKey": "` + recipientPubkey + `",
			"topic": "` + topic + `",
			"payload": "` + payload + `",
			"powTarget": 0.001,
			"powTime": 2
			}
		],
		"id": 1}`)
	postResp := baseRPCResponse{}
	err := json.Unmarshal([]byte(resp), &postResp)
	s.Require().NoError(err)
	s.Require().Nil(postResp.Error)

	return postResp.Result.(string)
}

func (s *WhisperMailboxSuite) postMessageToGroup(rpcCli *rpc.Client, groupChatKeyID string, topic string, payload string) string {
	resp := rpcCli.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"symKeyID": "` + groupChatKeyID + `",
			"topic": "` + topic + `",
			"payload": "` + payload + `",
			"powTarget": 0.001,
			"powTime": 2
			}
		],
		"id": 1}`)
	postResp := baseRPCResponse{}
	err := json.Unmarshal([]byte(resp), &postResp)
	s.Require().NoError(err)
	s.Require().Nil(postResp.Error)

	hash, ok := postResp.Result.(string)
	if !ok {
		s.FailNow("error decoding result", "expected string, got: %+v", postResp.Result)
	}

	if !strings.HasPrefix(hash, "0x") {
		s.FailNow("hash format error", "expected hex string, got: %s", hash)
	}

	return hash
}

// getMessagesByMessageFilterID gets received messages by messageFilterID.
func (s *WhisperMailboxSuite) getMessagesByMessageFilterID(rpcCli *rpc.Client, messageFilterID string) []map[string]interface{} {
	resp := rpcCli.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	messages := getFilterMessagesResponse{}
	err := json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Nil(messages.Error)
	return messages.Result
}

// addSymKey added symkey to node and return symkeyID.
func (s *WhisperMailboxSuite) addSymKey(rpcCli *rpc.Client, symkey string) string {
	resp := rpcCli.CallRaw(`{"jsonrpc":"2.0","method":"shh_addSymKey",
			"params":["` + symkey + `"],
			"id":1}`)
	symkeyAddResp := returnedIDResponse{}
	err := json.Unmarshal([]byte(resp), &symkeyAddResp)
	s.Require().NoError(err)
	s.Require().Nil(symkeyAddResp.Error)
	symkeyID := symkeyAddResp.Result
	s.Require().NotEmpty(symkeyID)
	return symkeyID
}

// requestHistoricMessagesFromLast12Hours asks a mailnode to resend messages from last 12 hours.
func (s *WhisperMailboxSuite) requestHistoricMessagesFromLast12Hours(w *whisper.Whisper, rpcCli *rpc.Client, mailboxEnode, mailServerKeyID, topic string, limit int, cursor string) common.Hash {
	currentTime := w.GetCurrentTime()
	from := currentTime.Add(-12 * time.Hour)
	to := currentTime
	return s.requestHistoricMessages(w, rpcCli, mailboxEnode, mailServerKeyID, topic, from, to, limit, cursor)
}

// requestHistoricMessages asks a mailnode to resend messages.
func (s *WhisperMailboxSuite) requestHistoricMessages(w *whisper.Whisper, rpcCli *rpc.Client, mailboxEnode, mailServerKeyID, topic string, from, to time.Time, limit int, cursor string) common.Hash {
	resp := rpcCli.CallRaw(`{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "shhext_requestMessages",
		"params": [{
					"mailServerPeer":"` + mailboxEnode + `",
					"topic":"` + topic + `",
					"symKeyID":"` + mailServerKeyID + `",
					"from":` + strconv.FormatInt(from.Unix(), 10) + `,
					"to":` + strconv.FormatInt(to.Unix(), 10) + `,
					"limit": ` + fmt.Sprintf("%d", limit) + `,
					"cursor": "` + cursor + `"
		}]
	}`)
	reqMessagesResp := baseRPCResponse{}
	err := json.Unmarshal([]byte(resp), &reqMessagesResp)
	s.Require().NoError(err)
	s.Require().Nil(reqMessagesResp.Error)

	switch hash := reqMessagesResp.Result.(type) {
	case string:
		s.Require().True(strings.HasPrefix(hash, "0x"))
		b, err := hex.DecodeString(hash[2:])
		s.Require().NoError(err)
		return common.BytesToHash(b)
	default:
		s.Failf("failed reading shh_newMessageFilter result", "expected a hash, got: %+v", reqMessagesResp.Result)
	}

	return common.Hash{}
}

func (s *WhisperMailboxSuite) joinPublicChat(w *whisper.Whisper, rpcClient *rpc.Client, name string) (string, whisper.TopicType, string) {
	keyID, err := w.AddSymKeyFromPassword(name)
	s.Require().NoError(err)

	h := sha3.NewKeccak256()
	_, err = h.Write([]byte(name))
	if err != nil {
		s.Fail("error generating topic", "failed gerating topic from chat name, %+v", err)
	}
	fullTopic := h.Sum(nil)
	topic := whisper.BytesToTopic(fullTopic)

	filterID := s.createGroupChatMessageFilter(rpcClient, keyID, topic.String())

	return keyID, topic, filterID
}

type getFilterMessagesResponse struct {
	Result []map[string]interface{}
	Error  interface{}
}

type returnedIDResponse struct {
	Result string
	Error  interface{}
}
type baseRPCResponse struct {
	Result interface{}
	Error  interface{}
}
