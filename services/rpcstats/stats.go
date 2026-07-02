package rpcstats

import (
	"sync"
)

type RPCUsageStats struct {
	total                  uint
	counterPerMethod       *sync.Map
	counterPerMethodPerTag *sync.Map
}

var stats *RPCUsageStats
var mu sync.Mutex
var statsOnce sync.Once

func getInstance() *RPCUsageStats {
	statsOnce.Do(func() {
		stats = &RPCUsageStats{
			counterPerMethod:       &sync.Map{},
			counterPerMethodPerTag: &sync.Map{},
		}
	})
	return stats
}

func getStats() (uint, *sync.Map, *sync.Map) {
	mu.Lock()
	defer mu.Unlock()

	s := getInstance()
	return s.total, s.counterPerMethod, s.counterPerMethodPerTag
}

func resetStats() {
	mu.Lock()
	defer mu.Unlock()

	s := getInstance()
	s.total = 0
	s.counterPerMethod = &sync.Map{}
	s.counterPerMethodPerTag = &sync.Map{}
}

func CountCall(method string) {
	mu.Lock()
	defer mu.Unlock()

	stats := getInstance()
	stats.total++
	value, _ := stats.counterPerMethod.LoadOrStore(method, uint(0))
	stats.counterPerMethod.Store(method, value.(uint)+1)
}

func CountCallWithTag(method string, tag string) {
	if tag == "" {
		CountCall(method)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	stats := getInstance()
	value, _ := stats.counterPerMethodPerTag.LoadOrStore(tag, &sync.Map{})
	methodMap := value.(*sync.Map)
	value, _ = methodMap.LoadOrStore(method, uint(0))
	methodMap.Store(method, value.(uint)+1)
	stats.total++
}
