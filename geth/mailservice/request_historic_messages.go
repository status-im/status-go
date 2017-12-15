package mailservice

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/rpc"
)

var (
	//ErrInvalidNumberOfArgs - error invalid aruments in request
	ErrInvalidNumberOfArgs = fmt.Errorf("invalid number of arguments, expected 1")
	//ErrInvalidArgs - error invalid request format
	ErrInvalidArgs = fmt.Errorf("invalid args")
	//ErrTopicNotExist - error topic field doesn't exist in request
	ErrTopicNotExist = fmt.Errorf("topic value does not exist")
	//ErrTopicNotString - error topic is not string type
	ErrTopicNotString = fmt.Errorf("topic value is not string")
	//ErrMailboxSymkeyIDNotExist - error symKeyID field doesn't exist in request
	ErrMailboxSymkeyIDNotExist = fmt.Errorf("symKeyID does not exist")
	//ErrMailboxSymkeyIDNotString - error symKeyID is not string type
	ErrMailboxSymkeyIDNotString = fmt.Errorf("symKeyID is not string")
	//ErrPeerNotExist - error peer field doesn't exist in request
	ErrPeerOrEnode = fmt.Errorf("enode or peer field should be not empty")
)

const defaultWorkTime = 5

//RequestHistoricMessagesHandler returns an RPC handler which sends a p2p request for historic messages.
func RequestHistoricMessagesHandler(nodeManager common.NodeManager) rpc.Handler {
	return func(ctx context.Context, args ...interface{}) (interface{}, error) {
		whisper, err := nodeManager.WhisperService()
		if err != nil {
			return nil, err
		}

		node, err := nodeManager.Node()
		if err != nil {
			return nil, err
		}

		r, err := parseArgs(args...)
		if err != nil {
			return nil, err
		}

		symkey, err := whisper.GetSymKey(r.SymkeyID)
		if err != nil {
			return nil, err
		}
		r.PoW = whisper.MinPow()
		env, err := makeEnvelop(r, symkey, node.Server().PrivateKey)
		if err != nil {
			return nil, err
		}

		err = whisper.RequestHistoricMessages(r.Peer, env)
		if err != nil {
			return nil, err
		}

		return true, nil
	}
}

type historicMessagesRequest struct {
	Peer     []byte              //mailbox peer
	TimeLow  uint32              //resend messages from
	TimeUp   uint32              //resend messages to
	Topic    whisperv5.TopicType //resend messages by topic
	SymkeyID string              //Mailbox symmetric key id
	PoW      float64             //whisper proof of work
}

func parseArgs(args ...interface{}) (historicMessagesRequest, error) {
	var (
		r = historicMessagesRequest{
			TimeLow: uint32(time.Now().Add(-24 * time.Hour).Unix()),
			TimeUp:  uint32(time.Now().Unix()),
		}
	)

	if len(args) != 1 {
		return historicMessagesRequest{}, ErrInvalidNumberOfArgs
	}

	historicMessagesArgs, ok := args[0].(map[string]interface{})
	if !ok {
		return historicMessagesRequest{}, ErrInvalidArgs
	}

	if t, ok := historicMessagesArgs["from"]; ok {
		if parsed, ok := t.(uint32); ok {
			r.TimeLow = parsed
		}
	}
	if t, ok := historicMessagesArgs["to"]; ok {
		if parsed, ok := t.(uint32); ok {
			r.TimeUp = parsed
		}
	}
	topicInterfaceValue, ok := historicMessagesArgs["topic"]
	if !ok {
		return historicMessagesRequest{}, ErrTopicNotExist
	}

	topicStringValue, ok := topicInterfaceValue.(string)
	if !ok {
		return historicMessagesRequest{}, ErrTopicNotString
	}

	if err := r.Topic.UnmarshalText([]byte(topicStringValue)); err != nil {
		return historicMessagesRequest{}, nil
	}

	symkeyIDInterfaceValue, ok := historicMessagesArgs["symKeyID"]
	if !ok {
		return historicMessagesRequest{}, ErrMailboxSymkeyIDNotExist
	}
	r.SymkeyID, ok = symkeyIDInterfaceValue.(string)
	if !ok {
		return historicMessagesRequest{}, ErrMailboxSymkeyIDNotString
	}

	peer, err := getPeerID(historicMessagesArgs)
	if err != nil {
		return historicMessagesRequest{}, err
	}
	r.Peer = peer
	return r, nil
}

//makeEnvelop make envelop for request histtoric messages. symmetric key to authenticate to MailServer node and pk is the current node ID.
func makeEnvelop(r historicMessagesRequest, symkey []byte, pk *ecdsa.PrivateKey) (*whisperv5.Envelope, error) {
	var params whisperv5.MessageParams
	params.PoW = r.PoW
	params.Payload = makePayloadData(r)
	params.KeySym = symkey
	params.WorkTime = defaultWorkTime
	params.Src = pk

	message, err := whisperv5.NewSentMessage(&params)
	if err != nil {
		return nil, err
	}
	return message.Wrap(&params)
}

//makePayloadData make specific payload for mailserver
func makePayloadData(r historicMessagesRequest) []byte {
	data := make([]byte, 8+whisperv5.TopicLength)
	binary.BigEndian.PutUint32(data, r.TimeLow)
	binary.BigEndian.PutUint32(data[4:], r.TimeUp)
	copy(data[8:], r.Topic[:])
	return data
}

//getPeerID is used to get peerID from string values of peerID or enode
func getPeerID(m map[string]interface{}) ([]byte, error) {
	peerInterfaceValue, okPeer := m["peer"]
	enodeInterfaceValue, okEnode := m["enode"]

	//only if existing peer or enode(!xor)
	if okPeer == okEnode {
		return nil, ErrPeerOrEnode
	}
	var peerOrEnode string
	if p, ok := peerInterfaceValue.(string); ok && okPeer {
		peerOrEnode = p
	} else if str, ok := enodeInterfaceValue.(string); ok && okEnode {
		peerOrEnode = str
	} else {
		return nil, ErrPeerOrEnode
	}

	n, err := discover.ParseNode(peerOrEnode)
	if err != nil {
		return nil, err
	}
	return n.ID[:], nil
}
