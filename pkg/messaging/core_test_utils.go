package messaging

import (
	"context"
	"crypto/ecdsa"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
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

func (f *TestMessagingEnvironment) TearDown() error {
	return f.waku.Waku.Stop()
}

func (f *TestMessagingEnvironment) NewTestCore(params CoreParams, options ...Options) (*Core, error) {
	return newCore(f.waku, params, newConfig(options...))
}

func (f *TestMessagingEnvironment) SubscribePostEvents() chan *PostMessageSubscription {
	return f.waku.SubscribePostEvents()
}

func (f *TestMessagingEnvironment) SimulateOffline() func() {
	f.waku.Waku.(*wakuv3.Waku).SkipPublishToTopic(true)
	return func() {
		f.waku.Waku.(*wakuv3.Waku).SkipPublishToTopic(false)
	}
}

// Wraps waku to provide ability to subscribe to post events.
type testWakuWrapper struct {
	types.Waku
	api *testPublicWakuAPI
}

func (tw *testWakuWrapper) PublicWakuAPI() types.PublicWakuAPI {
	return tw.api
}

func (tw *testWakuWrapper) SubscribePostEvents() chan *PostMessageSubscription {
	subscription := make(chan *PostMessageSubscription, 100)
	tw.api.postSubscriptions = append(tw.api.postSubscriptions, subscription)
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

type testPublicWakuAPI struct {
	*wakuv3.PublicWakuAPI

	postSubscriptions []chan *PostMessageSubscription
}

func (tp *testPublicWakuAPI) Post(ctx context.Context, req types.NewMessage) ([]byte, error) {
	id, err := tp.PublicWakuAPI.Post(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, s := range tp.postSubscriptions {
		select {
		case s <- &PostMessageSubscription{ID: id, Msg: &req}:
		default:
			// subscription channel full
		}
	}
	return id, err
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
		func([]byte, peer.AddrInfo, error) {},
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &testWakuWrapper{
		Waku: w,
		api: &testPublicWakuAPI{
			PublicWakuAPI: wakuv3.NewPublicWakuAPI(w),
		},
	}, nil
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
