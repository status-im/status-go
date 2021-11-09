package store

import (
	"sync"
	"time"

	"github.com/status-im/go-waku/waku/v2/utils"
)

type MessageQueue struct {
	sync.RWMutex

	seen        map[[32]byte]struct{}
	messages    []IndexedWakuMessage
	maxMessages int
	maxDuration time.Duration

	quit chan struct{}
}

func (self *MessageQueue) Push(msg IndexedWakuMessage) {
	self.Lock()
	defer self.Unlock()

	var k [32]byte
	copy(k[:], msg.index.Digest)
	if _, ok := self.seen[k]; ok {
		return
	}

	self.seen[k] = struct{}{}
	self.messages = append(self.messages, msg)

	if self.maxMessages != 0 && len(self.messages) > self.maxMessages {
		numToPop := len(self.messages) - self.maxMessages
		self.messages = self.messages[numToPop:len(self.messages)]
	}
}

func (self *MessageQueue) Messages() <-chan IndexedWakuMessage {
	c := make(chan IndexedWakuMessage)

	f := func() {
		self.RLock()
		defer self.RUnlock()
		for _, value := range self.messages {
			c <- value
		}
		close(c)
	}
	go f()

	return c
}

func (self *MessageQueue) cleanOlderRecords() {
	self.Lock()
	defer self.Unlock()

	// TODO: check if retention days was set

	t := utils.GetUnixEpochFrom(time.Now().Add(-self.maxDuration))

	var idx int
	for i := 0; i < len(self.messages); i++ {
		if self.messages[i].index.ReceiverTime >= t {
			idx = i
			break
		}
	}

	self.messages = self.messages[idx:]
}

func (self *MessageQueue) checkForOlderRecords(d time.Duration) {
	ticker := time.NewTicker(d)

	select {
	case <-self.quit:
		return
	case <-ticker.C:
		self.cleanOlderRecords()
	}
}

func (self *MessageQueue) Length() int {
	self.RLock()
	defer self.RUnlock()
	return len(self.messages)
}

func NewMessageQueue(maxMessages int, maxDuration time.Duration) *MessageQueue {
	result := &MessageQueue{
		maxMessages: maxMessages,
		maxDuration: maxDuration,
		seen:        make(map[[32]byte]struct{}),
		quit:        make(chan struct{}),
	}

	if maxDuration != 0 {
		go result.checkForOlderRecords(10 * time.Second) // is 10s okay?
	}

	return result
}

func (self *MessageQueue) Stop() {
	close(self.quit)
}
