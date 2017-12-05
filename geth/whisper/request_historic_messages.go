package whisper

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/rpc"
	"time"
)

//RequestHistoricMessagesHandler returns rpc which send p2p request for expired messages.
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
		var (
			timeLow uint32 = 0
			timeUpp        = uint32(time.Now().Unix())
		)

		if len(args) != 1 {
			return nil, fmt.Errorf("Invalid number of args")
		}

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

		var topic whisperv5.TopicType
		topic.UnmarshalText([]byte(topicStr))

		symkeyID, ok := historicMessagesArgs["symKeyID"]
		if ok == false {
			return nil, fmt.Errorf("SymkeyID is not exist")
		}

		symkeyIDstr := symkeyID.(string)
		symkey, err := whisper.GetSymKey(symkeyIDstr)
		if err != nil {
			return nil, err
		}

		data := make([]byte, 8+whisperv5.TopicLength)
		binary.BigEndian.PutUint32(data, timeLow)
		binary.BigEndian.PutUint32(data[4:], timeUpp)
		copy(data[8:], topic[:])

		var params whisperv5.MessageParams
		params.PoW = 1
		params.Payload = data
		params.KeySym = symkey
		params.WorkTime = 5
		params.Src = node.Server().PrivateKey

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

		return true, nil
	}, nil
}

func extractIdFromEnode(s string) ([]byte, error) {
	n, err := discover.ParseNode(s)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse enode: %s", err)
	}
	return n.ID[:], nil
}
