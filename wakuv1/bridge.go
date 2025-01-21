package wakuv1

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum/go-ethereum/event"
	ethtypes "github.com/status-im/status-go/eth-node/types"

	"github.com/status-im/status-go/waku/types"
	"github.com/status-im/status-go/wakuv1/common"
)

// NewWakuEnvelopeEventWrapper returns a types.EnvelopeEvent object that mimics Geth's EnvelopeEvent
func NewWakuEnvelopeEventWrapper(envelopeEvent *common.EnvelopeEvent) *types.EnvelopeEvent {
	if envelopeEvent == nil {
		panic("envelopeEvent should not be nil")
	}

	wrappedData := envelopeEvent.Data
	switch data := envelopeEvent.Data.(type) {
	case []common.EnvelopeError:
		wrappedData := make([]types.EnvelopeError, len(data))
		for index := range data {
			wrappedData[index] = *NewWakuEnvelopeErrorWrapper(&data[index])
		}
	case *MailServerResponse:
		wrappedData = NewWakuMailServerResponseWrapper(data)
	}
	return &types.EnvelopeEvent{
		Event: types.EventType(envelopeEvent.Event),
		Hash:  ethtypes.Hash(envelopeEvent.Hash),
		Batch: ethtypes.Hash(envelopeEvent.Batch),
		Peer:  ethtypes.EnodeID(envelopeEvent.Peer),
		Data:  wrappedData,
	}
}

// NewWakuEnvelopeErrorWrapper returns a types.EnvelopeError object that mimics Geth's EnvelopeError
func NewWakuEnvelopeErrorWrapper(envelopeError *common.EnvelopeError) *types.EnvelopeError {
	if envelopeError == nil {
		panic("envelopeError should not be nil")
	}

	return &types.EnvelopeError{
		Hash:        ethtypes.Hash(envelopeError.Hash),
		Code:        mapGethErrorCode(envelopeError.Code),
		Description: envelopeError.Description,
	}
}

func mapGethErrorCode(code uint) uint {
	switch code {
	case common.EnvelopeTimeNotSynced:
		return types.EnvelopeTimeNotSynced
	case common.EnvelopeOtherError:
		return types.EnvelopeOtherError
	}
	return types.EnvelopeOtherError
}

// NewWakuMailServerResponseWrapper returns a types.MailServerResponse object that mimics Geth's MailServerResponse
func NewWakuMailServerResponseWrapper(mailServerResponse *MailServerResponse) *types.MailServerResponse {
	if mailServerResponse == nil {
		panic("mailServerResponse should not be nil")
	}

	return &types.MailServerResponse{
		LastEnvelopeHash: ethtypes.Hash(mailServerResponse.LastEnvelopeHash),
		Cursor:           mailServerResponse.Cursor,
		Error:            mailServerResponse.Error,
	}
}

type gethSubscriptionWrapper struct {
	subscription event.Subscription
}

// NewGethSubscriptionWrapper returns an object that wraps Geth's Subscription in a types interface
func NewGethSubscriptionWrapper(subscription event.Subscription) types.Subscription {
	if subscription == nil {
		panic("subscription cannot be nil")
	}

	return &gethSubscriptionWrapper{
		subscription: subscription,
	}
}

func (w *gethSubscriptionWrapper) Err() <-chan error {
	return w.subscription.Err()
}

func (w *gethSubscriptionWrapper) Unsubscribe() {
	w.subscription.Unsubscribe()
}

type wakuEnvelope struct {
	env *common.Envelope
}

// NewWakuEnvelope returns an object that wraps Geth's Waku Envelope in a types interface.
func NewWakuEnvelope(e *common.Envelope) types.Envelope {
	return &wakuEnvelope{env: e}
}

func (w *wakuEnvelope) Unwrap() interface{} {
	return w.env
}

func (w *wakuEnvelope) Hash() ethtypes.Hash {
	return ethtypes.Hash(w.env.Hash())
}

func (w *wakuEnvelope) Bloom() []byte {
	return w.env.Bloom()
}

func (w *wakuEnvelope) PoW() float64 {
	return w.env.PoW()
}

func (w *wakuEnvelope) Expiry() uint32 {
	return w.env.Expiry
}

func (w *wakuEnvelope) TTL() uint32 {
	return w.env.TTL
}

func (w *wakuEnvelope) Topic() types.TopicType {
	return types.TopicType(w.env.Topic)
}

func (w *wakuEnvelope) Size() int {
	return len(w.env.Data)
}

func (w *wakuEnvelope) DecodeRLP(s *rlp.Stream) error {
	return w.env.DecodeRLP(s)
}

func (w *wakuEnvelope) EncodeRLP(writer io.Writer) error {
	return rlp.Encode(writer, w.env)
}
