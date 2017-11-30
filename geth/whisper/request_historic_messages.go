package whisper

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/rpc"
	"time"
)

type HistoricMessagesRequester interface {
	RequestHistoricMessages(peerID []byte, envelope *whisperv5.Envelope) error
	GetPrivateKey(id string) (*ecdsa.PrivateKey, error)
	GetSymKey(id string) ([]byte, error)
	Version() uint
}

func RequestHistoricMessages(whisper HistoricMessagesRequester) rpc.Handler {
	return func(ctx context.Context, args ...interface{}) (interface{}, error) {
		var (
			timeLow uint32 = 0
			timeUpp uint32 = uint32(time.Now().Unix())
		)
		fmt.Println(args)
		if len(args) != 1 {
			return nil, fmt.Errorf("Invalid number of args")
		}
		// prepare and send request to mail server for archive messages
		historicMessagesArgs, ok := args[0].(map[string]interface{})
		if ok == false {
			return nil, fmt.Errorf("Invalid args")
		}

		if t, ok := historicMessagesArgs["from"]; ok == true {
			if parsed, ok := t.(uint32); ok {
				timeLow = parsed
			}
		}
		if t, ok := historicMessagesArgs["to"]; ok == true {
			if parsed, ok := t.(uint32); ok {
				timeUpp = parsed
			}
		}
		t, ok := historicMessagesArgs["topic"]
		if ok == false {
			return nil, fmt.Errorf("Topic value is not exist")
		}
		topicStr, ok := t.(string)
		if ok == false {
			return nil, fmt.Errorf("Topic value is not string")
		}
		topic := whisperv5.BytesToTopic([]byte(topicStr))

		//todo check
		symkeyID := historicMessagesArgs["symkey"].(string)
		symkey, _ := whisper.GetSymKey(symkeyID)

		data := make([]byte, 8+whisperv5.TopicLength)
		binary.BigEndian.PutUint32(data, timeLow)
		binary.BigEndian.PutUint32(data[4:], timeUpp)
		copy(data[8:], topic[:])

		var params whisperv5.MessageParams
		params.PoW = 1
		params.Payload = data
		params.KeySym = symkey
		params.WorkTime = 5

		msg, err := whisperv5.NewSentMessage(&params)
		if err != nil {
			return nil, err
		}
		env, err := msg.Wrap(&params)
		if err != nil {
			return nil, err
		}

		peer, err := extractIdFromEnode(historicMessagesArgs["enode"].(string))
		if err != nil {
			return nil, err
		}

		err = whisper.RequestHistoricMessages(peer, env)
		if err != nil {
			return nil, err
		}

		return env, nil
	}
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

*/
