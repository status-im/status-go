package whisper

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/api"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/suite"
)

type WhisperMailboxSuite struct {
	suite.Suite
}

func TestWhisperMailboxTestSuite(t *testing.T) {
	suite.Run(t, new(WhisperMailboxSuite))
}

func (s *WhisperMailboxSuite) TestRequestMessageFromMailboxAsync() {
	//arrange
	mailboxBackend, stop := s.startMailboxBackend()
	defer stop()
	mailboxNode, err := mailboxBackend.NodeManager().Node()
	s.Require().NoError(err)
	mailboxEnode := mailboxNode.Server().NodeInfo().Enode

	sender, stop := s.startBackend()
	defer stop()
	node, err := sender.NodeManager().Node()
	s.Require().NoError(err)

	s.Require().NotEqual(mailboxEnode, node.Server().NodeInfo().Enode)

	err = sender.NodeManager().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	//wait async processes on adding peer
	time.Sleep(time.Second)

	w, err := sender.NodeManager().WhisperService()
	s.Require().NoError(err)

	//Mark mailbox node trusted
	parsedNode, err := discover.ParseNode(mailboxNode.Server().NodeInfo().Enode)
	s.Require().NoError(err)
	mailboxPeer := parsedNode.ID[:]
	mailboxPeerStr := parsedNode.ID.String()
	err = w.AllowP2PMessagesFromPeer(mailboxPeer)
	s.Require().NoError(err)

	//Generate mailbox symkey
	password := "status-offline-inbox"
	MailServerKeyID, err := w.AddSymKeyFromPassword(password)
	s.Require().NoError(err)

	rpcClient := sender.NodeManager().RPCClient()
	s.Require().NotNil(rpcClient)

	//create topic
	topic := whisperv5.BytesToTopic([]byte("topic name"))

	//Add key pair to whisper
	keyID, err := w.NewKeyPair()
	s.Require().NoError(err)
	key, err := w.GetPrivateKey(keyID)
	s.Require().NoError(err)
	pubkey := hexutil.Bytes(crypto.FromECDSAPub(&key.PublicKey))

	//Create message filter
	resp := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_newMessageFilter", "params": [
			{"privateKeyID": "` + keyID + `", "topics": [ "` + topic.String() + `"], "allowP2P":true}
		],
		"id": 1
	}`)
	msgFilterResp := newMessagesFilterResponse{}
	err = json.Unmarshal([]byte(resp), &msgFilterResp)
	messageFilterID := msgFilterResp.Result
	s.Require().NoError(err)
	s.Require().NotEqual("", messageFilterID)

	//Threre are no messages at filter
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	messages := getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(0, len(messages.Result))

	//Post message
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"pubKey": "` + pubkey.String() + `",
			"topic": "` + topic.String() + `",
			"payload": "0x73656e74206265666f72652066696c7465722077617320616374697665202873796d6d657472696329",
			"powTarget": 0.001,
			"powTime": 2
			}
		],
		"id": 1}`)
	postResp := baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &postResp)
	s.Require().NoError(err)
	s.Require().Nil(postResp.Err)

	// Propagate the sent message.
	time.Sleep(time.Second)

	// Receive the sent message.
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))

	// Make sure there are no new messages.
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(0, len(messages.Result))

	//act

	//Request messages from mailbox
	reqMessagesBody := `{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "shh_requestMessages",
		"params": [{
					"mailServerPeer":"` + mailboxPeerStr + `",
					"topic":"` + topic.String() + `",
					"symKeyID":"` + MailServerKeyID + `",
					"from":0,
					"to":` + strconv.FormatInt(time.Now().UTC().Unix(), 10) + `
		}]
	}`
	resp = rpcClient.CallRaw(reqMessagesBody)
	reqMessagesResp := baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &reqMessagesResp)
	s.Require().NoError(err)
	s.Require().Nil(postResp.Err)

	//wait to receive message
	time.Sleep(time.Second)
	//And we receive message
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)

	err = json.Unmarshal([]byte(resp), &messages)
	//assert
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))

	//check that there are no messages
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)

	err = json.Unmarshal([]byte(resp), &messages)
	//assert
	s.Require().NoError(err)
	s.Require().Equal(0, len(messages.Result))

	//Request each one messages from mailbox, using same params
	resp = rpcClient.CallRaw(reqMessagesBody)
	reqMessagesResp = baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &reqMessagesResp)
	s.Require().NoError(err)
	s.Require().Nil(postResp.Err)

	//wait to receive message
	time.Sleep(time.Second)
	//And we receive message
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)

	err = json.Unmarshal([]byte(resp), &messages)
	//assert
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))

	time.Sleep(time.Second)

	//Request each one messages from mailbox using enode
	resp = rpcClient.CallRaw(reqMessagesBody)
	reqMessagesResp = baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &reqMessagesResp)
	s.Require().NoError(err)
	s.Require().Nil(postResp.Err)

	//wait to receive message
	time.Sleep(time.Second)
	//And we receive message
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)

	err = json.Unmarshal([]byte(resp), &messages)
	//assert
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))

}

func (s *WhisperMailboxSuite) startBackend() (*api.StatusBackend, func()) {
	//Start sender node
	backend := api.NewStatusBackend()
	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.Require().NoError(err)
	s.Require().False(backend.IsNodeRunning())
	nodeStarted, err := backend.StartNode(nodeConfig)
	s.Require().NoError(err)
	<-nodeStarted // wait till node is started
	s.Require().True(backend.IsNodeRunning())

	return backend, func() {
		s.True(backend.IsNodeRunning())
		backendStopped, err := backend.StopNode()
		s.NoError(err)
		<-backendStopped
		s.False(backend.IsNodeRunning())
	}

}
func (s *WhisperMailboxSuite) startMailboxBackend() (*api.StatusBackend, func()) {
	//Start mailbox node
	mailboxBackend := api.NewStatusBackend()
	mailboxConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.Require().NoError(err)

	mailboxConfig.LightEthConfig.Enabled = false
	mailboxConfig.WhisperConfig.Enabled = true
	mailboxConfig.KeyStoreDir = "../../.ethereumtest/mailbox/"
	mailboxConfig.WhisperConfig.EnableMailServer = true
	mailboxConfig.WhisperConfig.IdentityFile = "../../static/keys/wnodekey"
	mailboxConfig.WhisperConfig.PasswordFile = "../../static/keys/wnodepassword"
	mailboxConfig.WhisperConfig.DataDir = "../../.ethereumtest/mailbox/w2"
	mailboxConfig.DataDir = "../../.ethereumtest/mailbox/"

	mailboxNodeStarted, err := mailboxBackend.StartNode(mailboxConfig)
	s.Require().NoError(err)
	<-mailboxNodeStarted // wait till node is started
	s.Require().True(mailboxBackend.IsNodeRunning())
	return mailboxBackend, func() {
		s.True(mailboxBackend.IsNodeRunning())
		backendStopped, err := mailboxBackend.StopNode()
		s.NoError(err)
		<-backendStopped
		s.False(mailboxBackend.IsNodeRunning())
	}
}

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
