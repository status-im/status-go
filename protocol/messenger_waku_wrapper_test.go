package protocol

import (
	"context"

	"go.uber.org/zap"

	wakutypes "github.com/status-im/status-go/waku/types"
	"github.com/status-im/status-go/wakuv2"
)

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
	id  []byte
	msg *wakutypes.NewMessage
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
		case s <- &PostMessageSubscription{id: id, msg: &req}:
		default:
			// subscription channel full
		}
	}
	return id, err
}

func newTestWakuWrapper(logger *zap.Logger) (*testWakuWrapper, error) {
	w, err := newTestWakuNode(logger)
	if err != nil {
		return nil, err
	}
	return newTestWaku(w.(*wakuv2.Waku)).(*testWakuWrapper), w.Start()
}
