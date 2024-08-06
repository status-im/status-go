package publish

import (
	"container/heap"
	"context"

	"github.com/waku-org/go-waku/waku/v2/protocol"
)

// MessagePriority determines the ordering for the message priority queue
type MessagePriority = int

const (
	LowPriority    MessagePriority = 1
	NormalPriority MessagePriority = 2
	HighPriority   MessagePriority = 3
)

type envelopePriority struct {
	envelope *protocol.Envelope
	priority int
	index    int
}

type envelopePriorityQueue []*envelopePriority

func (pq envelopePriorityQueue) Len() int { return len(pq) }

func (pq envelopePriorityQueue) Less(i, j int) bool {
	if pq[i].priority > pq[j].priority {
		return true
	} else if pq[i].priority == pq[j].priority {
		return pq[i].envelope.Message().GetTimestamp() < pq[j].envelope.Message().GetTimestamp()
	}

	return false
}

func (pq envelopePriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *envelopePriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*envelopePriority)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *envelopePriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// MessageQueue is a structure used to handle the ordering of the messages to publish
type MessageQueue struct {
	usePriorityQueue bool

	toSendChan                             chan *protocol.Envelope
	throttledPrioritySendQueue             chan *envelopePriority
	envelopeAvailableOnPriorityQueueSignal chan struct{}
	envelopePriorityQueue                  envelopePriorityQueue
}

// NewMessageQueue returns a new instance of MessageQueue. The MessageQueue can internally use a
// priority queue to handle the ordering of the messages, or use a simple FIFO queue.
func NewMessageQueue(bufferSize int, usePriorityQueue bool) *MessageQueue {
	m := &MessageQueue{
		usePriorityQueue: usePriorityQueue,
	}

	if m.usePriorityQueue {
		m.envelopePriorityQueue = make(envelopePriorityQueue, 0)
		m.throttledPrioritySendQueue = make(chan *envelopePriority, bufferSize)
		m.envelopeAvailableOnPriorityQueueSignal = make(chan struct{}, bufferSize)
		heap.Init(&m.envelopePriorityQueue)
	} else {
		m.toSendChan = make(chan *protocol.Envelope, bufferSize)
	}

	return m
}

// Start must be called to handle the lifetime of the internals of the message queue
func (m *MessageQueue) Start(ctx context.Context) {

	for {
		select {
		case envelopePriority, ok := <-m.throttledPrioritySendQueue:
			if !ok {
				continue
			}

			heap.Push(&m.envelopePriorityQueue, envelopePriority)

			m.envelopeAvailableOnPriorityQueueSignal <- struct{}{}

		case <-ctx.Done():
			return
		}
	}
}

// Push an envelope into the message queue. The priority is optional, and will be ignored
// if the message queue does not use a priority queue
func (m *MessageQueue) Push(ctx context.Context, envelope *protocol.Envelope, priority ...MessagePriority) error {
	if m.usePriorityQueue {
		msgPriority := NormalPriority
		if len(priority) != 0 {
			msgPriority = priority[0]
		}

		pEnvelope := &envelopePriority{
			envelope: envelope,
			priority: msgPriority,
		}

		select {
		case m.throttledPrioritySendQueue <- pEnvelope:
			// Do nothing
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		select {
		case m.toSendChan <- envelope:
			// Do nothing
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// Pop will return a channel on which a message can be retrieved from the message queue
func (m *MessageQueue) Pop(ctx context.Context) <-chan *protocol.Envelope {
	ch := make(chan *protocol.Envelope)

	go func() {
		defer close(ch)

		select {
		case _, ok := <-m.envelopeAvailableOnPriorityQueueSignal:
			if ok {
				ch <- heap.Pop(&m.envelopePriorityQueue).(*envelopePriority).envelope
			}

		case envelope, ok := <-m.toSendChan:
			if ok {
				ch <- envelope
			}

		case <-ctx.Done():
			return
		}

	}()

	return ch
}
