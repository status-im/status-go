// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package mailserver

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/geth/params"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	maxQueryRange = 24 * time.Hour
)

// WMailServer whisper mailserver
type WMailServer struct {
	db    *leveldb.DB
	w     *whisper.Whisper
	pow   float64
	key   []byte
	limit *limiter
	tick  *ticker
}

// DBKey key to be stored on db
type DBKey struct {
	timestamp uint32
	hash      common.Hash
	raw       []byte
}

// NewDbKey creates a new DBKey with the given values
func NewDbKey(t uint32, h common.Hash) *DBKey {
	const sz = common.HashLength + 4
	var k DBKey
	k.timestamp = t
	k.hash = h
	k.raw = make([]byte, sz)
	binary.BigEndian.PutUint32(k.raw, k.timestamp)
	copy(k.raw[4:], k.hash[:])
	return &k
}

// Init initializes mailServer
func (s *WMailServer) Init(shh *whisper.Whisper, config *params.WhisperConfig) error {
	var err error

	if len(config.DataDir) == 0 {
		return fmt.Errorf("data directory not provided")
	}

	path := filepath.Join(config.DataDir, "mailserver", "data")
	if len(config.Password) == 0 {
		return fmt.Errorf("password is not specified")
	}

	s.db, err = leveldb.OpenFile(path, nil)
	if err != nil {
		return fmt.Errorf("open DB: %s", err)
	}

	s.w = shh
	s.pow = config.MinimumPoW

	MailServerKeyID, err := s.w.AddSymKeyFromPassword(config.Password)
	if err != nil {
		return fmt.Errorf("create symmetric key: %s", err)
	}
	s.key, err = s.w.GetSymKey(MailServerKeyID)
	if err != nil {
		return fmt.Errorf("save symmetric key: %s", err)
	}
	limit := time.Duration(config.MailServerRateLimit) * time.Second
	if limit > 0 {
		s.limit = newLimiter(limit)
		s.setupMailServerCleanup(limit)
	}

	return nil
}

func (s *WMailServer) setupMailServerCleanup(period time.Duration) {
	if period <= 0 {
		return
	}
	if s.tick == nil {
		s.tick = &ticker{}
	}
	go s.tick.run(period, s.limit.deleteExpired)
}

// Close the mailserver and its associated db connection
func (s *WMailServer) Close() {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Error(fmt.Sprintf("s.db.Close failed: %s", err))
		}
	}
	if s.tick != nil {
		s.tick.stop()
	}
}

// Archive a whisper envelope
func (s *WMailServer) Archive(env *whisper.Envelope) {
	key := NewDbKey(env.Expiry-env.TTL, env.Hash())
	rawEnvelope, err := rlp.EncodeToBytes(env)
	if err != nil {
		log.Error(fmt.Sprintf("rlp.EncodeToBytes failed: %s", err))
	} else {
		err = s.db.Put(key.raw, rawEnvelope, nil)
		if err != nil {
			log.Error(fmt.Sprintf("Writing to DB failed: %s", err))
		}
	}
}

// DeliverMail sends mail to specified whisper peer
func (s *WMailServer) DeliverMail(peer *whisper.Peer, request *whisper.Envelope) {
	if peer == nil {
		log.Error("Whisper peer is nil")
		return
	}
	if s.limit != nil {
		peerID := string(peer.ID())
		if !s.limit.isAllowed(peerID) {
			log.Info("peerID exceeded the number of requests per second")
			return
		}
		s.limit.add(peerID)
	}

	ok, lower, upper, bloom := s.validateRequest(peer.ID(), request)
	if ok {
		s.processRequest(peer, lower, upper, bloom)
	}
}

func (s *WMailServer) processRequest(peer *whisper.Peer, lower, upper uint32, bloom []byte) []*whisper.Envelope {
	ret := make([]*whisper.Envelope, 0)
	var err error
	var zero common.Hash
	kl := NewDbKey(lower, zero)
	ku := NewDbKey(upper, zero)
	i := s.db.NewIterator(&util.Range{Start: kl.raw, Limit: ku.raw}, nil)
	defer i.Release()

	for i.Next() {
		var envelope whisper.Envelope
		err = rlp.DecodeBytes(i.Value(), &envelope)
		if err != nil {
			log.Error(fmt.Sprintf("RLP decoding failed: %s", err))
		}

		if whisper.BloomFilterMatch(bloom, envelope.Bloom()) {
			if peer == nil {
				// used for test purposes
				ret = append(ret, &envelope)
			} else {
				err = s.w.SendP2PDirect(peer, &envelope)
				if err != nil {
					log.Error(fmt.Sprintf("Failed to send direct message to peer: %s", err))
					return nil
				}
			}
		}
	}

	err = i.Error()
	if err != nil {
		log.Error(fmt.Sprintf("Level DB iterator error: %s", err))
	}

	return ret
}

func (s *WMailServer) validateRequest(peerID []byte, request *whisper.Envelope) (bool, uint32, uint32, []byte) {
	if s.pow > 0.0 && request.PoW() < s.pow {
		return false, 0, 0, nil
	}

	f := whisper.Filter{KeySym: s.key}
	decrypted := request.Open(&f)
	if decrypted == nil {
		log.Warn(fmt.Sprintf("Failed to decrypt p2p request"))
		return false, 0, 0, nil
	}

	src := crypto.FromECDSAPub(decrypted.Src)
	if len(src)-len(peerID) == 1 {
		src = src[1:]
	}

	// if you want to check the signature, you can do it here. e.g.:
	// if !bytes.Equal(peerID, src) {
	if src == nil {
		log.Warn(fmt.Sprintf("Wrong signature of p2p request"))
		return false, 0, 0, nil
	}

	payloadSize := len(decrypted.Payload)
	bloom := decrypted.Payload[8 : 8+whisper.BloomFilterSize]
	if payloadSize < 8 {
		log.Warn(fmt.Sprintf("Undersized p2p request"))
		return false, 0, 0, nil
	} else if payloadSize == 8 {
		bloom = whisper.MakeFullNodeBloom()
	} else if payloadSize < 8+whisper.BloomFilterSize {
		log.Warn(fmt.Sprintf("Undersized bloom filter in p2p request"))
		return false, 0, 0, nil
	}

	lower := binary.BigEndian.Uint32(decrypted.Payload[:4])
	upper := binary.BigEndian.Uint32(decrypted.Payload[4:8])

	lowerTime := time.Unix(int64(lower), 0)
	upperTime := time.Unix(int64(upper), 0)
	if upperTime.Sub(lowerTime) > maxQueryRange {
		log.Warn(fmt.Sprintf("Query range too big for peer %s", string(peerID)))
		return false, 0, 0, nil
	}

	return true, lower, upper, bloom
}

type limiter struct {
	mu sync.RWMutex

	timeout time.Duration
	db      map[string]time.Time
}

func newLimiter(timeout time.Duration) *limiter {
	return &limiter{
		timeout: timeout,
		db:      make(map[string]time.Time),
	}
}

func (l *limiter) add(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.db[id] = time.Now()
}

func (l *limiter) isAllowed(id string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if lastRequestTime, ok := l.db[id]; ok {
		return lastRequestTime.Add(l.timeout).Before(time.Now())
	}

	return true
}

func (l *limiter) deleteExpired() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for id, lastRequestTime := range l.db {
		if lastRequestTime.Add(l.timeout).Before(now) {
			delete(l.db, id)
		}
	}
}
