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

func getInstance() *RPCUsageStats {
	mu.Lock()
	defer mu.Unlock()

	if stats == nil {
		stats = &RPCUsageStats{}
		stats.counterPerMethod = &sync.Map{}
		stats.counterPerMethodPerTag = &sync.Map{}
	}
	return stats
}

func getStats() (uint, *sync.Map, *sync.Map) {
	stats := getInstance()
	return stats.total, stats.counterPerMethod, stats.counterPerMethodPerTag
}

func resetStats() {
	stats := getInstance()
	stats.total = 0
	stats.counterPerMethod = &sync.Map{}
	stats.counterPerMethodPerTag = &sync.Map{}
}

func CountCall(method string) {
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

	stats := getInstance()
	value, _ := stats.counterPerMethodPerTag.LoadOrStore(tag, &sync.Map{})
	methodMap := value.(*sync.Map)
	value, _ = methodMap.LoadOrStore(method, uint(0))
	methodMap.Store(method, value.(uint)+1)
	stats.total++
}
