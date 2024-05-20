package rpcstats

import (
	"sync"

	"github.com/ethereum/go-ethereum/log"
)

type RPCUsageStats struct {
	total                  uint
	counterPerMethod       sync.Map
	counterPerMethodPerTag sync.Map
}

var stats *RPCUsageStats

func getInstance() *RPCUsageStats {
	if stats == nil {
		stats = &RPCUsageStats{}
	}
	return stats
}

func getStats() (uint, sync.Map) {
	stats := getInstance()
	return stats.total, stats.counterPerMethod
}

// func getStatsWithTag(tag string) (sync.Map, bool) {
// 	stats := getInstance()
// 	value, ok := stats.counterPerMethodPerTag.Load(tag)
// 	return value.(sync.Map), ok
// }

func resetStats() {
	stats := getInstance()
	stats.total = 0
	stats.counterPerMethod = sync.Map{}
	stats.counterPerMethodPerTag = sync.Map{}
}

// func resetStatsWithTag(tag string) {
// 	stats := getInstance()
// 	stats.counterPerMethodPerTag.Delete(tag)
// }

func CountCall(method string) {
	log.Info("CountCall", "method", method)

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
	value, _ := stats.counterPerMethodPerTag.LoadOrStore(tag, sync.Map{})
	methodMap := value.(sync.Map)
	value, _ = methodMap.LoadOrStore(method, uint(0))
	methodMap.Store(method, value.(uint)+1)

	log.Info("CountCallWithTag", "method", method, "tag", tag, "count", value.(uint)+1)

	CountCall(method)
}
