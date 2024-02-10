package protocol

import (
	"context"

	"go.uber.org/zap"

	gethbridge "github.com/status-im/status-go/eth-node/bridge/geth"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/waku"
)

type testWakuWrapper struct {
	*gethbridge.GethWakuWrapper

	publicWakuAPIWrapper *testPublicWakuAPIWrapper
}

func newTestWaku(w *waku.Waku) types.Waku {
	wrapper := gethbridge.NewGethWakuWrapper(w)
	return &testWakuWrapper{
		GethWakuWrapper:      wrapper.(*gethbridge.GethWakuWrapper),
		publicWakuAPIWrapper: newTestPublicWakuAPI(waku.NewPublicWakuAPI(w)).(*testPublicWakuAPIWrapper),
	}
}

func (tw *testWakuWrapper) PublicWakuAPI() types.PublicWakuAPI {
	return tw.publicWakuAPIWrapper
}

func (tw *testWakuWrapper) SubscribePostEvents() chan *PostMessageSubscription {
	subscription := make(chan *PostMessageSubscription, 100)
	tw.publicWakuAPIWrapper.postSubscriptions = append(tw.publicWakuAPIWrapper.postSubscriptions, subscription)
	return subscription
}

type PostMessageSubscription struct {
	id  []byte
	msg *types.NewMessage
}

type testPublicWakuAPIWrapper struct {
	*gethbridge.GethPublicWakuAPIWrapper

	postSubscriptions []chan *PostMessageSubscription
}

func newTestPublicWakuAPI(api *waku.PublicWakuAPI) types.PublicWakuAPI {
	wrapper := gethbridge.NewGethPublicWakuAPIWrapper(api)
	return &testPublicWakuAPIWrapper{
		GethPublicWakuAPIWrapper: wrapper.(*gethbridge.GethPublicWakuAPIWrapper),
	}
}

func (tp *testPublicWakuAPIWrapper) Post(ctx context.Context, req types.NewMessage) ([]byte, error) {
	id, err := tp.GethPublicWakuAPIWrapper.Post(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, s := range tp.postSubscriptions {
		select {
		case s <- &PostMessageSubscription{id: id, msg: &req}:
		default:
			// subscription channel full
		}
	}
	return id, err
}

func newTestWakuWrapper(config *waku.Config, logger *zap.Logger) (*testWakuWrapper, error) {
	if config == nil {
		config = &waku.DefaultConfig
	}
	w := waku.New(config, logger)
	return newTestWaku(w).(*testWakuWrapper), w.Start()
}
