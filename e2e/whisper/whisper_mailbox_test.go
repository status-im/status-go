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

func TestOne(t *testing.T) {

	mailboxBackend := api.NewStatusBackend()
	mailboxConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	mailboxConfig.LightEthConfig.Enabled = false
	mailboxConfig.WhisperConfig.Enabled = true
	mailboxConfig.KeyStoreDir = "../../.ethereumtest/mailbox/"
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

	backend := api.NewStatusBackend()
	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())

	mailboxNode, err := mailboxBackend.NodeManager().Node()
	mailboxEnode := mailboxNode.Server().NodeInfo().Enode

	require.NoError(t, err)
	require.False(t, backend.IsNodeRunning())

	nodeStarted, err := backend.StartNode(nodeConfig)
	require.NoError(t, err)

	<-nodeStarted // wait till node is started
	time.Sleep(time.Second)
	require.True(t, backend.IsNodeRunning())

	node, err := backend.NodeManager().Node()
	require.NoError(t, err)
	require.NotEqual(t, mailboxEnode, node.Server().NodeInfo().Enode)

	err = backend.NodeManager().AddPeer(mailboxEnode)
	require.NoError(t, err)
	time.Sleep(time.Second)

	w, _ := backend.NodeManager().WhisperService()

	p, err := extractIdFromEnode(mailboxNode.Server().NodeInfo().Enode)
	require.NoError(t, err)
	err = w.AllowP2PMessagesFromPeer(p)
	require.NoError(t, err)

	//password := "asdfasdf"
	//MailServerKeyID, err := w.AddSymKeyFromPassword(password)

	require.NoError(t, err)

	rpcClient := backend.NodeManager().RPCClient()
	require.NotNil(t, rpcClient)

	err = backend.NodeManager().AddPeer(mailboxEnode)
	require.NoError(t, err)

	topic := whisperv5.BytesToTopic([]byte("topic name"))
	symkeyID, err := w.GenerateSymKey()
	fmt.Println("Topic::", topic.String())
	b := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_newMessageFilter", "params": [
			{"symKeyID": "` + symkeyID + `", "topics": [ "` + topic.String() + `"], "allowP2P":true}
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

	time.Sleep(10 * time.Second)
	a := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "shh_requestMessages",
		"params": [{
					"enode":"` + mailboxNode.Server().NodeInfo().Enode + `",
					"topic":"` + topic.String() + `",
					"symKeyID":"` + symkeyID + `",
					"password":"` + "password" + `",
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

func TestTwo(t *testing.T) {

	mailboxBackend := api.NewStatusBackend()
	mailboxConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())
	mailboxConfig.LightEthConfig.Enabled = false
	mailboxConfig.WhisperConfig.Enabled = true
	mailboxConfig.KeyStoreDir = "../../.ethereumtest/mailbox/"
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

	backend := api.NewStatusBackend()
	nodeConfig, err := e2e.MakeTestNodeConfig(GetNetworkID())

	mailboxNode, err := mailboxBackend.NodeManager().Node()
	mailboxEnode := mailboxNode.Server().NodeInfo().Enode

	require.NoError(t, err)
	require.False(t, backend.IsNodeRunning())

	nodeStarted, err := backend.StartNode(nodeConfig)
	require.NoError(t, err)

	<-nodeStarted // wait till node is started
	time.Sleep(time.Second)
	require.True(t, backend.IsNodeRunning())

	node, err := backend.NodeManager().Node()
	require.NoError(t, err)
	require.NotEqual(t, mailboxEnode, node.Server().NodeInfo().Enode)

	err = backend.NodeManager().AddPeer(mailboxEnode)
	require.NoError(t, err)
	time.Sleep(time.Second)

	w, _ := backend.NodeManager().WhisperService()

	p, err := extractIdFromEnode(mailboxNode.Server().NodeInfo().Enode)
	require.NoError(t, err)
	err = w.AllowP2PMessagesFromPeer(p)
	require.NoError(t, err)

	password := "asdfasdf"
	MailServerKeyID, err := w.AddSymKeyFromPassword(password)

	require.NoError(t, err)

	rpcClient := backend.NodeManager().RPCClient()
	require.NotNil(t, rpcClient)

	err = backend.NodeManager().AddPeer(mailboxEnode)
	require.NoError(t, err)

	topic := whisperv5.BytesToTopic([]byte("topic name"))
	keyID, err := w.NewKeyPair()
	key, err := w.GetPrivateKey(keyID)
	require.NoError(t, err)
	pubkey := hexutil.Bytes(crypto.FromECDSAPub(&key.PublicKey))

	fmt.Println("Topic::", topic.String())
	b := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_newMessageFilter", "params": [
			{"privateKeyID": "` + keyID + `", "topics": [ "` + topic.String() + `"], "allowP2P":true}
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
	fmt.Println(string(pubkey))

	d := rpcClient.CallRaw(`{
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
	fmt.Println("d", d)

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
					"symKeyID":"` + MailServerKeyID + `",
					"from":0,
					"to":` + strconv.FormatInt(time.Now().UnixNano(), 10) + `
		}]
	}`)

	fmt.Println("a:", a)

	time.Sleep(time.Second)
	j := rpcClient.CallRaw(`{
		"jsonrpc": "2.0",
		"method": "shh_getFilterMessages",
		"params": ["` + messageFilterID + `"],
		"id": 1}`)
	t.Log("j", j)
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

/**

func TestGetWhisperMessageMailServer_Asymmetric(t *testing.T) {
	alice := newCLI()
	bob := newCLI()
	mailbox := newCLI()

	topic := whisperv5.BytesToTopic([]byte("TestGetWhisperMessageMailServer topic name"))

	t.Log("Start nodes")
	closeCh := make(chan struct{})
	doneFn := composeNodesClose(
		startNode("mailserver", WNODE_BIN, closeCh, mailServerParams(mailbox.PortString())...),
		startNode("alice", STATUSD_BIN, closeCh, "-shh", "-httpport="+alice.PortString(), "-http=true", "-datadir=w1"),
	)

	t.Log("Start bob node")
	startLocalNode(bob.Port())
	time.Sleep(4 * time.Second)

	defer func() {
		close(closeCh)
		doneFn()
	}()

	t.Log("Alice create aliceKey")
	time.Sleep(time.Millisecond)
	_, err := alice.createAsymkey()
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Bob creates key pair to his node")
	bobKeyID, err := bob.createAsymkey()
	if err != nil {
		t.Fatal(err)
	}

	bobPrivateKey, bobPublicKey, err := bob.getKeyPair(bobKeyID)
	if err != nil {
		t.Fatal(err)
	}

	// At this time nodes do public keys exchange

	// Bob goes offline
	err = stopLocalNode()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Bob has been stopped", backend)

	t.Log("Alice send message to bob")
	_, err = alice.postAsymMessage(bobPublicKey, topic.String(), 4, "")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Wait that Alice message is being expired")
	time.Sleep(3 * time.Second)

	t.Log("Resume bob node")
	bob = newCLI()
	startLocalNode(bob.Port())
	time.Sleep(4 * time.Second)
	defer stopLocalNode()

	t.Log("Is Bob`s node running", backend.NodeManager().IsNodeRunning())

	t.Log("Bob restores private key")
	_, err = bob.addPrivateKey(bobPrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Bob makes filter on his node")
	bobFilterID, err := bob.makeAsyncMessageFilter(bobKeyID, topic.String())
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Second)
	t.Log("Bob check messages. There are no messages")
	r, err := bob.getFilterMessages(bobFilterID)
	if len(r.Result.([]interface{})) != 0 {
		t.Fatal("Has got a messages")
	}
	if err != nil {
		t.Fatal(err)
	}

	bobNode, err := backend.NodeManager().Node()
	if err != nil {
		t.Fatal(err)
	}

	mailServerPeerID, bobKeyFromPassword := bob.addMailServerNode(t, mailbox)

	// prepare and send request to mail server for archive messages
	timeLow := uint32(time.Now().Add(-2 * time.Minute).Unix())
	timeUpp := uint32(time.Now().Add(2 * time.Minute).Unix())
	t.Log("Time:", timeLow, timeUpp)

	data := make([]byte, 8+whisperv5.TopicLength)
	binary.BigEndian.PutUint32(data, timeLow)
	binary.BigEndian.PutUint32(data[4:], timeUpp)
	copy(data[8:], topic[:])

	var params whisperv5.MessageParams
	params.PoW = 1
	params.Payload = data
	params.KeySym = bobKeyFromPassword
	params.Src = bobNode.Server().PrivateKey
	params.WorkTime = 5

	msg, err := whisperv5.NewSentMessage(&params)
	if err != nil {
		t.Fatal(err)
	}
	env, err := msg.Wrap(&params)
	if err != nil {
		t.Fatal(err)
	}

	bobWhisper, _ := backend.NodeManager().WhisperService()
	err = bobWhisper.RequestHistoricMessages(mailServerPeerID, env)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)
	t.Log("Bob get alice message which sent from mailbox")
	r, err = bob.getFilterMessages(bobFilterID)
	t.Log(err, r)
	if len(r.Result.([]interface{})) == 0 {
		t.Fatal("Hasnt got any messages")
	}
}

*/

/**
 func (c cli) addMailServerNode(t *testing.T, nMail cli) (mailServerPeerID, bobKeyFromPassword []byte) {
	mNodeInfo, err := nMail.getNodeInfo()
	if err != nil {
		t.Fatal(err)
	}
	mailServerEnode := mNodeInfo["enode"].(string)

	t.Log("Add mailserver peer to bob node")
	err = backend.NodeManager().AddPeer(mailServerEnode)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Add mailserver peer to bob node too??")
	time.Sleep(5 * time.Second)

	t.Log("Mark mailserver as bob trusted")
	rsp, err := c.markTrusted(mailServerEnode)
	t.Log(rsp, err)

	t.Log("extractIdFromEnode")
	mailServerPeerID, err = extractIdFromEnode(mailServerEnode)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Get bob's symkey for mailserver")
	bobWhisper, _ := backend.NodeManager().WhisperService()
	keyID, err := bobWhisper.AddSymKeyFromPassword("asdfasdf") // mailserver password
	if err != nil {
		t.Fatalf("Failed to create symmetric key for mail request: %s", err)
	}
	t.Log("Add symkey by id")
	bobKeyFromPassword, err = bobWhisper.GetSymKey(keyID)
	if err != nil {
		t.Fatalf("Failed to save symmetric key for mail request: %s", err)
	}

	return mailServerPeerID, bobKeyFromPassword
}

func (c cli) postAsymMessage(pubKey, topic string, ttl int, targetPeer string) (RpcResponse, error) {
	r, err := makeBody(MakeRpcRequest("shh_post", []shhPost{{
		PubKey:     pubKey,
		Topic:      topic,
		Payload:    hexutil.Encode([]byte("hello world!!")),
		PowTarget:  0.001,
		PowTime:    1,
		TTL:        119,
		TargetPeer: targetPeer,
	}}))
	if err != nil {
		return RpcResponse{}, err
	}

	resp, err := c.c.Post(c.addr, "application/json", r)
	if err != nil {
		return RpcResponse{}, err
	}
	return makeRpcResponse(resp.Body)
}

func (c cli) makeAsyncMessageFilter(privateKeyID string, topic string) (string, error) {
	//make filter
	r, err := makeBody(MakeRpcRequest("shh_newMessageFilter", []shhNewMessageFilter{{
		PrivateKeyID: privateKeyID,
		Topics:       []string{topic},
		AllowP2P:     true,
	}}))
	if err != nil {
		return "", err
	}

	resp, err := c.c.Post(c.addr, "application/json", r)
	if err != nil {
		return "", err
	}

	rsp, err := makeRpcResponse(resp.Body)
	if err != nil {
		return "", err
	}

	if rsp.Error.Message != "" {
		return "", errors.New(rsp.Error.Message)
	}

	return rsp.Result.(string), nil
}

*/
