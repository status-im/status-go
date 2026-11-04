package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/lightpush"
	"go.uber.org/zap"
)

const routeLightPushV1Messages = "/lightpush/v1/message"

type LightpushService struct {
	node *node.WakuNode
	log  *zap.Logger
}

func NewLightpushService(node *node.WakuNode, m *chi.Mux, log *zap.Logger) *LightpushService {
	serv := &LightpushService{
		node: node,
		log:  log.Named("lightpush"),
	}

	m.Post(routeLightPushV1Messages, serv.postMessagev1)

	return serv
}

func (msg lightpushRequest) Check() error {
	if msg.Message == nil {
		return errors.New("waku message is required")
	}

	return nil
}

type lightpushRequest struct {
	PubSubTopic string           `json:"pubsubTopic"`
	Message     *RestWakuMessage `json:"message"`
}

// handled error codes are 200, 400, 500, 503
func (serv *LightpushService) postMessagev1(w http.ResponseWriter, req *http.Request) {
	request := &lightpushRequest{}
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if err := request.Check(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte(err.Error()))
		serv.log.Error("writing response", zap.Error(err))
		return
	}

	if serv.node.Lightpush() == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	message, err := request.Message.ToProto()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			serv.log.Error("writing response", zap.Error(err))
		}
		return
	}

	if err = message.Validate(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			serv.log.Error("writing response", zap.Error(err))
		}
		return
	}

	_, err = serv.node.Lightpush().Publish(req.Context(), message, lightpush.WithPubSubTopic(request.PubSubTopic))
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err = w.Write([]byte(err.Error()))
		if err != nil {
			serv.log.Error("writing response", zap.Error(err))
		}
	} else {
		writeErrOrResponse(w, err, true)
	}
}
