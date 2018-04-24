package shhext

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

const (
	// defaultWorkTime is a work time reported in messages sent to MailServer nodes.
	defaultWorkTime = 5
)

var (
	// ErrInvalidMailServerPeer is returned when it fails to parse enode from params.
	ErrInvalidMailServerPeer = errors.New("invalid mailServerPeer value")
	// ErrInvalidSymKeyID is returned when it fails to get a symmetric key.
	ErrInvalidSymKeyID = errors.New("invalid symKeyID value")
)

// PublicAPI extends whisper public API.
type PublicAPI struct {
	service *Service
	log     log.Logger
}

// NewPublicAPI returns instance of the public API.
func NewPublicAPI(s *Service) *PublicAPI {
	return &PublicAPI{
		service: s,
		log:     log.New("package", "status-go/services/sshext.PublicAPI"),
	}
}

// Post shamelessly copied from whisper codebase with slight modifications.
func (api *PublicAPI) Post(ctx context.Context, req whisper.NewMessage) (hash common.Hash, err error) {
	env, err := whisper.MakeEnvelope(api.service.w, req)
	if err != nil {
		return hash, err
	}
	// send to specific node (skip PoW check)
	if len(req.TargetPeer) > 0 {
		n, err := discover.ParseNode(req.TargetPeer)
		if err != nil {
			return hash, fmt.Errorf("failed to parse target peer: %s", err)
		}
		err = api.service.w.SendP2PMessage(n.ID[:], env)
		if err == nil {
			api.service.tracker.Add(env.Hash())
			return env.Hash(), nil
		}
		return hash, err
	}

	// ensure that the message PoW meets the node's minimum accepted PoW
	if req.PowTarget < api.service.w.MinPow() {
		return hash, whisper.ErrTooLowPoW
	}
	err = api.service.w.Send(env)
	if err == nil {
		api.service.tracker.Add(env.Hash())
		return env.Hash(), nil
	}
	return hash, err
}

// MessagesRequest is a payload send to a MailServer to get messages.
type MessagesRequest struct {
	// MailServerPeer is MailServer's enode address.
	MailServerPeer string `json:"mailServerPeer"`

	// From is a lower bound of time range (optional).
	// Default is 24 hours back from now.
	From uint32 `json:"from"`

	// To is a upper bound of time range (optional).
	// Default is now.
	To uint32 `json:"to"`

	// Topic is a regular Whisper topic.
	Topic whisper.TopicType `json:"topic"`

	// SymKeyID is an ID of a symmetric key to authenticate to MailServer.
	// It's derived from MailServer password.
	SymKeyID string `json:"symKeyID"`
}

func setMessagesRequestDefaults(r *MessagesRequest) {
	// set From and To defaults
	if r.From == 0 && r.To == 0 {
		r.From = uint32(time.Now().UTC().Add(-24 * time.Hour).Unix())
		r.To = uint32(time.Now().UTC().Unix())
	}
}

// RequestMessages sends a request for historic messages to a MailServer.
func (api *PublicAPI) RequestMessages(_ context.Context, r MessagesRequest) (bool, error) {
	api.log.Info("RequestMessages", "request", r)

	setMessagesRequestDefaults(&r)

	shh := api.service.w

	mailServerNode, err := discover.ParseNode(r.MailServerPeer)
	if err != nil {
		return false, fmt.Errorf("%v: %v", ErrInvalidMailServerPeer, err)
	}

	symKey, err := shh.GetSymKey(r.SymKeyID)
	if err != nil {
		return false, fmt.Errorf("%v: %v", ErrInvalidSymKeyID, err)
	}

	envelope, err := makeEnvelop(makePayload(r), symKey, api.service.nodeID, shh.MinPow())
	if err != nil {
		return false, err
	}

	if err := shh.RequestHistoricMessages(mailServerNode.ID[:], envelope); err != nil {
		return false, err
	}

	return true, nil
}

// makeEnvelop makes an envelop for a historic messages request.
// Symmetric key is used to authenticate to MailServer.
// PK is the current node ID.
func makeEnvelop(payload []byte, symKey []byte, nodeID *ecdsa.PrivateKey, pow float64) (*whisper.Envelope, error) {
	params := whisper.MessageParams{
		PoW:      pow,
		Payload:  payload,
		KeySym:   symKey,
		WorkTime: defaultWorkTime,
		Src:      nodeID,
	}
	message, err := whisper.NewSentMessage(&params)
	if err != nil {
		return nil, err
	}
	return message.Wrap(&params)
}

// makePayload makes a specific payload for MailServer to request historic messages.
func makePayload(r MessagesRequest) []byte {
	// first 8 bytes are lowed and upper bounds as uint32
	data := make([]byte, 8+whisper.TopicLength)
	binary.BigEndian.PutUint32(data, r.From)
	binary.BigEndian.PutUint32(data[4:], r.To)
	copy(data[8:], r.Topic[:])
	return data
}
