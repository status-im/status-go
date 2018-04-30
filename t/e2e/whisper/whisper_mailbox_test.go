package whisper

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"os"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/rpc"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

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
	time.Sleep(time.Second)

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
	password := "status-offline-inbox"
	MailServerKeyID, err := senderWhisperService.AddSymKeyFromPassword(password)
	s.Require().NoError(err)

	rpcClient := sender.StatusNode().RPCClient()
	s.Require().NotNil(rpcClient)

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
	s.Require().Equal(0, len(messages))

	// Post message matching with filter (key and topic).
	s.postMessageToPrivate(rpcClient, pubkey.String(), topic.String(), hexutil.Encode([]byte("Hello world!")))

	// Get message to make sure that it will come from the mailbox later.
	time.Sleep(1 * time.Second)
	messages = s.getMessagesByMessageFilterID(rpcClient, messageFilterID)
	s.Require().Equal(1, len(messages))

	// Act.

	// Request messages (including the previous one, expired) from mailbox.
	reqMessagesBody := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "shhext_requestMessages",
		"params": [{
					"mailServerPeer":"` + mailboxPeerStr + `",
					"topic":"` + topic.String() + `",
					"symKeyID":"` + MailServerKeyID + `",
					"from":0,
					"to":` + strconv.FormatInt(senderWhisperService.GetCurrentTime().Unix(), 10) + `
		}]
	}`
	resp := rpcClient.CallRaw(reqMessagesBody)
	reqMessagesResp := baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &reqMessagesResp)
	s.Require().NoError(err)
	s.Require().Nil(reqMessagesResp.Error)

	// Wait to receive message.
	time.Sleep(time.Second)
	// And we receive message, it comes from mailbox.
	messages = s.getMessagesByMessageFilterID(rpcClient, messageFilterID)
	s.Require().Equal(1, len(messages))

	// Check that there are no messages.
	messages = s.getMessagesByMessageFilterID(rpcClient, messageFilterID)
	s.Require().Equal(0, len(messages))
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
	time.Sleep(time.Second)

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

	// Bob and charlie add the mailserver key.
	password := "status-offline-inbox"
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

	// Bob and charlie create message filter.
	bobMessageFilterID := s.createPrivateChatMessageFilter(bobRPCClient, bobKeyID, bobAliceKeySendTopic.String())
	charlieMessageFilterID := s.createPrivateChatMessageFilter(charlieRPCClient, charlieKeyID, charlieAliceKeySendTopic.String())

	// Alice send message with symkey and topic to Bob and Charlie.
	s.postMessageToPrivate(aliceRPCClient, bobPubkey.String(), bobAliceKeySendTopic.String(), payloadStr)
	s.postMessageToPrivate(aliceRPCClient, charliePubkey.String(), charlieAliceKeySendTopic.String(), payloadStr)

	// Wait to receive.
	time.Sleep(5 * time.Second)

	// Bob receive group chat data and add it to his node.
	// Bob get group chat details.
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
	s.postMessageToGroup(aliceRPCClient, groupChatKeyID, groupChatTopic.String(), helloWorldMessage)
	// It need to receive envelopes by bob and charlie nodes.
	time.Sleep(5 * time.Second)

	// Bob receive group chat message.
	messages = s.getMessagesByMessageFilterID(bobRPCClient, bobGroupChatMessageFilterID)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))

	// Charlie receive group chat message.
	messages = s.getMessagesByMessageFilterID(charlieRPCClient, charlieGroupChatMessageFilterID)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))

	// Check that we don't receive messages each one time.
	messages = s.getMessagesByMessageFilterID(bobRPCClient, bobGroupChatMessageFilterID)
	s.Require().Equal(0, len(messages))
	messages = s.getMessagesByMessageFilterID(charlieRPCClient, charlieGroupChatMessageFilterID)
	s.Require().Equal(0, len(messages))

	// Request each one messages from mailbox using enode.
	s.requestHistoricMessages(bobWhisperService, bobRPCClient, mailboxEnode, bobMailServerKeyID, groupChatTopic.String())
	s.requestHistoricMessages(charlieWhisperService, charlieRPCClient, mailboxEnode, charlieMailServerKeyID, groupChatTopic.String())
	// Wait to receive p2p messages.
	time.Sleep(5 * time.Second)

	// Bob receive p2p message from group chat filter.
	messages = s.getMessagesByMessageFilterID(bobRPCClient, bobGroupChatMessageFilterID)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))

	// Charlie receive p2p message from group chat filter.
	messages = s.getMessagesByMessageFilterID(charlieRPCClient, charlieGroupChatMessageFilterID)
	s.Require().Equal(1, len(messages))
	s.Require().Equal(helloWorldMessage, messages[0]["payload"].(string))
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

func (s *WhisperMailboxSuite) postMessageToPrivate(rpcCli *rpc.Client, bobPubkey string, topic string, payload string) {
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
}

func (s *WhisperMailboxSuite) postMessageToGroup(rpcCli *rpc.Client, groupChatKeyID string, topic string, payload string) {
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

// requestHistoricMessages asks a mailnode to resend messages.
func (s *WhisperMailboxSuite) requestHistoricMessages(w *whisper.Whisper, rpcCli *rpc.Client, mailboxEnode, mailServerKeyID, topic string) {
	resp := rpcCli.CallRaw(`{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "shhext_requestMessages",
		"params": [{
					"mailServerPeer":"` + mailboxEnode + `",
					"topic":"` + topic + `",
					"symKeyID":"` + mailServerKeyID + `",
					"from":0,
					"to":` + strconv.FormatInt(w.GetCurrentTime().Unix(), 10) + `
		}]
	}`)
	reqMessagesResp := baseRPCResponse{}
	err := json.Unmarshal([]byte(resp), &reqMessagesResp)
	s.Require().NoError(err)
	s.Require().Nil(reqMessagesResp.Error)
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
