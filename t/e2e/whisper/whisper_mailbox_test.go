package whisper

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
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
	// Wait async processes on adding peer.
	time.Sleep(500 * time.Millisecond)

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
	mailboxTracer := newTracer()
	mailboxWhisperService.RegisterEnvelopeTracer(mailboxTracer)

	tracer := newTracer()
	senderWhisperService.RegisterEnvelopeTracer(tracer)

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
	messages = s.getMessagesByMessageFilterIDWithTracer(rpcClient, messageFilterID, mailboxTracer, messageHash)
	s.Require().Equal(1, len(messages))

	// Act.

	events := make(chan whisper.EnvelopeEvent)
	senderWhisperService.SubscribeEnvelopeEvents(events)

	// Request messages (including the previous one, expired) from mailbox.
	result := s.requestHistoricMessages(senderWhisperService, rpcClient, mailboxPeerStr, MailServerKeyID, topic.String(), 0, "")
	requestID := common.BytesToHash(result)

	// And we receive message, it comes from mailbox.
	messages = s.getMessagesByMessageFilterIDWithTracer(rpcClient, messageFilterID, tracer, messageHash)
	s.Require().Equal(1, len(messages))

	// Check that there are no messages.
	messages = s.getMessagesByMessageFilterID(rpcClient, messageFilterID)
	s.Require().Empty(messages)

	select {
	case e := <-events:
		s.Equal(whisper.EventMailServerRequestCompleted, e.Event)
		s.Equal(requestID, e.Hash)
	case <-time.After(time.Second):
		s.Fail("timed out while waiting for request completed event")
	}
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
	err = bobBackend.StatusNode().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	err = charlieBackend.StatusNode().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	// Wait async processes on adding peer.
	time.Sleep(500 * time.Millisecond)

	// Get whisper service.
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

	aliceTracer := newTracer()
	aliceWhisperService.RegisterEnvelopeTracer(aliceTracer)
	bobTracer := newTracer()
	bobWhisperService.RegisterEnvelopeTracer(bobTracer)
	charlieTracer := newTracer()
	charlieWhisperService.RegisterEnvelopeTracer(charlieTracer)

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
	messages := s.getMessagesByMessageFilterIDWithTracer(bobRPCClient, bobMessageFilterID, bobTracer, aliceToBobMessageHash)
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
	messages = s.getMessagesByMessageFilterIDWithTracer(charlieRPCClient, charlieMessageFilterID, charlieTracer, aliceToCharlieMessageHash)
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
	messages = s.getMessagesByMessageFilterIDWithTracer(bobRPCClient, bobGroupChatMessageFilterID, bobTracer, groupChatMessageHash)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))

	// Charlie receive group chat message.
	messages = s.getMessagesByMessageFilterIDWithTracer(charlieRPCClient, charlieGroupChatMessageFilterID, charlieTracer, groupChatMessageHash)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))

	// Check that we don't receive messages each one time.
	messages = s.getMessagesByMessageFilterID(bobRPCClient, bobGroupChatMessageFilterID)
	s.Require().Empty(messages)
	messages = s.getMessagesByMessageFilterID(charlieRPCClient, charlieGroupChatMessageFilterID)
	s.Require().Empty(messages)

	// Request each one messages from mailbox using enode.
	s.requestHistoricMessages(bobWhisperService, bobRPCClient, mailboxEnode, bobMailServerKeyID, groupChatTopic.String(), 0, "")
	s.requestHistoricMessages(charlieWhisperService, charlieRPCClient, mailboxEnode, charlieMailServerKeyID, groupChatTopic.String(), 0, "")

	// Bob receive p2p message from group chat filter.
	messages = s.getMessagesByMessageFilterIDWithTracer(bobRPCClient, bobGroupChatMessageFilterID, bobTracer, groupChatMessageHash)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))

	// Charlie receive p2p message from group chat filter.
	messages = s.getMessagesByMessageFilterIDWithTracer(charlieRPCClient, charlieGroupChatMessageFilterID, charlieTracer, groupChatMessageHash)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))
}

func (s *WhisperMailboxSuite) TestRequestMessagesWithPagination() {
	// Start mailbox and status node.
	mailbox, stop := s.startMailboxBackend()
	defer stop()
	s.Require().True(mailbox.IsNodeRunning())

	// sender
	sender, stop := s.startBackend("sender")
	defer stop()
	s.Require().True(sender.IsNodeRunning())

	// Add mailbox to sender's peers
	s.addPeerAndWait(sender.StatusNode(), mailbox.StatusNode())

	// Whisper service
	mailboxWhisperService, err := mailbox.StatusNode().WhisperService()
	s.Require().NoError(err)
	senderWhisperService, err := sender.StatusNode().WhisperService()
	s.Require().NoError(err)

	senderRPCClient := sender.StatusNode().RPCClient()

	// public chat
	var (
		keyID    string
		topic    whisper.TopicType
		filterID string
	)
	publicChatName := "test public chat"
	keyID, topic, _ = s.joinPublicChat(senderWhisperService, senderRPCClient, publicChatName)

	envelopesCount := 5

	// watch mailserver envelopeFeed
	mailboxEvents := make(chan whisper.EnvelopeEvent, envelopesCount)
	mailboxWhisperService.SubscribeEnvelopeEvents(mailboxEvents)

	type check struct {
		archived  bool
		retrieved bool
	}
	sentEnvelopes := make(map[string]*check)

	// send envelopes
	for i := 0; i < envelopesCount; i++ {
		hash := s.postMessageToGroup(senderRPCClient, keyID, topic.String(), "")
		sentEnvelopes[hash] = &check{}
	}

	// wait for envelopes to be archived
	for i := 0; i < envelopesCount; i++ {
		select {
		case event := <-mailboxEvents:
			if event.Event != whisper.EventMailServerEnvelopeArchived {
				s.FailNow("error archiving", "expected envelope archived event, got: %v", event)
			}
			check, found := sentEnvelopes[event.Hash.String()]
			if !found {
				s.FailNow("error archiving", "archived envelope is not in the sent envelopes")
			}
			check.archived = true
		case <-time.After(5 * time.Second):
			s.FailNow("timed out while waiting for an envelope to be archived")
		}
	}

	// check that all envelopes have been archived
	for hash, check := range sentEnvelopes {
		if !check.archived {
			s.FailNow("error archiving", "envelope %x has not been archived", hash)
		}
	}

	// receiver
	receiver, stop := s.startBackend("receiver")
	defer stop()
	s.Require().True(receiver.IsNodeRunning())
	receiverWhisperService, err := receiver.StatusNode().WhisperService()
	s.Require().NoError(err)
	// Add mailbox to receiver's peers
	s.addPeerAndWait(receiver.StatusNode(), mailbox.StatusNode())
	receiverRPCClient := receiver.StatusNode().RPCClient()
	// public chat
	_, topic, filterID = s.joinPublicChat(receiverWhisperService, receiverRPCClient, publicChatName)

	// watch receiver envelopeFeed
	receiverEvents := make(chan whisper.EnvelopeEvent, envelopesCount)
	receiverWhisperService.SubscribeEnvelopeEvents(receiverEvents)

	// request historic messages
	mailServerKeyID, err := receiverWhisperService.AddSymKeyFromPassword(mailboxPassword)
	s.Require().NoError(err)
	mailboxEnode := mailbox.StatusNode().GethNode().Server().NodeInfo().Enode

	// first page
	limit := 3
	s.requestHistoricMessages(receiverWhisperService, receiverRPCClient, mailboxEnode, mailServerKeyID, topic.String(), limit, "")
	cursor := s.waitForMailServerResponse(receiverEvents)
	messages := s.getMessagesByMessageFilterID(receiverRPCClient, filterID)
	s.Equal(3, len(messages))
	s.NotEmpty(cursor)

	// second page
	cursorHex := fmt.Sprintf("%x", cursor)
	s.requestHistoricMessages(receiverWhisperService, receiverRPCClient, mailboxEnode, mailServerKeyID, topic.String(), limit, cursorHex)
	cursor = s.waitForMailServerResponse(receiverEvents)
	messages = s.getMessagesByMessageFilterID(receiverRPCClient, filterID)
	s.Equal(2, len(messages))
	s.Empty(cursor)
}

func (s *WhisperMailboxSuite) waitForMailServerResponse(events chan whisper.EnvelopeEvent) []byte {
	select {
	case event := <-events:
		if event.Event != whisper.EventMailServerRequestCompleted {
			s.FailNow("error mailserver response", "expected to receive mailserver response, got: %+v", event)
		}

		cursor, ok := event.Data.([]byte)
		if !ok {
			s.FailNow("cursor error", "expected cursor to be []byte, got: %+v", event.Data)
		}

		return cursor
	case <-time.After(5 * time.Second):
		s.FailNow("timed out while waiting for mailserver reponse")
	}

	return nil
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

func (s *WhisperMailboxSuite) postMessageToPrivate(rpcCli *rpc.Client, bobPubkey string, topic string, payload string) string {
	resp := rpcCli.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"pubKey": "` + bobPubkey + `",
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

func (s *WhisperMailboxSuite) getMessagesByMessageFilterIDWithTracer(rpcCli *rpc.Client, messageFilterID string, tracer *envelopeTracer, messageHash string) (messages []map[string]interface{}) {
	select {
	case envelope := <-tracer.envelopChan:
		s.Require().Equal(envelope.Hash, messageHash)
	case <-time.After(5 * time.Second):
		s.Fail("Timed out waiting for new messages after 5 seconds")
	}

	// Attempt to retrieve messages up to 3 times, 1 second apart
	// TODO: There is a lag between the time when the envelope is traced by EventTracer
	// and when it is decoded and actually available. Ideally this would be event-driven as well
	// I.e. instead of signing up for EnvelopeTracer, we'd sign up for an event happening later
	// which tells us that a call to shh_getFilterMessages will return some messages
	for i := 0; i < 3; i++ {
		messages = s.getMessagesByMessageFilterID(rpcCli, messageFilterID)
		if len(messages) > 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	return
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

// requestHistoricMessages asks a mailnode to resend messages.
func (s *WhisperMailboxSuite) requestHistoricMessages(w *whisper.Whisper, rpcCli *rpc.Client, mailboxEnode, mailServerKeyID, topic string, limit int, cursor string) []byte {
	currentTime := w.GetCurrentTime()
	from := currentTime.Add(-12 * time.Hour)
	to := currentTime
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
		return b
	default:
		s.Failf("failed reading shh_newMessageFilter result", "expected a hash, got: %+v", reqMessagesResp.Result)
	}

	return nil
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

// envelopeTracer traces incoming envelopes. We leverage it to know when a peer has received an envelope
// so we rely less on timeouts for the tests
type envelopeTracer struct {
	envelopChan chan *whisper.EnvelopeMeta
}

func newTracer() *envelopeTracer {
	return &envelopeTracer{make(chan *whisper.EnvelopeMeta, 1)}
}

// Trace is called for every incoming envelope.
func (t *envelopeTracer) Trace(envelope *whisper.EnvelopeMeta) {
	// Do not block notifier
	go func() { t.envelopChan <- envelope }()
}
