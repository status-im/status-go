package server

import (
	"container/heap"
	"sync"
	"time"
)

// definitely rename
// Rewrite cleaner to operate on a leveldb directly
// if it is impossible to query on topic+timestamp(big endian) for purging
// store an additional key
func NewCleaner() *Cleaner {
	return &Cleaner{
		heap:      []string{},
		deadlines: map[string]time.Time{},
	}
}

type Cleaner struct {
	mu        sync.RWMutex
	heap      []string
	deadlines map[string]time.Time
}

func (c *Cleaner) Id(index int) string {
	return c.heap[index]
}

func (c *Cleaner) Len() int {
	return len(c.heap)
}

func (c *Cleaner) Less(i, j int) bool {
	return c.deadlines[c.Id(i)].Before(c.deadlines[c.Id(j)])
}

func (c *Cleaner) Swap(i, j int) {
	c.heap[i], c.heap[j] = c.heap[j], c.heap[i]
}

func (c *Cleaner) Push(record interface{}) {
	c.heap = append(c.heap, record.(string))
}

func (c *Cleaner) Pop() interface{} {
	old := c.heap
	n := len(old)
	x := old[n-1]
	c.heap = old[0 : n-1]
	delete(c.deadlines, x)
	return x
}

func (c *Cleaner) Add(deadline time.Time, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deadlines[key] = deadline
	heap.Push(c, key)
}

func (c *Cleaner) PopOneSince(now time.Time) (rst string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.heap) == 0 {
		return
	}
	if now.After(c.deadlines[c.heap[0]]) {
		return heap.Pop(c).(string)
	}
	return
}
