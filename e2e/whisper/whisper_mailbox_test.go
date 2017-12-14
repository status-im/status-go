package whisper

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/suite"
)

func TestMailboxSuite(t *testing.T) {
	suite.Run(t, new(MailboxSuite))
}

type MailboxSuite struct {
	e2e.BackendTestSuite
	MailboxBackend *api.StatusBackend
	SymKey         []byte
	Topic          whisperv5.TopicType
}

func (s *MailboxSuite) SetupTest() {
	s.Topic = whisperv5.BytesToTopic([]byte("test-topic"))
	s.BackendTestSuite.SetupTest()
}

func (s *MailboxSuite) startMailboxTestBackend() {
	s.T().Log("startMailboxTestBackend")

	require := s.Require()

	s.MailboxBackend = api.NewStatusBackend()

	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	require.NoError(err)

	nodeConfig.LightEthConfig.Enabled = false
	nodeConfig.HTTPPort = TestConfig.Node.HTTPPort + 1
	nodeConfig.DataDir = filepath.Join(TestDataDir, "mailbox")
	nodeConfig.LogLevel = "INFO"
	nodeConfig.LogFile = filepath.Join(TestDataDir, "mailbox", "wnode.log")
	nodeConfig.WhisperConfig.Enabled = true
	nodeConfig.WhisperConfig.EnableMailServer = true
	nodeConfig.WhisperConfig.IdentityFile = "../../static/keys/wnodekey"
	nodeConfig.WhisperConfig.PasswordFile = "../../static/keys/wnodepassword"
	nodeConfig.WhisperConfig.DataDir = filepath.Join(TestDataDir, "mailbox", "wnode-data")

	nodeStarted, err := s.MailboxBackend.StartNode(nodeConfig)
	require.NoError(err)
	<-nodeStarted
}

func (s *MailboxSuite) stopMailboxTestBackend() {
	s.T().Log("stopMailboxTestBackend")

	s.True(s.MailboxBackend.IsNodeRunning())
	backendStopped, err := s.MailboxBackend.StopNode()
	s.NoError(err)
	<-backendStopped
}

func (s *MailboxSuite) sendMessageFromNodeA() {
	require := s.Require()

	// Add Mailbox as a peer.
	mailbox, err := s.MailboxBackend.NodeManager().Node()
	require.NoError(err)
	mailboxEnode := mailbox.Server().NodeInfo().Enode
	require.NoError(s.Backend.NodeManager().AddPeer(mailboxEnode))

	// AddPeer is async and requires some time to propagate.
	time.Sleep(time.Second)

	// Get Whisper service.
	shh, err := s.Backend.NodeManager().WhisperService()
	require.NoError(err)

	// Create a symmetric key for encrypting shh messages.
	symKeyID, err := shh.GenerateSymKey()
	require.NoError(err)

	// Cache the symmetric key.
	symKey, err := shh.GetSymKey(symKeyID)
	require.NoError(err)
	s.SymKey = symKey

	// Post a message.
	client := s.Backend.NodeManager().RPCClient()
	require.NotNil(client)

	var result bool
	require.NoError(client.Call(&result, "shh_post", map[string]interface{}{
		"topic":     s.Topic,
		"powTarget": 2.01,
		"powTime":   5.0,
		"symKeyID":  symKeyID,
		"ttl":       30,
	}))
	require.True(result)
}

func (s *MailboxSuite) requestMessagesFromMailboxByNodeB() {
	require := s.Require()

	shh, err := s.Backend.NodeManager().WhisperService()
	require.NoError(err)

	// Add Mailbox as a peer.
	mailbox, err := s.MailboxBackend.NodeManager().Node()
	require.NoError(err)
	mailboxEnode := mailbox.Server().NodeInfo().Enode
	require.NoError(s.Backend.NodeManager().AddPeer(mailboxEnode))

	s.T().Logf("Mailbox enode: %s", mailboxEnode)

	// AddPeer is async and requires some time to propagate.
	time.Sleep(time.Second)

	// Allow p2p messages from Mailbox.
	mailboxPeerID, err := discover.ParseNode(mailboxEnode)
	require.NoError(err)
	require.NoError(shh.AllowP2PMessagesFromPeer(mailboxPeerID.ID[:]))

	// Create a symmetric key from Mailbox password.
	mailboxSymKeyID, err := shh.AddSymKeyFromPassword("status-offline-inbox")
	require.NoError(err)

	// Restore the symmetric key.
	symKeyID, err := shh.AddSymKeyDirect(s.SymKey)
	require.NoError(err)

	client := s.Backend.NodeManager().RPCClient()
	require.NotNil(client)

	// Create a filter.
	var filterID string
	require.NoError(client.Call(&filterID, "shh_newMessageFilter", map[string]interface{}{
		"topic":    s.Topic,
		"symKeyID": symKeyID,
	}))

	// Get messages.
	var messages []interface{}
	require.NoError(client.Call(&messages, "shh_getFilterMessages", filterID))
	require.Len(messages, 0)

	// Request messages.
	var result bool
	require.NoError(client.Call(&result, "shh_requestMessages", map[string]interface{}{
		"enode":    mailboxEnode,
		"topic":    s.Topic,
		"symKeyID": mailboxSymKeyID,
		"from":     0,
		"to":       time.Now().UTC().Unix(),
	}))
	require.True(result)

	time.Sleep(time.Second * 5)

	// Get messages.
	require.NoError(client.Call(&messages, "shh_getFilterMessages", filterID))
	require.Len(messages, 1)
}

func (s *MailboxSuite) TestRequestMessagesFromMailbox() {
	s.startMailboxTestBackend()

	// NodeA sends a message.
	s.StartTestBackend(func(config *params.NodeConfig) {
		config.HTTPPort = TestConfig.Node.HTTPPort + 2
		config.DataDir = filepath.Join(TestDataDir, "node-a")
		config.LogLevel = "INFO"
		config.LogFile = filepath.Join(TestDataDir, "node-a", "wnode.log")
	})
	s.sendMessageFromNodeA()
	s.StopTestBackend()

	time.Sleep(time.Second * 20)

	// NodeB comes online and requests that message.
	s.StartTestBackend(func(config *params.NodeConfig) {
		config.HTTPPort = TestConfig.Node.HTTPPort + 3
		config.DataDir = filepath.Join(TestDataDir, "node-b")
		config.LogLevel = "INFO"
		config.LogFile = filepath.Join(TestDataDir, "node-b", "wnode.log")
	})
	s.requestMessagesFromMailboxByNodeB()
	s.StopTestBackend()

	s.stopMailboxTestBackend()
}

// func (s *MailboxSuite) TestRequestMessageFromMailboxAsync() {
// 	//arrange
// 	mailboxBackend, stop := s.startMailboxBackend()
// 	defer stop()
// 	mailboxNode, err := mailboxBackend.NodeManager().Node()
// 	s.Require().NoError(err)
// 	mailboxEnode := mailboxNode.Server().NodeInfo().Enode

// 	sender, stop := s.startBackend()
// 	defer stop()
// 	node, err := sender.NodeManager().Node()
// 	s.Require().NoError(err)

// 	s.Require().NotEqual(mailboxEnode, node.Server().NodeInfo().Enode)

// 	err = sender.NodeManager().AddPeer(mailboxEnode)
// 	s.Require().NoError(err)
// 	//wait async processes on adding peer
// 	time.Sleep(time.Second)

// 	w, err := sender.NodeManager().WhisperService()
// 	s.Require().NoError(err)

// 	//Mark mailbox node trusted
// 	parsedNode, err := discover.ParseNode(mailboxNode.Server().NodeInfo().Enode)
// 	s.Require().NoError(err)
// 	mailboxPeer := parsedNode.ID[:]
// 	mailboxPeerStr := parsedNode.ID.String()
// 	err = w.AllowP2PMessagesFromPeer(mailboxPeer)
// 	s.Require().NoError(err)

// 	//Generate mailbox symkey
// 	password := "status-offline-inbox"
// 	MailServerKeyID, err := w.AddSymKeyFromPassword(password)
// 	s.Require().NoError(err)

// 	rpcClient := sender.NodeManager().RPCClient()
// 	s.Require().NotNil(rpcClient)

// 	//create topic
// 	topic := whisperv5.BytesToTopic([]byte("topic name"))

// 	//Add key pair to whisper
// 	keyID, err := w.NewKeyPair()
// 	s.Require().NoError(err)
// 	key, err := w.GetPrivateKey(keyID)
// 	s.Require().NoError(err)
// 	pubkey := hexutil.Bytes(crypto.FromECDSAPub(&key.PublicKey))

// 	//Create message filter
// 	resp := rpcClient.CallRaw(`{
// 		"jsonrpc": "2.0",
// 		"method": "shh_newMessageFilter", "params": [
// 			{"privateKeyID": "` + keyID + `", "topics": [ "` + topic.String() + `"], "allowP2P":true}
// 		],
// 		"id": 1
// 	}`)
// 	msgFilterResp := newMessagesFilterResponse{}
// 	err = json.Unmarshal([]byte(resp), &msgFilterResp)
// 	messageFilterID := msgFilterResp.Result
// 	s.Require().NoError(err)
// 	s.Require().NotEqual("", messageFilterID)

// 	//Threre are no messages at filter
// 	resp = rpcClient.CallRaw(`{
// 		"jsonrpc": "2.0",
// 		"method": "shh_getFilterMessages",
// 		"params": ["` + messageFilterID + `"],
// 		"id": 1}`)
// 	messages := getFilterMessagesResponse{}
// 	err = json.Unmarshal([]byte(resp), &messages)
// 	s.Require().NoError(err)
// 	s.Require().Equal(0, len(messages.Result))

// 	//Post message
// 	resp = rpcClient.CallRaw(`{
// 		"jsonrpc": "2.0",
// 		"method": "shh_post",
// 		"params": [
// 			{
// 			"pubKey": "` + pubkey.String() + `",
// 			"topic": "` + topic.String() + `",
// 			"payload": "0x73656e74206265666f72652066696c7465722077617320616374697665202873796d6d657472696329",
// 			"powTarget": 0.001,
// 			"powTime": 2
// 			}
// 		],
// 		"id": 1}`)
// 	postResp := baseRPCResponse{}
// 	err = json.Unmarshal([]byte(resp), &postResp)
// 	s.Require().NoError(err)
// 	s.Require().Nil(postResp.Err)

// 	//There are no messages, because it's a sender filter
// 	resp = rpcClient.CallRaw(`{
// 		"jsonrpc": "2.0",
// 		"method": "shh_getFilterMessages",
// 		"params": ["` + messageFilterID + `"],
// 		"id": 1}`)
// 	err = json.Unmarshal([]byte(resp), &messages)
// 	s.Require().NoError(err)
// 	s.Require().Equal(0, len(messages.Result))

// 	//act

// 	//Request messages from mailbox
// 	reqMessagesBody := `{
// 		"jsonrpc": "2.0",
// 		"id": 1,
// 		"method": "shh_requestMessages",
// 		"params": [{
// 					"peer":"` + mailboxPeerStr + `",
// 					"topic":"` + topic.String() + `",
// 					"symKeyID":"` + MailServerKeyID + `",
// 					"from":0,
// 					"to":` + strconv.FormatInt(time.Now().UnixNano()/int64(time.Second), 10) + `
// 		}]
// 	}`
// 	resp = rpcClient.CallRaw(reqMessagesBody)
// 	reqMessagesResp := baseRPCResponse{}
// 	err = json.Unmarshal([]byte(resp), &reqMessagesResp)
// 	s.Require().NoError(err)
// 	s.Require().Nil(postResp.Err)

// 	//wait to receive message
// 	time.Sleep(time.Second)
// 	//And we receive message
// 	resp = rpcClient.CallRaw(`{
// 		"jsonrpc": "2.0",
// 		"method": "shh_getFilterMessages",
// 		"params": ["` + messageFilterID + `"],
// 		"id": 1}`)

// 	err = json.Unmarshal([]byte(resp), &messages)
// 	//assert
// 	s.Require().NoError(err)
// 	s.Require().Equal(1, len(messages.Result))

// 	//check that there are no messages
// 	resp = rpcClient.CallRaw(`{
// 		"jsonrpc": "2.0",
// 		"method": "shh_getFilterMessages",
// 		"params": ["` + messageFilterID + `"],
// 		"id": 1}`)

// 	err = json.Unmarshal([]byte(resp), &messages)
// 	//assert
// 	s.Require().NoError(err)
// 	s.Require().Equal(0, len(messages.Result))

// 	//Request each one messages from mailbox, using same params
// 	resp = rpcClient.CallRaw(reqMessagesBody)
// 	reqMessagesResp = baseRPCResponse{}
// 	err = json.Unmarshal([]byte(resp), &reqMessagesResp)
// 	s.Require().NoError(err)
// 	s.Require().Nil(postResp.Err)

// 	//wait to receive message
// 	time.Sleep(time.Second)
// 	//And we receive message
// 	resp = rpcClient.CallRaw(`{
// 		"jsonrpc": "2.0",
// 		"method": "shh_getFilterMessages",
// 		"params": ["` + messageFilterID + `"],
// 		"id": 1}`)

// 	err = json.Unmarshal([]byte(resp), &messages)
// 	//assert
// 	s.Require().NoError(err)
// 	s.Require().Equal(1, len(messages.Result))

// 	time.Sleep(time.Second)

// 	//Request each one messages from mailbox using enode
// 	resp = rpcClient.CallRaw(`{
// 		"jsonrpc": "2.0",
// 		"id": 2,
// 		"method": "shh_requestMessages",
// 		"params": [{
// 					"enode":"` + mailboxEnode + `",
// 					"topic":"` + topic.String() + `",
// 					"symKeyID":"` + MailServerKeyID + `",
// 					"from":0,
// 					"to":` + strconv.FormatInt(time.Now().UnixNano()/int64(time.Second), 10) + `
// 		}]
// 	}`)
// 	reqMessagesResp = baseRPCResponse{}
// 	err = json.Unmarshal([]byte(resp), &reqMessagesResp)
// 	s.Require().NoError(err)
// 	s.Require().Nil(postResp.Err)

// 	//wait to receive message
// 	time.Sleep(time.Second)
// 	//And we receive message
// 	resp = rpcClient.CallRaw(`{
// 		"jsonrpc": "2.0",
// 		"method": "shh_getFilterMessages",
// 		"params": ["` + messageFilterID + `"],
// 		"id": 1}`)

// 	err = json.Unmarshal([]byte(resp), &messages)
// 	//assert
// 	s.Require().NoError(err)
// 	s.Require().Equal(1, len(messages.Result))

// }

// func (s *WhisperMailboxSuite) startBackend() (*api.StatusBackend, func()) {
// 	//Start sender node
// 	backend := api.NewStatusBackend()
// 	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
// 	s.Require().NoError(err)
// 	s.Require().False(backend.IsNodeRunning())
// 	nodeStarted, err := backend.StartNode(nodeConfig)
// 	s.Require().NoError(err)
// 	<-nodeStarted // wait till node is started
// 	s.Require().True(backend.IsNodeRunning())

// 	return backend, func() {
// 		s.True(backend.IsNodeRunning())
// 		backendStopped, err := backend.StopNode()
// 		s.NoError(err)
// 		<-backendStopped
// 		s.False(backend.IsNodeRunning())
// 	}

// }
// func (s *WhisperMailboxSuite) startMailboxBackend() (*api.StatusBackend, func()) {
// 	//Start mailbox node
// 	mailboxBackend := api.NewStatusBackend()
// 	mailboxConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
// 	s.Require().NoError(err)

// 	mailboxConfig.LightEthConfig.Enabled = false
// 	mailboxConfig.WhisperConfig.Enabled = true
// 	mailboxConfig.KeyStoreDir = "../../.ethereumtest/mailbox/"
// 	mailboxConfig.WhisperConfig.EnableMailServer = true
// 	mailboxConfig.WhisperConfig.IdentityFile = "../../static/keys/wnodekey"
// 	mailboxConfig.WhisperConfig.PasswordFile = "../../static/keys/wnodepassword"
// 	mailboxConfig.WhisperConfig.DataDir = "../../.ethereumtest/mailbox/w2"
// 	mailboxConfig.DataDir = "../../.ethereumtest/mailbox/"

// 	mailboxNodeStarted, err := mailboxBackend.StartNode(mailboxConfig)
// 	s.Require().NoError(err)
// 	<-mailboxNodeStarted // wait till node is started
// 	s.Require().True(mailboxBackend.IsNodeRunning())
// 	return mailboxBackend, func() {
// 		s.True(mailboxBackend.IsNodeRunning())
// 		backendStopped, err := mailboxBackend.StopNode()
// 		s.NoError(err)
// 		<-backendStopped
// 		s.False(mailboxBackend.IsNodeRunning())
// 	}
// }

type getFilterMessagesResponse struct {
	Result []map[string]interface{}
	Err    interface{}
}

type newMessagesFilterResponse struct {
	Result string
	Err    interface{}
}
type baseRPCResponse struct {
	Result interface{}
	Err    interface{}
}
