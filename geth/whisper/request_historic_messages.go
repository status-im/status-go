package whisper

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
	"github.com/status-im/status-go/geth/rpc"
)

var (
	errInvalidNumberOfParams = errors.New("invalid number of parameters, expected 1 parameter")
	errInvalidParams         = errors.New("invalid parameter data type")
	errMissingTopic          = errors.New("topic value is required")
	errMissingSymKeyID       = errors.New("symKeyID value is required")
	errInvalidSymKeyID       = errors.New("symKeyID must be a string")
	errMissingEnode          = errors.New("enode value is missing")
	errInvalidEnode          = errors.New("enode must be a string and have a valid format")
	errInvalidTimeRange      = errors.New("invalid TimeLow and TimeUp values")
)

const defaultWorkTime = 5

// RequestHistoricMessagesHandler returns an RPC handler
// which sends a p2p request for historic messages.
func RequestHistoricMessagesHandler(nodeManager common.NodeManager) rpc.Handler {
	return func(ctx context.Context, args ...interface{}) (interface{}, error) {
		log.Info("RequestHistoricMessagesHandler start")

		whisper, err := nodeManager.WhisperService()
		if err != nil {
			return nil, err
		}

		node, err := nodeManager.Node()
		if err != nil {
			return nil, err
		}

		r, err := parseArgs(args)
		if err != nil {
			return nil, err
		}

		r.PoW = whisper.MinPow()

		log.Info("RequestHistoricMessagesHandler parsed request", "request", r)

		symKey, err := whisper.GetSymKey(r.SymKeyID)
		if err != nil {
			return nil, err
		}

		envelope, err := makeEnvelop(r, symKey, node.Server().PrivateKey)
		if err != nil {
			return nil, err
		}

		err = whisper.RequestHistoricMessages(r.Peer, envelope)
		if err != nil {
			return nil, err
		}

		return true, nil
	}
}

type historicMessagesRequest struct {
	// MailServer enode address.
	Peer []byte

	// TimeLow is a lower bound of time range (optional).
	// Default is 24 hours back from now.
	TimeLow uint32

	// TimeUp is a upper bound of time range (optional).
	// Default is now.
	TimeUp uint32

	// Topic is a regular Whisper topic.
	Topic whisperv5.TopicType

	// SymKeyID is an ID of a symmetric key used to authenticate to MailServer.
	// It's derived from MailServer password.
	SymKeyID string

	// PoW is Whisper Proof of Work value.
	PoW float64
}

func parseArgs(args []interface{}) (r historicMessagesRequest, err error) { //nolint: gocyclo
	if len(args) != 1 {
		return r, errInvalidNumberOfParams
	}

	param, ok := args[0].(map[string]interface{})
	if !ok {
		return r, errInvalidParams
	}

	enodeInterface, ok := param["enode"]
	if !ok {
		return r, errMissingEnode
	}
	enode, ok := enodeInterface.(string)
	if !ok {
		return r, errInvalidEnode
	}
	nodeInfo, err := discover.ParseNode(enode)
	if err != nil {
		return r, fmt.Errorf("%v: %v", errInvalidEnode, err)
	}
	r.Peer = nodeInfo.ID[:]

	topicInterface, ok := param["topic"]
	if !ok {
		return r, errMissingTopic
	}
	switch t := topicInterface.(type) {
	case string:
		r.Topic = whisperv5.BytesToTopic([]byte(t))
	case []byte:
		r.Topic = whisperv5.BytesToTopic(t)
	case whisperv5.TopicType:
		r.Topic = t
	}

	symKeyIDInterface, ok := param["symKeyID"]
	if !ok {
		return r, errMissingSymKeyID
	}
	r.SymKeyID, ok = symKeyIDInterface.(string)
	if !ok {
		return r, errInvalidSymKeyID
	}

	if t, ok := param["from"]; ok {
		if value, ok := parseToUint32(t); ok {
			r.TimeLow = value
		} else {
			return r, fmt.Errorf("from value must be unix time in seconds, got: %T", t)
		}
	} else {
		r.TimeLow = uint32(time.Now().UTC().Add(-24 * time.Hour).Unix())
	}
	if t, ok := param["to"]; ok {
		if value, ok := parseToUint32(t); ok {
			r.TimeUp = value
		} else {
			return r, fmt.Errorf("to value must be unix time in seconds, got: %T", t)
		}
	} else {
		r.TimeUp = uint32(time.Now().UTC().Unix())
	}
	if r.TimeLow > r.TimeUp {
		return r, errInvalidTimeRange
	}

	return r, nil
}

func parseToUint32(val interface{}) (uint32, bool) {
	switch t := val.(type) {
	case float64:
		if t >= 0 && t < math.MaxUint32 {
			return uint32(t), true
		}
	case int:
		if t >= 0 && t < math.MaxUint32 {
			return uint32(t), true
		}
	case int64:
		if t >= 0 && t < math.MaxUint32 {
			return uint32(t), true
		}
	}

	return 0, false
}

// makeEnvelop makes an envelop for a historic messages request.
// Symmetric key is used to authenticate to MailServer.
// PK is the current node ID.
func makeEnvelop(r historicMessagesRequest, symKey []byte, pk *ecdsa.PrivateKey) (*whisperv5.Envelope, error) {
	params := whisperv5.MessageParams{
		PoW:      r.PoW,
		Payload:  makePayload(r),
		KeySym:   symKey,
		WorkTime: defaultWorkTime,
		Src:      pk,
	}
	message, err := whisperv5.NewSentMessage(&params)
	if err != nil {
		return nil, err
	}

	return message.Wrap(&params)
}

// makePayload makes a specific payload for MailServer to request historic messages.
func makePayload(r historicMessagesRequest) []byte {
	// first 8 bytes are lowed and upper bounds as uint32
	data := make([]byte, 8+whisperv5.TopicLength)
	binary.BigEndian.PutUint32(data, r.TimeLow)
	binary.BigEndian.PutUint32(data[4:], r.TimeUp)
	copy(data[8:], r.Topic[:])
	return data
}
