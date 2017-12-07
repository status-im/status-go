package whisper

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/rpc"
	"time"
)

//RequestHistoricMessagesHandler returns an RPC handler which sends a p2p request for historic messages.
func RequestHistoricMessagesHandler(nodeManager common.NodeManager) (rpc.Handler, error) {
	whisper, err := nodeManager.WhisperService()
	if err != nil {
		return nil, err
	}

	node, err := nodeManager.Node()
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, args ...interface{}) (interface{}, error) {
		r, err := parseArgs(args)
		if err != nil {
			return nil, err
		}

		peer, err := extractIdFromEnode(r.Enode)
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

		err = whisper.RequestHistoricMessages(peer, env)
		if err != nil {
			return nil, err
		}

		return true, nil
	}, nil
}

type historicMessagesRequest struct {
	TimeLow  uint32
	TimeUp   uint32
	Topic    whisperv5.TopicType
	SymkeyID string
	Enode    string
	PoW      float64
}

func parseArgs(args ...interface{}) (historicMessagesRequest, error) {
	var (
		r = historicMessagesRequest{
			TimeLow: uint32(time.Now().Add(-24 * time.Hour).Unix()),
			TimeUp:  uint32(time.Now().Unix()),
		}
	)

	if len(args) != 1 {
		return historicMessagesRequest{}, fmt.Errorf("invalid number of arguments, expected 1")
	}

	historicMessagesArgs, ok := args[0].(map[string]interface{})
	if !ok {
		return historicMessagesRequest{}, fmt.Errorf("invalid args")
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
		return historicMessagesRequest{}, fmt.Errorf("topic value does not exist")
	}

	topicStringValue, ok := topicInterfaceValue.(string)
	if !ok {
		return historicMessagesRequest{}, fmt.Errorf("topic value is not string")
	}

	if err := r.Topic.UnmarshalText([]byte(topicStringValue)); err != nil {
		return historicMessagesRequest{}, nil
	}

	symkeyIDInterfaceValue, ok := historicMessagesArgs["symKeyID"]
	if !ok {
		return historicMessagesRequest{}, fmt.Errorf("symKeyID does not exist")
	}
	r.SymkeyID, ok = symkeyIDInterfaceValue.(string)
	if !ok {
		return historicMessagesRequest{}, fmt.Errorf("symKeyID is not string")
	}
	enodeInterfaceValue, ok := historicMessagesArgs["enode"]
	if !ok {
		return historicMessagesRequest{}, fmt.Errorf("enode does not exist")
	}
	r.Enode, ok = enodeInterfaceValue.(string)
	if !ok {
		return historicMessagesRequest{}, fmt.Errorf("enode is not string")
	}

	return r, nil
}

//makeEnvelop make envelop for request histtoric messages. symmetric key to authenticate to MailServer node and pk is the current node ID.
func makeEnvelop(r historicMessagesRequest, symkey []byte, pk *ecdsa.PrivateKey) (*whisperv5.Envelope, error) {
	var params whisperv5.MessageParams
	params.PoW = r.PoW
	params.Payload = makePayloadData(r)
	params.KeySym = symkey
	params.WorkTime = 5
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
func extractIdFromEnode(s string) ([]byte, error) {
	n, err := discover.ParseNode(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse enode: %s", err)
	}
	return n.ID[:], nil
}
