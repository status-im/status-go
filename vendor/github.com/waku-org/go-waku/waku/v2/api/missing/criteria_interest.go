package missing

import (
	"context"
	"slices"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/waku-org/go-waku/waku/v2/protocol"
)

type criteriaInterest struct {
	peerID        peer.ID
	contentFilter protocol.ContentFilter
	lastChecked   time.Time

	ctx    context.Context
	cancel context.CancelFunc
}

func (c criteriaInterest) equals(other criteriaInterest) bool {
	if c.peerID != other.peerID {
		return false
	}

	if c.contentFilter.PubsubTopic != other.contentFilter.PubsubTopic {
		return false
	}

	contentTopics := c.contentFilter.ContentTopics.ToList()
	otherContentTopics := other.contentFilter.ContentTopics.ToList()

	slices.Sort(contentTopics)
	slices.Sort(otherContentTopics)

	if len(contentTopics) != len(otherContentTopics) {
		return false
	}

	for i, contentTopic := range contentTopics {
		if contentTopic != otherContentTopics[i] {
			return false
		}
	}

	return true
}
