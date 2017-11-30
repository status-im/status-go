package whisper

import (
	"encoding/json"
	"fmt"
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

func TestOne(t *testing.T) {
	backend := api.NewStatusBackend()
	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	require.NoError(t, err)
	require.False(t, backend.IsNodeRunning())

	nodeStarted, err := backend.StartNode(nodeConfig)
	require.NoError(t, err)

	<-nodeStarted // wait till node is started
	time.Sleep(time.Second)
	require.True(t, backend.IsNodeRunning())

	mailboxBackend := api.NewStatusBackend()
	mailboxConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	mailboxConfig.WhisperConfig.Enabled = true
	//mailboxConfig.WhisperConfig.BootstrapNode=true
	//mailboxConfig.WhisperConfig.ForwarderNode=true
	mailboxConfig.WhisperConfig.MailServerNode = true
	mailboxConfig.WhisperConfig.IdentityFile = "../../static/keys/wnodekey"
	mailboxConfig.WhisperConfig.PasswordFile = "../../static/keys/wnodepassword"
	mailboxConfig.WhisperConfig.DataDir = "../../.ethereumtest/mailbox/w2"
	mailboxConfig.DataDir = "../../.ethereumtest/mailbox/"

	mailboxNodeStarted, err := mailboxBackend.StartNode(mailboxConfig)
	require.NoError(t, err)
	<-mailboxNodeStarted // wait till node is started
	time.Sleep(time.Second)
	require.True(t, mailboxBackend.IsNodeRunning())

	mailboxNode, err := mailboxBackend.NodeManager().Node()
	node, err := mailboxBackend.NodeManager().Node()
	require.NoError(t, err)

	mailboxEnode := mailboxNode.Server().NodeInfo().Enode
	t.Log("enode - ", node.Server().NodeInfo().Enode)
	t.Log("mailbox enode - ", mailboxEnode)
	err = backend.NodeManager().AddPeer(mailboxEnode)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	w, _ := backend.NodeManager().WhisperService()

	p, err := extractIdFromEnode(mailboxNode.Server().NodeInfo().Enode)
	require.NoError(t, err)
	err = w.AllowP2PMessagesFromPeer(p)
	require.NoError(t, err)

	password := "asdsa"
	MailServerKeyID, err := w.AddSymKeyFromPassword(password)
	require.NoError(t, err)

	rpcClient := backend.NodeManager().RPCClient()
	require.NotNil(t, rpcClient)

	topic := whisperv5.BytesToTopic([]byte("topic name"))
	symkeyID, err := w.GenerateSymKey()

	b := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_newMessageFilter", "params": [
			{"symKeyID": "` + symkeyID + `", "topics": [ "` + topic.String() + `"]}
		],
		"id": 1
	}`)
	msgFilterResp := newMessagesFilterResponse{}
	err = json.Unmarshal([]byte(b), &msgFilterResp)
	messageFilterID := msgFilterResp.Result
	require.NoError(t, err)
	require.NotEqual(t, "", messageFilterID)

	c := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	messages := getFilterMessagesResponse{}
	err = json.Unmarshal([]byte(c), &messages)
	require.NoError(t, err)
	require.Equal(t, 0, len(messages.Result))

	d := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_post",
		"params": [
			{
			"symKeyID": "` + symkeyID + `",
			"topic": "` + topic.String() + `",
			"payload": "0x73656e74206265666f72652066696c7465722077617320616374697665202873796d6d657472696329",
			"powTarget": 0.001,
			"powTime": 2
			}
		],
		"id": 1}`)
	fmt.Println("d", d)

	e := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	fmt.Println("e", e)
	err = json.Unmarshal([]byte(e), &messages)
	require.NoError(t, err)
	require.Equal(t, 1, len(messages.Result))

	f := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	fmt.Println("f", f)
	err = json.Unmarshal([]byte(f), &messages)
	require.NoError(t, err)
	require.Equal(t, 0, len(messages.Result))

	a := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "shh_requestMessages",
		"params": [{
					"enode":"` + mailboxNode.Server().NodeInfo().Enode + `",
					"topic":"` + topic.String() + `",
					"symkey":"` + MailServerKeyID + `",
					"password":"` + password + `",
					"from":0,
					"to":` + strconv.FormatInt(time.Now().UnixNano(), 10) + `
		}]
	}`)

	err = json.Unmarshal([]byte(f), &messages)
	require.NoError(t, err)
	require.Equal(t, 0, len(messages.Result))

	fmt.Println("a:", a)

	time.Sleep(time.Second)
	j := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	t.Log("j", j)
	time.Sleep(time.Second)
	err = json.Unmarshal([]byte(j), &messages)
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
