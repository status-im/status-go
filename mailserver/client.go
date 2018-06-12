package mailserver

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

const acksQueueLimit = 1024

type Watcher struct {
	SymKey    []byte
	PeerID    []byte
	RequestID common.Hash
	Done      chan struct{}
}

type Client struct {
	sync.Mutex
	shh      *whisper.Whisper
	acks     chan *whisper.Envelope
	watchers map[common.Hash]*Watcher
}

func NewClient(shh *whisper.Whisper) *Client {
	c := &Client{
		shh:      shh,
		acks:     make(chan *whisper.Envelope, acksQueueLimit),
		watchers: make(map[common.Hash]*Watcher),
	}

	shh.AddAckWatcher(c.acks)
	go c.watch()

	return c
}

func (c *Client) RequestHistoricMessages(peerID []byte, envelope *whisper.Envelope, symKey []byte) (*Watcher, error) {
	w := c.getOrCreateWatcher(peerID, envelope, symKey)
	if err := c.shh.RequestHistoricMessages(peerID, envelope); err != nil {
		c.RemoveWatcher(w)
		return nil, err
	}

	return w, nil
}

func (c *Client) getOrCreateWatcher(peerID []byte, envelope *whisper.Envelope, symKey []byte) *Watcher {
	c.Lock()
	defer c.Unlock()

	requestID := envelope.Hash()
	if w, ok := c.watchers[requestID]; ok {
		return w
	}

	w := &Watcher{
		SymKey:    symKey,
		PeerID:    peerID,
		RequestID: requestID,
		Done:      make(chan struct{}),
	}

	c.watchers[requestID] = w

	return w
}

func (c *Client) RemoveWatcher(w *Watcher) {
	delete(c.watchers, w.RequestID)
}

func (c *Client) watch() {
	for {
		select {
		case env := <-c.acks:
			for _, watcher := range c.watchers {
				// TODO: check PoW?
				f := whisper.Filter{KeySym: watcher.SymKey}
				decrypted := env.Open(&f)
				if decrypted == nil {
					continue
				}

				// TODO:@pilu check signature?
				// if err := c.checkMsgSignature(decrypted, watcher.peerID); err != nil {

				// TODO:@pilu check size of payload before converting it to common.Hash?
				payload := common.BytesToHash(decrypted.Payload)
				if watcher.RequestID == payload {
					close(watcher.Done)
					c.RemoveWatcher(watcher)
				}
			}
		}
	}
}
