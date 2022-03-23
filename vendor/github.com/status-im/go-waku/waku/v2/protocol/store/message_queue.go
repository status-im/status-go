package store

import (
	"errors"
	"sync"
	"time"

	"github.com/status-im/go-waku/waku/v2/utils"
)

// MaxTimeVariance is the maximum duration in the future allowed for a message timestamp
const MaxTimeVariance = time.Duration(20) * time.Second

type MessageQueue struct {
	sync.RWMutex

	seen        map[[32]byte]struct{}
	messages    []IndexedWakuMessage
	maxMessages int
	maxDuration time.Duration

	quit chan struct{}
	wg   *sync.WaitGroup
}

var ErrDuplicatedMessage = errors.New("duplicated message")
var ErrFutureMessage = errors.New("message timestamp in the future")
var ErrTooOld = errors.New("message is too old")

func (self *MessageQueue) Push(msg IndexedWakuMessage) error {
	self.Lock()
	defer self.Unlock()

	var k [32]byte
	copy(k[:], msg.index.Digest)

	if _, ok := self.seen[k]; ok {
		return ErrDuplicatedMessage
	}

	// Ensure that messages don't "jump" to the front of the queue with future timestamps
	if msg.index.SenderTime-msg.index.ReceiverTime > int64(MaxTimeVariance) {
		return ErrFutureMessage
	}

	self.seen[k] = struct{}{}
	self.messages = append(self.messages, msg)

	if self.maxMessages != 0 && len(self.messages) > self.maxMessages {

		if indexComparison(msg.index, self.messages[0].index) < 0 {
			return ErrTooOld // :(
		}

		numToPop := len(self.messages) - self.maxMessages
		self.messages = self.messages[numToPop:len(self.messages)]
	}

	return nil
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
	defer self.wg.Done()

	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-self.quit:
			return
		case <-ticker.C:
			self.cleanOlderRecords()
		}
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
		wg:          &sync.WaitGroup{},
	}

	if maxDuration != 0 {
		result.wg.Add(1)
		go result.checkForOlderRecords(10 * time.Second) // is 10s okay?
	}

	return result
}

func (self *MessageQueue) Stop() {
	close(self.quit)
	self.wg.Wait()
}
