package peers

import (
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
)

// Register manages register topic queries
type Register struct {
	topics []discv5.Topic

	wg   sync.WaitGroup
	quit chan struct{}
}

// NewRegister creates instance of topic register
func NewRegister(topics ...discv5.Topic) *Register {
	return &Register{topics: topics}
}

// Start topic register query for every topic
func (r *Register) Start(server *p2p.Server) error {
	if server.DiscV5 == nil {
		return ErrDiscv5NotRunning
	}
	r.quit = make(chan struct{})
	for _, topic := range r.topics {
		r.wg.Add(1)
		go func(t discv5.Topic) {
			log.Debug("v5 register topic", "topic", t)
			server.DiscV5.RegisterTopic(t, r.quit)
			r.wg.Done()
		}(topic)
	}
	return nil
}

// Stop all register topic queries and waits for them to exit
func (r *Register) Stop() {
	if r.quit == nil {
		return
	}
	select {
	case <-r.quit:
		return
	default:
		close(r.quit)
	}
	log.Debug("waiting for register queries to exit")
	r.wg.Wait()
}
