package history

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type SortedStorenode struct {
	Storenode       peer.ID
	RTT             time.Duration
	CanConnectAfter time.Time
}

type byRTTMsAndCanConnectBefore []SortedStorenode

func (s byRTTMsAndCanConnectBefore) Len() int {
	return len(s)
}

func (s byRTTMsAndCanConnectBefore) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byRTTMsAndCanConnectBefore) Less(i, j int) bool {
	// Slightly inaccurate as time sensitive sorting, but it does not matter so much
	now := time.Now()
	if s[i].CanConnectAfter.Before(now) && s[j].CanConnectAfter.Before(now) {
		return s[i].RTT < s[j].RTT
	}
	return s[i].CanConnectAfter.Before(s[j].CanConnectAfter)
}
