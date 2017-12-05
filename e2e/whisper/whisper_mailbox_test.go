package whisper

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/api"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"
)

func TestRequestMessageFromMailboxAsync(t *testing.T) {
	//Start mailbox node
	mailboxBackend := api.NewStatusBackend()
	mailboxConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	mailboxConfig.LightEthConfig.Enabled = false
	mailboxConfig.WhisperConfig.Enabled = true
	mailboxConfig.KeyStoreDir = "../../.ethereumtest/mailbox/"
	mailboxConfig.WhisperConfig.BootstrapNode = true
	mailboxConfig.WhisperConfig.ForwarderNode = true
	mailboxConfig.WhisperConfig.MailServerNode = true
	mailboxConfig.WhisperConfig.IdentityFile = "../../static/keys/wnodekey"
	mailboxConfig.WhisperConfig.PasswordFile = "../../static/keys/wnodepassword"
	mailboxConfig.WhisperConfig.DataDir = "../../.ethereumtest/mailbox/w2"
	mailboxConfig.DataDir = "../../.ethereumtest/mailbox/"

	mailboxNodeStarted, err := mailboxBackend.StartNode(mailboxConfig)
	require.NoError(t, err)
	<-mailboxNodeStarted // wait till node is started
	require.True(t, mailboxBackend.IsNodeRunning())

	mailboxNode, err := mailboxBackend.NodeManager().Node()
	mailboxEnode := mailboxNode.Server().NodeInfo().Enode

	//Start sender node
	backend := api.NewStatusBackend()
	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	require.NoError(t, err)
	require.False(t, backend.IsNodeRunning())
	nodeStarted, err := backend.StartNode(nodeConfig)
	require.NoError(t, err)
	<-nodeStarted // wait till node is started
	require.True(t, backend.IsNodeRunning())
	node, err := backend.NodeManager().Node()
	require.NoError(t, err)

	require.NotEqual(t, mailboxEnode, node.Server().NodeInfo().Enode)

	err = backend.NodeManager().AddPeer(mailboxEnode)
	require.NoError(t, err)
	//wait async processes on adding peer
	time.Sleep(time.Second)

	w, err := backend.NodeManager().WhisperService()
	require.NoError(t, err)

	//Mark mailbox node trusted
	p, err := extractIdFromEnode(mailboxNode.Server().NodeInfo().Enode)
	require.NoError(t, err)
	err = w.AllowP2PMessagesFromPeer(p)
	require.NoError(t, err)

	//Generate mailbox symkey
	password := "asdfasdf"
	MailServerKeyID, err := w.AddSymKeyFromPassword(password)
	require.NoError(t, err)

	rpcClient := backend.NodeManager().RPCClient()
	require.NotNil(t, rpcClient)

	//create topic
	topic := whisperv5.BytesToTopic([]byte("topic name"))

	//Add key pair to whisper
	keyID, err := w.NewKeyPair()
	key, err := w.GetPrivateKey(keyID)
	require.NoError(t, err)
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
	require.NoError(t, err)
	require.NotEqual(t, "", messageFilterID)

	//Threre are no messages at filter
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	messages := getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(resp), &messages)
	require.NoError(t, err)
	require.Equal(t, 0, len(messages.Result))

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
	fmt.Println("d", resp)

	//Threre are no messages, because it's sender filter
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	err = json.Unmarshal([]byte(resp), &messages)
	require.NoError(t, err)
	require.Equal(t, 0, len(messages.Result))

	//Request messages from mailbox
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "shh_requestMessages",
		"params": [{
					"enode":"` + mailboxNode.Server().NodeInfo().Enode + `",
					"topic":"` + topic.String() + `",
					"symKeyID":"` + MailServerKeyID + `",
					"from":0,
					"to":` + strconv.FormatInt(time.Now().UnixNano(), 10) + `
		}]
	}`)

	//And we receive message
	resp = rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)

	err = json.Unmarshal([]byte(resp), &messages)
	require.NoError(t, err)
	require.Equal(t, 1, len(messages.Result))

}

type getFilterMessagesResponse struct {
	Result []map[string]interface{}
	Err    interface{}
}

type newMessagesFilterResponse struct {
	Result string
	Err    interface{}
}

func extractIdFromEnode(s string) ([]byte, error) {
	n, err := discover.ParseNode(s)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse enode: %s", err)
	}
	return n.ID[:], nil
}
