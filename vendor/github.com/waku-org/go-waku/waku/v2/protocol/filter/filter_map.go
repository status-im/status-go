package filter

import (
	"sync"

	"github.com/waku-org/go-waku/waku/v2/protocol"
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

type FilterMap struct {
	sync.RWMutex
	items map[string]Filter
}

type FilterMapItem struct {
	Key   string
	Value Filter
}

func NewFilterMap() *FilterMap {
	return &FilterMap{
		items: make(map[string]Filter),
	}
}

func (fm *FilterMap) Set(key string, value Filter) {
	fm.Lock()
	defer fm.Unlock()

	fm.items[key] = value
}

func (fm *FilterMap) Get(key string) (Filter, bool) {
	fm.Lock()
	defer fm.Unlock()

	value, ok := fm.items[key]

	return value, ok
}

func (fm *FilterMap) Delete(key string) {
	fm.Lock()
	defer fm.Unlock()

	close(fm.items[key].Chan)
	delete(fm.items, key)
}

func (fm *FilterMap) RemoveAll() {
	fm.Lock()
	defer fm.Unlock()

	for k, v := range fm.items {
		close(v.Chan)
		delete(fm.items, k)
	}
}

func (fm *FilterMap) Items() <-chan FilterMapItem {
	c := make(chan FilterMapItem)

	f := func() {
		fm.RLock()
		defer fm.RUnlock()

		for k, v := range fm.items {
			c <- FilterMapItem{k, v}
		}
		close(c)
	}
	go f()

	return c
}

func (fm *FilterMap) Notify(msg *pb.WakuMessage, requestId string) {
	fm.RLock()
	defer fm.RUnlock()

	for key, filter := range fm.items {
		envelope := protocol.NewEnvelope(msg, utils.GetUnixEpoch(), filter.Topic)

		// We do this because the key for the filter is set to the requestId received from the filter protocol.
		// This means we do not need to check the content filter explicitly as all MessagePushs already contain
		// the requestId of the coresponding filter.
		if requestId != "" && requestId == key {
			filter.Chan <- envelope
			continue
		}

		// TODO: In case of no topics we should either trigger here for all messages,
		// or we should not allow such filter to exist in the first place.
		for _, contentTopic := range filter.ContentFilters {
			if msg.ContentTopic == contentTopic {
				filter.Chan <- envelope
				break
			}
		}
	}
}
