package rest

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/waku-org/go-waku/cmd/waku/server"
	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
	"go.uber.org/zap"
)

const routeRelayV1Subscriptions = "/relay/v1/subscriptions"
const routeRelayV1Messages = "/relay/v1/messages/{topic}"

const routeRelayV1AutoSubscriptions = "/relay/v1/auto/subscriptions"
const routeRelayV1AutoMessages = "/relay/v1/auto/messages"

// RelayService represents the REST service for WakuRelay
type RelayService struct {
	node *node.WakuNode

	log *zap.Logger

	cacheCapacity uint
}

// NewRelayService returns an instance of RelayService
func NewRelayService(node *node.WakuNode, m *chi.Mux, cacheCapacity uint, log *zap.Logger) *RelayService {
	s := &RelayService{
		node:          node,
		log:           log.Named("relay"),
		cacheCapacity: cacheCapacity,
	}

	m.Post(routeRelayV1Subscriptions, s.postV1Subscriptions)
	m.Delete(routeRelayV1Subscriptions, s.deleteV1Subscriptions)
	m.Get(routeRelayV1Messages, s.getV1Messages)
	m.Post(routeRelayV1Messages, s.postV1Message)

	m.Post(routeRelayV1AutoSubscriptions, s.postV1AutoSubscriptions)
	m.Delete(routeRelayV1AutoSubscriptions, s.deleteV1AutoSubscriptions)

	m.Route(routeRelayV1AutoMessages, func(r chi.Router) {
		r.Get("/{contentTopic}", s.getV1AutoMessages)
		r.Post("/", s.postV1AutoMessage)
	})

	return s
}

func (r *RelayService) deleteV1Subscriptions(w http.ResponseWriter, req *http.Request) {
	var topics []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&topics); err != nil {
		r.log.Error("decoding request failure", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var err error
	for _, topic := range topics {
		err = r.node.Relay().Unsubscribe(req.Context(), protocol.NewContentFilter(topic))
		if err != nil {
			r.log.Error("unsubscribing from topic", zap.String("topic", strings.Replace(strings.Replace(topic, "\n", "", -1), "\r", "", -1)), zap.Error(err))
		}
	}

	writeErrOrResponse(w, err, true)
}

func (r *RelayService) postV1Subscriptions(w http.ResponseWriter, req *http.Request) {
	var topics []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&topics); err != nil {
		r.log.Error("decoding request failure", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var err error
	var successCnt int
	var topicToSubscribe string
	for _, topic := range topics {
		if topic == "" {
			topicToSubscribe = relay.DefaultWakuTopic
		} else {
			topicToSubscribe = topic
		}
		_, err = r.node.Relay().Subscribe(r.node.Relay().Context(), protocol.NewContentFilter(topicToSubscribe), relay.WithCacheSize(r.cacheCapacity))

		if err != nil {
			r.log.Error("subscribing to topic", zap.String("topic", strings.Replace(topicToSubscribe, "\n", "", -1)), zap.Error(err))
			continue
		}
		successCnt++
	}

	// on partial subscribe failure
	if successCnt > 0 && err != nil {
		r.log.Error("partial subscribe failed", zap.Error(err))
		// on partial failure
		writeResponse(w, err, http.StatusOK)
		return
	}

	writeErrOrResponse(w, err, true)
}

func (r *RelayService) getV1Messages(w http.ResponseWriter, req *http.Request) {
	topic := topicFromPath(w, req, "topic", r.log)
	if topic == "" {
		r.log.Debug("topic is not specified, using default waku topic")
		topic = relay.DefaultWakuTopic
	}
	//TODO: Update the API to also take a contentTopic since relay now supports filtering based on contentTopic as well.
	sub, err := r.node.Relay().GetSubscriptionWithPubsubTopic(topic, "")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte(err.Error()))
		r.log.Error("writing response", zap.Error(err))
		return
	}
	var response []*RestWakuMessage
	done := false
	for {
		if done || len(response) > int(r.cacheCapacity) {
			break
		}
		select {
		case envelope, open := <-sub.Ch:
			if !open {
				r.log.Error("consume channel is closed for subscription", zap.String("pubsubTopic", topic))
				w.WriteHeader(http.StatusNotFound)
				_, err = w.Write([]byte("consume channel is closed for subscription"))
				if err != nil {
					r.log.Error("writing response", zap.Error(err))
				}
				return
			}

			message := &RestWakuMessage{}
			if err := message.FromProto(envelope.Message()); err != nil {
				r.log.Error("converting protobuffer msg into rest msg", zap.Error(err))
			} else {
				response = append(response, message)
			}
		case <-req.Context().Done():
			done = true
		default:
			done = true
		}
	}

	writeErrOrResponse(w, nil, response)
}

func (r *RelayService) postV1Message(w http.ResponseWriter, req *http.Request) {
	topic := topicFromPath(w, req, "topic", r.log)
	if topic == "" {
		r.log.Debug("topic is not specified, using default waku topic")
		topic = relay.DefaultWakuTopic
	}

	var restMessage *RestWakuMessage
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&restMessage); err != nil {
		r.log.Error("decoding request failure", zap.Error(err))
		writeErrResponse(w, r.log, err, http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	message, err := restMessage.ToProto()
	if err != nil {
		r.log.Error("failed to convert message to proto", zap.Error(err))
		writeErrResponse(w, r.log, err, http.StatusBadRequest)
		return
	}

	if err := server.AppendRLNProof(r.node, message); err != nil {
		r.log.Error("failed to append RLN proof for the message", zap.Error(err))
		writeErrOrResponse(w, err, nil)
		return
	}

	_, err = r.node.Relay().Publish(req.Context(), message, relay.WithPubSubTopic(strings.Replace(topic, "\n", "", -1)))
	if err != nil {
		r.log.Error("publishing message", zap.Error(err))
		if err == pb.ErrMissingPayload || err == pb.ErrMissingContentTopic || err == pb.ErrInvalidMetaLength {
			writeErrResponse(w, r.log, err, http.StatusBadRequest)
			return
		}
	}

	writeErrOrResponse(w, err, true)
}

func (r *RelayService) deleteV1AutoSubscriptions(w http.ResponseWriter, req *http.Request) {
	var cTopics []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&cTopics); err != nil {
		r.log.Error("decoding request failure", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	err := r.node.Relay().Unsubscribe(req.Context(), protocol.NewContentFilter("", cTopics...))
	if err != nil {
		r.log.Error("unsubscribing from topics", zap.Strings("contentTopics", cTopics), zap.Error(err))
	}

	writeErrOrResponse(w, err, true)
}

func (r *RelayService) postV1AutoSubscriptions(w http.ResponseWriter, req *http.Request) {
	var cTopics []string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&cTopics); err != nil {
		r.log.Error("decoding request failure", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var err error
	_, err = r.node.Relay().Subscribe(r.node.Relay().Context(), protocol.NewContentFilter("", cTopics...), relay.WithCacheSize(r.cacheCapacity))
	if err != nil {
		r.log.Error("subscribing to topics", zap.Strings("contentTopics", cTopics), zap.Error(err))
	}
	r.log.Debug("subscribed to topics", zap.Strings("contentTopics", cTopics))

	if err != nil {
		r.log.Error("writing response", zap.Error(err))
		writeErrResponse(w, r.log, err, http.StatusBadRequest)
	} else {
		writeErrOrResponse(w, err, true)
	}

}

func (r *RelayService) getV1AutoMessages(w http.ResponseWriter, req *http.Request) {

	cTopic := topicFromPath(w, req, "contentTopic", r.log)
	sub, err := r.node.Relay().GetSubscription(cTopic)
	if err != nil {
		r.log.Error("writing response", zap.Error(err))
		writeErrResponse(w, r.log, err, http.StatusNotFound)
		return
	}
	var response []*RestWakuMessage
	done := false
	for {
		if done || len(response) > int(r.cacheCapacity) {
			break
		}
		select {
		case envelope := <-sub.Ch:
			message := &RestWakuMessage{}
			if err := message.FromProto(envelope.Message()); err != nil {
				r.log.Error("converting protobuffer msg into rest msg", zap.Error(err))
			} else {
				response = append(response, message)
			}
		case <-req.Context().Done():
			done = true
		default:
			done = true
		}
	}

	writeErrOrResponse(w, nil, response)
}

func (r *RelayService) postV1AutoMessage(w http.ResponseWriter, req *http.Request) {

	var restMessage *RestWakuMessage
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&restMessage); err != nil {
		r.log.Error("decoding request failure", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	message, err := restMessage.ToProto()
	if err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}

	if err = server.AppendRLNProof(r.node, message); err != nil {
		writeErrOrResponse(w, err, nil)
		return
	}

	_, err = r.node.Relay().Publish(req.Context(), message)
	if err != nil {
		r.log.Error("publishing message", zap.Error(err))
		if err == pb.ErrMissingPayload || err == pb.ErrMissingContentTopic || err == pb.ErrInvalidMetaLength {
			writeErrResponse(w, r.log, err, http.StatusBadRequest)
			return
		}
		writeErrResponse(w, r.log, err, http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusOK)
	}

}
