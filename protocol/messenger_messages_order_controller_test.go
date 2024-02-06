package protocol

import (
	"sort"
	"sync"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/transport"
)

type messagesOrderType int

const (
	messagesOrderRandom messagesOrderType = iota
	messagesOrderAsPosted
	messagesOrderReversed
)

type MessagesOrderController struct {
	order               messagesOrderType
	messagesInPostOrder [][]byte
	mutex               sync.RWMutex
	quit                chan struct{}
	quitOnce            sync.Once
}

func NewMessagesOrderController(order messagesOrderType) *MessagesOrderController {
	return &MessagesOrderController{
		order:               order,
		messagesInPostOrder: [][]byte{},
		mutex:               sync.RWMutex{},
		quit:                make(chan struct{}),
	}
}

func (m *MessagesOrderController) Start(c chan *PostMessageSubscription) {
	go func() {
		for {
			select {
			case sub, more := <-c:
				if !more {
					return
				}
				m.mutex.Lock()
				m.messagesInPostOrder = append(m.messagesInPostOrder, sub.id)
				m.mutex.Unlock()

			case <-m.quit:
				return
			}
		}
	}()
}

func (m *MessagesOrderController) Stop() {
	m.quitOnce.Do(func() {
		close(m.quit)
	})
}

func (m *MessagesOrderController) newMessagesIterator(chatWithMessages map[transport.Filter][]*types.Message) MessagesIterator {
	switch m.order {
	case messagesOrderAsPosted, messagesOrderReversed:
		return &messagesIterator{chatWithMessages: m.sort(chatWithMessages, m.order)}
	}

	return NewDefaultMessagesIterator(chatWithMessages)
}

func buildIndexMap(messages [][]byte) map[string]int {
	indexMap := make(map[string]int)
	for i, hash := range messages {
		hashStr := string(hash)
		indexMap[hashStr] = i
	}
	return indexMap
}

func (m *MessagesOrderController) sort(chatWithMessages map[transport.Filter][]*types.Message, order messagesOrderType) []*chatWithMessage {
	allMessages := make([]*chatWithMessage, 0)
	for chat, messages := range chatWithMessages {
		for _, message := range messages {
			allMessages = append(allMessages, &chatWithMessage{chat: chat, message: message})
		}
	}

	m.mutex.RLock()
	indexMap := buildIndexMap(m.messagesInPostOrder)
	m.mutex.RUnlock()

	sort.SliceStable(allMessages, func(i, j int) bool {
		indexI, okI := indexMap[string(allMessages[i].message.Hash)]
		indexJ, okJ := indexMap[string(allMessages[j].message.Hash)]

		if okI && okJ {
			if order == messagesOrderReversed {
				return indexI > indexJ
			}
			return indexI < indexJ
		}

		return !okI && okJ // keep messages with unknown hashes at the end
	})

	return allMessages
}

type chatWithMessage struct {
	chat    transport.Filter
	message *types.Message
}

type messagesIterator struct {
	chatWithMessages []*chatWithMessage
	currentIndex     int
}

func (it *messagesIterator) HasNext() bool {
	return it.currentIndex < len(it.chatWithMessages)
}

func (it *messagesIterator) Next() (transport.Filter, []*types.Message) {
	if it.HasNext() {
		m := it.chatWithMessages[it.currentIndex]
		it.currentIndex++
		return m.chat, []*types.Message{m.message}
	}

	return transport.Filter{}, nil
}
