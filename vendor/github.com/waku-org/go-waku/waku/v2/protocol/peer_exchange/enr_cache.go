package peer_exchange

import (
	"bufio"
	"bytes"
	"math/rand"
	"sync"

	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange/pb"
)

// simpleLRU internal uses container/list, which is ring buffer(double linked list)
type enrCache struct {
	// using lru, saves us from periodically cleaning the cache to maintain a certain size
	data *simplelru.LRU
	rng  *rand.Rand
	mu   sync.RWMutex
}

// err on negative size
func newEnrCache(size int) (*enrCache, error) {
	inner, err := simplelru.NewLRU(size, nil)
	return &enrCache{
		data: inner,
		rng:  rand.New(rand.NewSource(rand.Int63())),
	}, err
}

// updating cache
func (c *enrCache) updateCache(node *enode.Node) {
	c.mu.Lock()
	defer c.mu.Unlock()
	currNode, ok := c.data.Get(node.ID())
	if !ok || node.Seq() > currNode.(*enode.Node).Seq() {
		c.data.Add(node.ID(), node)
	}
}

// get `numPeers` records of enr
func (c *enrCache) getENRs(neededPeers int) ([]*pb.PeerInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	//
	availablePeers := c.data.Len()
	if availablePeers == 0 {
		return nil, nil
	}
	if availablePeers < neededPeers {
		neededPeers = availablePeers
	}

	perm := c.rng.Perm(availablePeers)[0:neededPeers]
	keys := c.data.Keys()
	result := []*pb.PeerInfo{}
	for _, ind := range perm {
		node, ok := c.data.Get(keys[ind])
		if !ok {
			continue
		}
		var b bytes.Buffer
		writer := bufio.NewWriter(&b)
		err := node.(*enode.Node).Record().EncodeRLP(writer)
		if err != nil {
			return nil, err
		}
		writer.Flush()
		result = append(result, &pb.PeerInfo{
			ENR: b.Bytes(),
		})
	}
	return result, nil
}
