package protocol

import (
	"context"

	"go.uber.org/zap"

	"github.com/status-im/status-go/waku/bridge"
	wakutypes "github.com/status-im/status-go/waku/types"
	"github.com/status-im/status-go/wakuv1"
)

type testWakuWrapper struct {
	*bridge.GethWakuWrapper

	api *testPublicWakuAPI
}

func newTestWaku(w *wakuv1.Waku) wakutypes.Waku {
	wrapper := bridge.NewGethWakuWrapper(w)
	return &testWakuWrapper{
		GethWakuWrapper: wrapper.(*bridge.GethWakuWrapper),
		api:             newTestPublicWakuAPI(wakuv1.NewPublicWakuAPI(w)),
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
	id  []byte
	msg *wakutypes.NewMessage
}

type testPublicWakuAPI struct {
	*wakuv1.PublicWakuAPI

	postSubscriptions []chan *PostMessageSubscription
}

func newTestPublicWakuAPI(api *wakuv1.PublicWakuAPI) *testPublicWakuAPI {
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
		case s <- &PostMessageSubscription{id: id, msg: &req}:
		default:
			// subscription channel full
		}
	}
	return id, err
}

func newTestWakuWrapper(config *wakuv1.Config, logger *zap.Logger) (*testWakuWrapper, error) {
	if config == nil {
		config = &wakuv1.DefaultConfig
	}
	w := wakuv1.New(config, logger)
	return newTestWaku(w).(*testWakuWrapper), w.Start()
}
