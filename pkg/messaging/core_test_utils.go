package messaging

import (
	"context"
	"crypto/ecdsa"
	"testing"
	"time"

	"go.uber.org/zap"

	wakuv3 "github.com/status-im/status-go/pkg/messaging/waku"
	"github.com/status-im/status-go/pkg/messaging/waku/types"
)

type TestMessagingEnvironment struct {
	// To enable communication between multiple messaging core instances in tests,
	// share a single Waku instance across them.
	waku *testWakuWrapper
}

func NewTestMessagingEnvironment() (*TestMessagingEnvironment, error) {
	waku, err := newTestWakuWrapper()
	if err != nil {
		return nil, err
	}

	return &TestMessagingEnvironment{
		waku: waku,
	}, nil
}

func (f *TestMessagingEnvironment) Setup(t *testing.T) error {
	err := f.waku.Waku.Start()
	if err != nil {
		return err
	}

	t.Cleanup(func() {
		err = f.waku.Waku.Stop()
		if err != nil {
			t.Error(err)
		}
	})

	return nil
}

func (f *TestMessagingEnvironment) NewTestCore(params CoreParams, options ...Options) (*Core, error) {
	return newCore(f.waku, params, newConfig(options...))
}

func (f *TestMessagingEnvironment) SubscribePostEvents() chan *PostMessageSubscription {
	return f.waku.SubscribePostEvents()
}

func (f *TestMessagingEnvironment) SimulateOffline() func() {
	f.waku.Waku.SkipPublishToTopic(true)
	return func() {
		f.waku.Waku.SkipPublishToTopic(false)
	}
}

// Wraps waku to provide ability to subscribe to post events. It embeds the
// concrete *wakuv3.Waku (not the types.Waku interface) so that the messaging
// API methods that live on the backend — Send / Subscribe / Unsubscribe /
// envelope events, i.e. transport.MessagingAPI — are promoted too.
type testWakuWrapper struct {
	*wakuv3.Waku
	postSubscriptions []chan *PostMessageSubscription
}

// Send overrides the embedded Waku's Send to fan out post events to any
// subscribers registered via SubscribePostEvents. Primary sends go through
// Send (the transport encodes payloads via rfc26.Encode before publishing).
// Msg is nil because Send takes raw bytes, not a NewMessage; the only
// consumer (MessagesOrderController) reads ID.
func (tw *testWakuWrapper) Send(ctx context.Context, pubsubTopic, contentTopic string, payload []byte, ephemeral bool, priority *int) ([]byte, error) {
	id, err := tw.Waku.Send(ctx, pubsubTopic, contentTopic, payload, ephemeral, priority)
	if err != nil {
		return nil, err
	}
	for _, s := range tw.postSubscriptions {
		select {
		case s <- &PostMessageSubscription{ID: id}:
		default:
			// subscription channel full
		}
	}
	return id, nil
}

func (tw *testWakuWrapper) SubscribePostEvents() chan *PostMessageSubscription {
	subscription := make(chan *PostMessageSubscription, 100)
	tw.postSubscriptions = append(tw.postSubscriptions, subscription)
	return subscription
}

func (tw *testWakuWrapper) Start() error {
	// No-op, as one waku instance is shared across multiple cores in tests
	// and it must be started only once.
	return nil
}

func (tw *testWakuWrapper) Stop() error {
	// No-op, as one waku instance is shared across multiple cores in tests
	// and it must be stopped only once.
	return nil
}

type PostMessageSubscription struct {
	ID  []byte
	Msg *types.NewMessage
}

type testTimeSource struct {
}

func (ts *testTimeSource) Now() time.Time {
	return time.Now()
}

func newTestWakuWrapper() (*testWakuWrapper, error) {
	w, err := wakuv3.New(
		nil,
		&wakuv3.DefaultConfig,
		zap.NewNop(),
		&testTimeSource{},
	)
	if err != nil {
		return nil, err
	}

	return &testWakuWrapper{Waku: w}, nil
}

type TestUtils struct {
	API *API
}

func (t TestUtils) GetAllHRKeysCount(groupID []byte) (int, error) {
	keys, err := t.API.core.stack.Encryption.GetAllHRKeys(groupID)
	if err != nil {
		return 0, err
	}
	if keys == nil {
		return 0, nil
	}
	return len(keys.Keys), nil
}

func (t TestUtils) GetKeysForGroupCount(groupID []byte) (int, error) {
	keys, err := t.API.core.stack.Encryption.GetKeysForGroup(groupID)
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

func (t TestUtils) ProcessPublicBundle(myIdentityKey *ecdsa.PrivateKey, theirAPI *API, theirIdentityKey *ecdsa.PrivateKey) error {
	theirBundle, err := theirAPI.core.stack.Encryption.GetBundle(theirIdentityKey)
	if err != nil {
		return err
	}

	_, err = t.API.core.stack.Encryption.ProcessPublicBundle(myIdentityKey, theirBundle)
	return err
}
