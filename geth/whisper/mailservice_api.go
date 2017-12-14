package whisper

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/log"
)

// MailServicePublicAPI defines a MailServer public API.
type MailServicePublicAPI struct {
	nodeManager common.NodeManager
}

// NewMailServicePublicAPI returns a new MailServicePublicAPI.
func NewMailServicePublicAPI(nodeManager common.NodeManager) *MailServicePublicAPI {
	return &MailServicePublicAPI{nodeManager}
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
	Topic whisperv5.TopicType `json:"topic"`

	// SymKeyID is an ID of a symmetric key to authenticate to MailServer.
	// It's derived from MailServer password.
	SymKeyID string `json:"symKeyID"`
}

// RequestMessages sends a request for historic messages to a MailServer.
func (api *MailServicePublicAPI) RequestMessages(ctx context.Context, r MessagesRequest) (bool, error) {
	log.Info("RequestMessages", "request", r)

	// set defaults
	if r.From == 0 && r.To == 0 {
		r.From = uint32(time.Now().UTC().Add(-24 * time.Hour).Unix())
		r.To = uint32(time.Now().UTC().Unix())
	}

	shh, err := api.nodeManager.WhisperService()
	if err != nil {
		return false, err
	}

	node, err := api.nodeManager.Node()
	if err != nil {
		return false, err
	}

	symKey, err := shh.GetSymKey(r.SymKeyID)
	if err != nil {
		return false, err
	}

	envelope, err := makeEnvelopAPI(r, symKey, node.Server().PrivateKey, shh.MinPow())
	if err != nil {
		return false, err
	}

	mailServerNode, err := discover.ParseNode(r.MailServerPeer)
	if err != nil {
		return false, fmt.Errorf("%v: %v", errInvalidEnode, err)
	}

	if err := shh.RequestHistoricMessages(mailServerNode.ID[:], envelope); err != nil {
		return false, err
	}

	return true, nil
}

// makeEnvelopAPI makes an envelop for a historic messages request.
// Symmetric key is used to authenticate to MailServer.
// PK is the current node ID.
func makeEnvelopAPI(r MessagesRequest, symKey []byte, pk *ecdsa.PrivateKey, pow float64) (*whisperv5.Envelope, error) {
	params := whisperv5.MessageParams{
		PoW:      pow,
		Payload:  makePayloadAPI(r),
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

// makePayloadAPI makes a specific payload for MailServer to request historic messages.
func makePayloadAPI(r MessagesRequest) []byte {
	// first 8 bytes are lowed and upper bounds as uint32
	data := make([]byte, 8+whisperv5.TopicLength)
	binary.BigEndian.PutUint32(data, r.From)
	binary.BigEndian.PutUint32(data[4:], r.To)
	copy(data[8:], r.Topic[:])
	return data
}
