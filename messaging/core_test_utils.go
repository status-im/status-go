package messaging

import (
	"context"
	"crypto/ecdsa"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	wakuv2 "github.com/status-im/status-go/messaging/waku"
	wakutypes "github.com/status-im/status-go/messaging/waku/types"
)

type TestMessagingEnvironment struct {
	// To enable communication between multiple messaging core instances in tests,
	// share a single Waku instance across them.
	waku *testWakuWrapper

	cores []*Core
}

func NewTestMessagingEnvironment() (*TestMessagingEnvironment, error) {
	waku, err := newTestWakuWrapper()
	if err != nil {
		return nil, err
	}

	return &TestMessagingEnvironment{
		waku:  waku,
		cores: make([]*Core, 0),
	}, nil
}

func (f *TestMessagingEnvironment) Setup() error {
	return f.waku.Waku.Start()
}

func (f *TestMessagingEnvironment) TearDown() error {
	for _, core := range f.cores {
		if err := core.stop(); err != nil {
			return err
		}
	}
	f.cores = make([]*Core, 0)

	return f.waku.Waku.Stop()
}

func (f *TestMessagingEnvironment) NewTestCore(params CoreParams, options ...Options) (*Core, error) {
	core, err := newCore(f.waku, params, newConfig(options...))
	if err != nil {
		return nil, err
	}
	f.cores = append(f.cores, core)
	return core, nil
}

func (f *TestMessagingEnvironment) SubscribePostEvents() chan *PostMessageSubscription {
	return f.waku.SubscribePostEvents()
}

func (f *TestMessagingEnvironment) SimulateOffline() func() {
	f.waku.Waku.(*wakuv2.Waku).SkipPublishToTopic(true)
	return func() {
		f.waku.Waku.(*wakuv2.Waku).SkipPublishToTopic(false)
	}
}

// Wraps waku to provide ability to subscribe to post events.
type testWakuWrapper struct {
	wakutypes.Waku
	api *testPublicWakuAPI
}

func (tw *testWakuWrapper) PublicWakuAPI() wakutypes.PublicWakuAPI {
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
	Msg *wakutypes.NewMessage
}

type testPublicWakuAPI struct {
	*wakuv2.PublicWakuAPI

	postSubscriptions []chan *PostMessageSubscription
}

func (tp *testPublicWakuAPI) Post(ctx context.Context, req wakutypes.NewMessage) ([]byte, error) {
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
	w, err := wakuv2.New(
		nil,
		&wakuv2.DefaultConfig,
		zap.NewNop(),
		nil,
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
			PublicWakuAPI: wakuv2.NewPublicWakuAPI(w),
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
