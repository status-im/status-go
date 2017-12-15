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
	"os"
)

type WhisperMailboxSuite struct {
	suite.Suite
}

func TestWhisperMailboxTestSuite(t *testing.T) {
	suite.Run(t, new(WhisperMailboxSuite))
}

func (s *WhisperMailboxSuite) TestRequestMessageFromMailboxAsync() {
	//Start mailbox and status node
	mailboxBackend, stop := s.startMailboxBackend()
	defer stop()
	mailboxNode, err := mailboxBackend.NodeManager().Node()
	s.Require().NoError(err)
	mailboxEnode := mailboxNode.Server().NodeInfo().Enode

	sender, stop := s.startBackend("sender")
	defer stop()
	node, err := sender.NodeManager().Node()
	s.Require().NoError(err)

	s.Require().NotEqual(mailboxEnode, node.Server().NodeInfo().Enode)

	err = sender.NodeManager().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	//wait async processes on adding peer
	time.Sleep(time.Second)

	senderWhisperService, err := sender.NodeManager().WhisperService()
	s.Require().NoError(err)

	//Mark mailbox node trusted
	parsedNode, err := discover.ParseNode(mailboxNode.Server().NodeInfo().Enode)
	s.Require().NoError(err)
	mailboxPeer := parsedNode.ID[:]
	mailboxPeerStr := parsedNode.ID.String()
	err = senderWhisperService.AllowP2PMessagesFromPeer(mailboxPeer)
	s.Require().NoError(err)

	//Generate mailbox symkey
	password := "status-offline-inbox"
	MailServerKeyID, err := senderWhisperService.AddSymKeyFromPassword(password)
	s.Require().NoError(err)

	rpcClient := sender.NodeManager().RPCClient()
	s.Require().NotNil(rpcClient)

	//create topic
	topic := whisperv5.BytesToTopic([]byte("topic name"))

	//Add key pair to whisper
	keyID, err := senderWhisperService.NewKeyPair()
	s.Require().NoError(err)
	key, err := senderWhisperService.GetPrivateKey(keyID)
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
	msgFilterResp := returnedIDResponse{}
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

func (s *WhisperMailboxSuite) TestRequestMessagesFromMailboxFromGroupChat() {
	//Start mailbox, alice, bob, charlie node
	mailboxBackend, stop := s.startMailboxBackend()
	defer stop()
	mailboxNode, err := mailboxBackend.NodeManager().Node()
	s.Require().NoError(err)
	mailboxEnode := mailboxNode.Server().NodeInfo().Enode

	aliceBackend, stop := s.startBackend("alice")
	defer stop()
	aliceNode, err := aliceBackend.NodeManager().Node()
	s.Require().NoError(err)
	aliceEnode := aliceNode.Server().NodeInfo().Enode

	bobBackend, stop := s.startBackend("bob")
	defer stop()

	charlieBackend, stop := s.startBackend("charlie")
	defer stop()

	s.Require().NotEqual(mailboxEnode, aliceEnode)

	err = aliceBackend.NodeManager().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	err = bobBackend.NodeManager().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	err = charlieBackend.NodeManager().AddPeer(mailboxEnode)
	s.Require().NoError(err)
	//wait async processes on adding peer
	time.Sleep(time.Second)

	aliceWhisperService, err := aliceBackend.NodeManager().WhisperService()
	s.Require().NoError(err)
	bobWhisperService, err := bobBackend.NodeManager().WhisperService()
	s.Require().NoError(err)
	charlieWhisperService, err := charlieBackend.NodeManager().WhisperService()
	s.Require().NoError(err)

	//add mailserver key
	password := "status-offline-inbox"
	bobMailServerKeyID, err := bobWhisperService.AddSymKeyFromPassword(password)
	s.Require().NoError(err)
	charlieMailServerKeyID, err := charlieWhisperService.AddSymKeyFromPassword(password)
	s.Require().NoError(err)

	//get rpc client
	aliceRpcClient := aliceBackend.NodeManager().RPCClient()
	s.Require().NotNil(aliceRpcClient)
	bobRpcClient := bobBackend.NodeManager().RPCClient()
	s.Require().NotNil(bobRpcClient)
	charlieRpcClient := charlieBackend.NodeManager().RPCClient()
	s.Require().NotNil(charlieRpcClient)

	//generate group chat symkey and topic
	groupChatKeyID, err := aliceWhisperService.GenerateSymKey()
	s.Require().NoError(err)
	groupChatKey, err := aliceWhisperService.GetSymKey(groupChatKeyID)
	s.Require().NoError(err)
	//generate group chat topic
	groupChatTopic := whisperv5.BytesToTopic([]byte("groupChatTopic"))
	groupChatPayload := newGroupChatParams(groupChatKey, groupChatTopic)
	payloadStr, err := groupChatPayload.Encode()
	s.Require().NoError(err)

	//Add bob and charlie create key pairs to receive symmetric key for group chat from alice
	bobKeyID, err := bobWhisperService.NewKeyPair()
	s.Require().NoError(err)
	bobKey, err := bobWhisperService.GetPrivateKey(bobKeyID)
	s.Require().NoError(err)
	bobPubkey := hexutil.Bytes(crypto.FromECDSAPub(&bobKey.PublicKey))
	bobAliceKeySendTopic := whisperv5.BytesToTopic([]byte("bobAliceKeySendTopic "))

	charlieKeyID, err := charlieWhisperService.NewKeyPair()
	s.Require().NoError(err)
	charlieKey, err := charlieWhisperService.GetPrivateKey(charlieKeyID)
	s.Require().NoError(err)
	charliePubkey := hexutil.Bytes(crypto.FromECDSAPub(&charlieKey.PublicKey))
	charlieAliceKeySendTopic := whisperv5.BytesToTopic([]byte("charlieAliceKeySendTopic "))

	//bob and charlie create message filter
	resp := bobRpcClient.CallRaw(`{
			"jsonrpc": "2.0",
			"method": "shh_newMessageFilter", "params": [
				{"privateKeyID": "` + bobKeyID + `", "topics": [ "` + bobAliceKeySendTopic.String() + `"], "allowP2P":true}
			],
			"id": 1
		}`)

	msgFilterResp := returnedIDResponse{}
	err = json.Unmarshal([]byte(resp), &msgFilterResp)
	bobMessageFilterID := msgFilterResp.Result
	s.Require().NoError(err)
	s.Require().NotEqual("", bobMessageFilterID, resp)

	resp = charlieRpcClient.CallRaw(`{
			"jsonrpc": "2.0",
			"method": "shh_newMessageFilter", "params": [
				{"privateKeyID": "` + charlieKeyID + `", "topics": [ "` + charlieAliceKeySendTopic.String() + `"], "allowP2P":true}
			],
			"id": 1
		}`)

	msgFilterResp = returnedIDResponse{}
	err = json.Unmarshal([]byte(resp), &msgFilterResp)
	charlieMessageFilterID := msgFilterResp.Result
	s.Require().NoError(err)
	s.Require().NotEqual("", charlieMessageFilterID, resp)

	//Alice send message with symkey and topic to bob and charlie
	resp = aliceRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"pubKey": "` + bobPubkey.String() + `",
			"topic": "` + bobAliceKeySendTopic.String() + `",
			"payload": "` + payloadStr + `",
			"powTarget": 0.001,
			"powTime": 2
			}
		],
		"id": 1}`)
	postResp := baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &postResp)
	s.Require().NoError(err)
	s.Require().Nil(postResp.Err)

	resp = aliceRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"pubKey": "` + charliePubkey.String() + `",
			"topic": "` + charlieAliceKeySendTopic.String() + `",
			"payload": "` + payloadStr + `",
			"powTarget": 0.001,
			"powTime": 2
			}
		],
		"id": 1}`)
	postResp = baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &postResp)
	s.Require().NoError(err)
	s.Require().Nil(postResp.Err)
	//wait to receive
	time.Sleep(time.Second)

	//bob receive group chat data and add it to his node
	//1. bob get group chat details
	resp = bobRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + bobMessageFilterID + `"],
		"id": 1}`)
	messages := getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))
	s.Require().NoError(err)

	bobGroupChatData := groupChatParams{}
	bobGroupChatData.Decode(messages.Result[0]["payload"].(string))
	s.EqualValues(groupChatPayload, bobGroupChatData)

	//2. bob add symkey to his node
	resp = bobRpcClient.CallRaw(`{"jsonrpc":"2.0","method":"shh_addSymKey",
			"params":["` + bobGroupChatData.Key + `"],
			"id":1}`)
	symkeyAddResp := returnedIDResponse{}
	err = json.Unmarshal([]byte(resp), &symkeyAddResp)
	s.Require().NoError(err)
	bobGroupChatSymkeyID := symkeyAddResp.Result
	s.Require().NotEmpty(bobGroupChatSymkeyID)

	//3. bob create message filter to node by group chat topic
	resp = bobRpcClient.CallRaw(`{
			"jsonrpc": "2.0",
			"method": "shh_newMessageFilter", "params": [
				{"symKeyID": "` + bobGroupChatSymkeyID + `", "topics": [ "` + bobGroupChatData.Topic + `"], "allowP2P":true}
			],
			"id": 1
		}`)

	msgFilterResp = returnedIDResponse{}
	err = json.Unmarshal([]byte(resp), &msgFilterResp)
	s.Require().NoError(err)
	bobGroupChatMessageFilterID := msgFilterResp.Result

	//charlie receive group chat data and add it to his node
	//1. charlie get group chat details
	resp = charlieRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + charlieMessageFilterID + `"],
		"id": 1}`)
	messages = getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))
	charlieGroupChatData := groupChatParams{}
	charlieGroupChatData.Decode(messages.Result[0]["payload"].(string))
	s.EqualValues(groupChatPayload, charlieGroupChatData)

	//2. charlie add symkey to his node
	resp = charlieRpcClient.CallRaw(`{"jsonrpc":"2.0","method":"shh_addSymKey",
			"params":["` + bobGroupChatData.Key + `"],
			"id":1}`)
	symkeyAddResp = returnedIDResponse{}
	err = json.Unmarshal([]byte(resp), &symkeyAddResp)
	s.Require().NoError(err)
	charlieGroupChatSymkeyID := symkeyAddResp.Result
	s.Require().NotEmpty(charlieGroupChatSymkeyID)

	//3. charlie create message filter to node by group chat topic
	resp = charlieRpcClient.CallRaw(`{
			"jsonrpc": "2.0",
			"method": "shh_newMessageFilter", "params": [
				{"symKeyID": "` + charlieGroupChatSymkeyID + `", "topics": [ "` + charlieGroupChatData.Topic + `"], "allowP2P":true}
			],
			"id": 1
		}`)

	msgFilterResp = returnedIDResponse{}
	err = json.Unmarshal([]byte(resp), &msgFilterResp)
	s.Require().NoError(err)
	charlieGroupChatMessageFilterID := msgFilterResp.Result

	helloWorldMessage := hexutil.Encode([]byte("Hello world!"))
	//Post message
	resp = aliceRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"symKeyID": "` + groupChatKeyID + `",
			"topic": "` + groupChatTopic.String() + `",
			"payload": "` + helloWorldMessage + `",
			"powTarget": 0.001,
			"powTime": 2
			}
		],
		"id": 1}`)
	postResp = baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &postResp)
	s.Require().NoError(err)
	s.Require().Nil(postResp.Err)
	time.Sleep(time.Second)

	resp = bobRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + bobGroupChatMessageFilterID + `"],
		"id": 1}`)
	messages = getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))
	s.Require().Equal(helloWorldMessage, messages.Result[0]["payload"].(string))
	s.Require().NoError(err)

	resp = charlieRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + charlieGroupChatMessageFilterID + `"],
		"id": 1}`)
	messages = getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))
	s.Require().Equal(helloWorldMessage, messages.Result[0]["payload"].(string))
	s.Require().NoError(err)

	resp = bobRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + bobGroupChatMessageFilterID + `"],
		"id": 1}`)
	messages = getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(0, len(messages.Result))
	s.Require().NoError(err)

	resp = charlieRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + charlieGroupChatMessageFilterID + `"],
		"id": 1}`)
	messages = getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(0, len(messages.Result))

	//Request each one messages from mailbox using enode
	resp = bobRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "shh_requestMessages",
		"params": [{
					"enode":"` + mailboxEnode + `",
					"topic":"` + groupChatTopic.String() + `",
					"symKeyID":"` + bobMailServerKeyID + `",
					"from":0,
					"to":` + strconv.FormatInt(time.Now().UnixNano()/int64(time.Second), 10) + `
		}]
	}`)
	reqMessagesResp := baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &reqMessagesResp)
	s.Require().NoError(err)

	resp = charlieRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "shh_requestMessages",
		"params": [{
					"enode":"` + mailboxEnode + `",
					"topic":"` + groupChatTopic.String() + `",
					"symKeyID":"` + charlieMailServerKeyID + `",
					"from":0,
					"to":` + strconv.FormatInt(time.Now().UnixNano()/int64(time.Second), 10) + `
		}]
	}`)
	reqMessagesResp = baseRPCResponse{}
	err = json.Unmarshal([]byte(resp), &reqMessagesResp)
	s.Require().NoError(err)

	//wait to receive p2p messages
	time.Sleep(time.Second)

	resp = bobRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + bobGroupChatMessageFilterID + `"],
		"id": 1}`)
	messages = getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))
	s.Require().Equal(helloWorldMessage, messages.Result[0]["payload"].(string))
	s.Require().NoError(err)

	resp = charlieRpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + charlieGroupChatMessageFilterID + `"],
		"id": 1}`)
	messages = getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	s.Require().NoError(err)
	s.Require().Equal(1, len(messages.Result))
	s.Require().Equal(helloWorldMessage, messages.Result[0]["payload"].(string))
	s.Require().NoError(err)
}

func newGroupChatParams(symkey []byte, topic whisperv5.TopicType) groupChatParams {
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

//Start status node
func (s *WhisperMailboxSuite) startBackend(name string) (*api.StatusBackend, func()) {
	datadir := "../../.ethereumtest/mailbox/" + name
	backend := api.NewStatusBackend()
	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	nodeConfig.DataDir = datadir
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
		os.RemoveAll(datadir)
	}

}

//Start mailbox node
func (s *WhisperMailboxSuite) startMailboxBackend() (*api.StatusBackend, func()) {
	mailboxBackend := api.NewStatusBackend()
	mailboxConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	s.Require().NoError(err)
	datadir := "../../.ethereumtest/mailbox/mailbox/"

	mailboxConfig.LightEthConfig.Enabled = false
	mailboxConfig.WhisperConfig.Enabled = true
	mailboxConfig.KeyStoreDir = "../../.ethereumtest/mailbox/mailbox"
	mailboxConfig.WhisperConfig.EnableMailServer = true
	mailboxConfig.WhisperConfig.IdentityFile = "../../static/keys/wnodekey"
	mailboxConfig.WhisperConfig.PasswordFile = "../../static/keys/wnodepassword"
	mailboxConfig.WhisperConfig.DataDir = "../../.ethereumtest/mailbox/mailbox/data"
	mailboxConfig.DataDir = datadir

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
		os.RemoveAll(datadir)
	}
}

type getFilterMessagesResponse struct {
	Result []map[string]interface{}
	Err    interface{}
}

type returnedIDResponse struct {
	Result string
	Err    interface{}
}
type baseRPCResponse struct {
	Result interface{}
	Err    interface{}
}
