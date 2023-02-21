package timecache

import "time"

type Strategy uint8

const (
	Strategy_FirstSeen Strategy = iota
	Strategy_LastSeen
)

type TimeCache interface {
	Add(string)
	Has(string) bool
}

// NewTimeCache defaults to the original ("first seen") cache implementation
func NewTimeCache(span time.Duration) TimeCache {
	return NewTimeCacheWithStrategy(Strategy_FirstSeen, span)
}

func NewTimeCacheWithStrategy(strategy Strategy, span time.Duration) TimeCache {
	switch strategy {
	case Strategy_FirstSeen:
		return newFirstSeenCache(span)
	case Strategy_LastSeen:
		return newLastSeenCache(span)
	default:
		// Default to the original time cache implementation
		return newFirstSeenCache(span)
	}
}
