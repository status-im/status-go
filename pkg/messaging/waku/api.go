// Copyright 2019 The Waku Library Authors.
//
// The Waku library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Waku library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty off
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Waku library. If not, see <http://www.gnu.org/licenses/>.
//
// This software uses the go-ethereum library, which is licensed
// under the GNU Lesser General Public Library, version 3 or any later.

package wakuv2

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/waku-org/go-waku/waku/v2/protocol/pb"

	"github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/pkg/messaging/layers/transport/encoding"
	"github.com/status-im/status-go/pkg/messaging/waku/common"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

// List of errors
var (
	ErrSymAsym             = errors.New("specify either a symmetric or an asymmetric key")
	ErrInvalidSymmetricKey = errors.New("invalid symmetric key")
	ErrInvalidPublicKey    = errors.New("invalid public key")
	ErrNoTopics            = errors.New("missing topic(s)")
)

// PublicWakuAPI provides the waku RPC service that can be
// use publicly without security implications.
type PublicWakuAPI struct {
	w *Waku

	mu       sync.Mutex
	lastUsed map[string]time.Time // keeps track when a filter was polled for the last time.
}

// NewPublicWakuAPI create a new RPC waku service.
func NewPublicWakuAPI(w *Waku) *PublicWakuAPI {
	api := &PublicWakuAPI{
		w:        w,
		lastUsed: make(map[string]time.Time),
	}
	return api
}

// Post posts a message on the Waku network.
// returns the hash of the message in case of success.
func (api *PublicWakuAPI) Post(ctx context.Context, req types.NewMessage) ([]byte, error) {
	var (
		symKeyGiven = len(req.SymKeyID) > 0
		pubKeyGiven = len(req.PublicKey) > 0
		err         error
	)

	// user must specify either a symmetric or an asymmetric key
	if (symKeyGiven && pubKeyGiven) || (!symKeyGiven && !pubKeyGiven) {
		return nil, ErrSymAsym
	}

	var keyInfo *encoding.KeyInfo = new(encoding.KeyInfo)

	// Set key that is used to sign the message
	if len(req.SigID) > 0 {
		privKey, err := api.w.GetPrivateKey(req.SigID)
		if err != nil {
			return nil, err
		}
		keyInfo.PrivKey = privKey
	}

	contentTopic := common.TopicType(req.Topic)

	// Set symmetric key that is used to encrypt the message
	if symKeyGiven {
		keyInfo.Kind = encoding.Symmetric

		if contentTopic == (common.TopicType{}) { // topics are mandatory with symmetric encryption
			return nil, ErrNoTopics
		}
		if keyInfo.SymKey, err = api.w.GetSymKey(req.SymKeyID); err != nil {
			return nil, err
		}
		if !common.ValidateDataIntegrity(keyInfo.SymKey, common.AESKeyLength) {
			return nil, ErrInvalidSymmetricKey
		}
	}

	// Set asymmetric key that is used to encrypt the message
	if pubKeyGiven {
		keyInfo.Kind = encoding.Asymmetric

		var pubK *ecdsa.PublicKey
		if pubK, err = crypto.UnmarshalPubkey(req.PublicKey); err != nil {
			return nil, ErrInvalidPublicKey
		}
		keyInfo.PubKey = *pubK
	}

	var version uint32 = 1 // Use wakuv1 encryption

	p := new(encoding.Payload)
	p.Data = req.Payload
	p.Key = keyInfo

	payload, err := p.Encode(version)
	if err != nil {
		return nil, err
	}

	wakuMsg := &pb.WakuMessage{
		Payload:      payload,
		Version:      &version,
		ContentTopic: contentTopic.ContentTopic(),
		Timestamp:    proto.Int64(api.w.timestamp()),
		Meta:         []byte{}, // TODO: empty for now. Once we use Waku Archive v2, we should deprecate the timestamp and use an ULID here
		Ephemeral:    &req.Ephemeral,
	}

	hash, err := api.w.Send(req.PubsubTopic, wakuMsg, req.Priority)

	if err != nil {
		return nil, err
	}

	return hash, nil
}

// ToWakuMessage converts an internal message into an API version.
func ToWakuMessage(message *common.ReceivedMessage) *types.Message {
	msg := types.Message{
		Payload:   message.Data,
		Padding:   message.Padding,
		Timestamp: message.Sent,
		Hash:      message.Hash().Bytes(),
		Topic:     types.TopicType(message.ContentTopic),
	}

	if message.Dst != nil {
		b := crypto.FromECDSAPub(message.Dst)
		if b != nil {
			msg.Dst = b
		}
	}

	if message.Src != nil {
		b := crypto.FromECDSAPub(message.Src)
		if b != nil {
			msg.Sig = b
		}
	}

	return &msg
}

// GetFilterMessages returns the messages that match the filter criteria and
// are received between the last poll and now.
func (api *PublicWakuAPI) GetFilterMessages(id string) ([]*types.Message, error) {
	api.mu.Lock()
	f := api.w.getFilter(id)
	if f == nil {
		api.mu.Unlock()
		return nil, fmt.Errorf("filter not found")
	}
	api.lastUsed[id] = time.Now()
	api.mu.Unlock()

	receivedMessages := f.Retrieve()
	messages := make([]*types.Message, 0, len(receivedMessages))
	for _, msg := range receivedMessages {
		messages = append(messages, ToWakuMessage(msg))
	}

	return messages, nil
}
