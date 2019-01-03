package server

import (
	"container/heap"
	"sync"
	"time"
)

type deadline struct {
	time time.Time
}

// definitely rename
// Rewrite cleaner to operate on a leveldb directly
// if it is impossible to query on topic+timestamp(big endian) for purging
// store an additional key
func NewCleaner() *Cleaner {
	return &Cleaner{
		heap:      []string{},
		deadlines: map[string]deadline{},
	}
}

type Cleaner struct {
	mu        sync.RWMutex
	heap      []string
	deadlines map[string]deadline
}

func (c *Cleaner) Id(index int) string {
	return c.heap[index]
}

func (c *Cleaner) Len() int {
	return len(c.heap)
}

func (c *Cleaner) Less(i, j int) bool {
	return c.deadlines[c.Id(i)].time.Before(c.deadlines[c.Id(j)].time)
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
	c.heap = append([]string{}, old[0:n-1]...)
	_, exist := c.deadlines[x]
	if !exist {
		return x
	}
	delete(c.deadlines, x)
	return x
}

func (c *Cleaner) Add(deadlineTime time.Time, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	dl, exist := c.deadlines[key]
	if !exist {
		dl = deadline{time: deadlineTime}
	} else {
		dl.time = deadlineTime
		for i, n := range c.heap {
			if n == key {
				heap.Remove(c, i)
				break
			}
		}
	}
	c.deadlines[key] = dl
	heap.Push(c, key)
}

func (c *Cleaner) Exist(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exist := c.deadlines[key]
	return exist
}

func (c *Cleaner) PopSince(now time.Time) (rst []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for len(c.heap) != 0 {
		dl, exist := c.deadlines[c.heap[0]]
		if !exist {
			continue
		}
		if now.After(dl.time) {
			rst = append(rst, heap.Pop(c).(string))
		} else {
			return rst
		}
	}
	return rst
}
