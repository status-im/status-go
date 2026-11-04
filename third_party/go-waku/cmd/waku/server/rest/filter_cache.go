package rest

import (
	"fmt"
	"sync"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"go.uber.org/zap"
)

type filterCache struct {
	capacity int
	mu       sync.RWMutex
	log      *zap.Logger
	data     map[string]map[string][]*RestWakuMessage
}

func newFilterCache(capacity int, log *zap.Logger) *filterCache {
	return &filterCache{
		capacity: capacity,
		data:     make(map[string]map[string][]*RestWakuMessage),
		log:      log.Named("cache"),
	}
}

func (c *filterCache) subscribe(contentFilter protocol.ContentFilter) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pubSubTopicMap, _ := protocol.ContentFilterToPubSubTopicMap(contentFilter)
	for pubsubTopic, contentTopics := range pubSubTopicMap {
		if c.data[pubsubTopic] == nil {
			c.data[pubsubTopic] = make(map[string][]*RestWakuMessage)
		}
		for _, topic := range contentTopics {
			if c.data[pubsubTopic][topic] == nil {
				c.data[pubsubTopic][topic] = []*RestWakuMessage{}
			}
		}
	}
}

func (c *filterCache) unsubscribe(pubsubTopic string, contentTopic string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data[pubsubTopic], contentTopic)
}

func (c *filterCache) addMessage(envelope *protocol.Envelope) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pubsubTopic := envelope.PubsubTopic()
	contentTopic := envelope.Message().ContentTopic
	if c.data[pubsubTopic] == nil || c.data[pubsubTopic][contentTopic] == nil {
		return
	}

	// Keep a specific max number of message per topic
	if len(c.data[pubsubTopic][contentTopic]) >= c.capacity {
		c.data[pubsubTopic][contentTopic] = c.data[pubsubTopic][contentTopic][1:]
	}

	message := &RestWakuMessage{}
	if err := message.FromProto(envelope.Message()); err != nil {
		c.log.Error("converting protobuffer msg into rest msg", zap.Error(err))
		return
	}

	c.data[pubsubTopic][contentTopic] = append(c.data[pubsubTopic][contentTopic], message)
}

func (c *filterCache) getMessages(pubsubTopic string, contentTopic string) ([]*RestWakuMessage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.data[pubsubTopic] == nil || c.data[pubsubTopic][contentTopic] == nil {
		return nil, fmt.Errorf("not subscribed to pubsubTopic:%s contentTopic: %s", pubsubTopic, contentTopic)
	}
	msgs := c.data[pubsubTopic][contentTopic]
	c.data[pubsubTopic][contentTopic] = []*RestWakuMessage{}
	return msgs, nil
}
