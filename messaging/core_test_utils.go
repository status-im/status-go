package messaging

import (
	"context"
	"crypto/ecdsa"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"github.com/status-im/status-go/messaging/types"
	wakutypes "github.com/status-im/status-go/waku/types"
	"github.com/status-im/status-go/wakuv2"
	"github.com/waku-org/waku-go-bindings/waku"
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

func (f *TestMessagingEnvironment) Setup() error {
	return f.waku.Start()
}

func (f *TestMessagingEnvironment) TearDown() error {
	return f.waku.Stop()
}

func (f *TestMessagingEnvironment) NewTestCore(identity *ecdsa.PrivateKey, persistence types.Persistence, options ...Options) (*Core, error) {
	return NewCore(
		f.waku,
		identity,
		persistence,
		options...,
	)
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

func newTestWaku(w *wakuv2.Waku) wakutypes.Waku {
	return &testWakuWrapper{
		Waku: w,
		api:  newTestPublicWakuAPI(wakuv2.NewPublicWakuAPI(w)),
	}
}

func (tw *testWakuWrapper) PublicWakuAPI() wakutypes.PublicWakuAPI {
	return tw.api
}

func (tw *testWakuWrapper) SubscribePostEvents() chan *PostMessageSubscription {
	subscription := make(chan *PostMessageSubscription, 100)
	tw.api.postSubscriptions = append(tw.api.postSubscriptions, subscription)
	return subscription
}

type PostMessageSubscription struct {
	ID  []byte
	Msg *wakutypes.NewMessage
}

type testPublicWakuAPI struct {
	*wakuv2.PublicWakuAPI

	postSubscriptions []chan *PostMessageSubscription
}

func newTestPublicWakuAPI(api *wakuv2.PublicWakuAPI) *testPublicWakuAPI {
	return &testPublicWakuAPI{
		PublicWakuAPI: api,
	}
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

func newTestWakuWrapper() (*testWakuWrapper, error) {

	config := wakuv2.DefaultConfig

	tcpPort, udpPort, err := waku.GetFreePortIfNeeded(0, 0)

	if err != nil {
		return nil, err
	}

	config.Port = tcpPort
	config.UDPPort = udpPort

	w, err := wakuv2.New(
		nil,
		&config,
		zap.NewNop(),
		nil,
		nil,
		func([]byte, peer.AddrInfo, error) {},
		nil,
	)
	if err != nil {
		return nil, err
	}
	return newTestWaku(w).(*testWakuWrapper), w.Start()
}
