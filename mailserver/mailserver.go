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
	"errors"
	"fmt"

	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/params"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	maxQueryRange = 24 * time.Hour
)

var (
	errDirectoryNotProvided = errors.New("data directory not provided")
	errPasswordNotProvided  = errors.New("password is not specified")
	// By default go-ethereum/metrics creates dummy metrics that don't register anything.
	// Real metrics are collected only if -metrics flag is set
	requestProcessTimer    = metrics.NewRegisteredTimer("mailserver/processTime", nil)
	requestsCounter        = metrics.NewRegisteredCounter("mailserver/requests", nil)
	requestErrorsCounter   = metrics.NewRegisteredCounter("mailserver/requestErrors", nil)
	sentEnvelopesMeter     = metrics.NewRegisteredMeter("mailserver/envelopes", nil)
	sentEnvelopesSizeMeter = metrics.NewRegisteredMeter("mailserver/envelopesSize", nil)
)

// WMailServer whisper mailserver.
type WMailServer struct {
	db    *leveldb.DB
	w     *whisper.Whisper
	pow   float64
	key   []byte
	limit *limiter
	tick  *ticker
}

// DBKey key to be stored on db.
type DBKey struct {
	timestamp uint32
	hash      common.Hash
	raw       []byte
}

// NewDbKey creates a new DBKey with the given values.
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

// Init initializes mailServer.
func (s *WMailServer) Init(shh *whisper.Whisper, config *params.WhisperConfig) error {
	var err error

	if len(config.DataDir) == 0 {
		return errDirectoryNotProvided
	}

	if len(config.Password) == 0 {
		return errPasswordNotProvided
	}

	s.db, err = leveldb.OpenFile(config.DataDir, nil)
	if err != nil {
		return fmt.Errorf("open DB: %s", err)
	}

	s.w = shh
	s.pow = config.MinimumPoW

	if err := s.setupWhisperIdentity(config); err != nil {
		return err
	}
	s.setupLimiter(time.Duration(config.MailServerRateLimit) * time.Second)

	return nil
}

// setupLimiter in case limit is bigger than 0 it will setup an automated
// limit db cleanup.
func (s *WMailServer) setupLimiter(limit time.Duration) {
	if limit > 0 {
		s.limit = newLimiter(limit)
		s.setupMailServerCleanup(limit)
	}
}

// setupWhisperIdentity setup the whisper identity (symkey) for the current mail
// server.
func (s *WMailServer) setupWhisperIdentity(config *params.WhisperConfig) error {
	MailServerKeyID, err := s.w.AddSymKeyFromPassword(config.Password)
	if err != nil {
		return fmt.Errorf("create symmetric key: %s", err)
	}

	s.key, err = s.w.GetSymKey(MailServerKeyID)
	if err != nil {
		return fmt.Errorf("save symmetric key: %s", err)
	}

	return nil
}

// setupMailServerCleanup periodically runs an expired entries deleteion for
// stored limits.
func (s *WMailServer) setupMailServerCleanup(period time.Duration) {
	if s.tick == nil {
		s.tick = &ticker{}
	}
	go s.tick.run(period, s.limit.deleteExpired)
}

// Close the mailserver and its associated db connection.
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

// Archive a whisper envelope.
func (s *WMailServer) Archive(env *whisper.Envelope) {
	key := NewDbKey(env.Expiry-env.TTL, env.Hash())
	rawEnvelope, err := rlp.EncodeToBytes(env)
	if err != nil {
		log.Error(fmt.Sprintf("rlp.EncodeToBytes failed: %s", err))
	} else {
		if err = s.db.Put(key.raw, rawEnvelope, nil); err != nil {
			log.Error(fmt.Sprintf("Writing to DB failed: %s", err))
		}
	}
}

// DeliverMail sends mail to specified whisper peer.
func (s *WMailServer) DeliverMail(peer *whisper.Peer, request *whisper.Envelope) {
	log.Info("Delivering mail", "peer", peer.ID)
	requestsCounter.Inc(1)

	if peer == nil {
		requestErrorsCounter.Inc(1)
		log.Error("Whisper peer is nil")
		return
	}
	if s.exceedsPeerRequests(peer.ID()) {
		requestErrorsCounter.Inc(1)
		return
	}

	if ok, lower, upper, bloom := s.validateRequest(peer.ID(), request); ok {
		s.processRequest(peer, lower, upper, bloom)
	}
}

// exceedsPeerRequests in case limit its been setup on the current server and limit
// allows the query, it will store/update new query time for the current peer.
func (s *WMailServer) exceedsPeerRequests(peer []byte) bool {
	if s.limit != nil {
		peerID := string(peer)
		if !s.limit.isAllowed(peerID) {
			log.Info("peerID exceeded the number of requests per second")
			return true
		}
		s.limit.add(peerID)
	}
	return false
}

// processRequest processes the current request and re-sends all stored messages
// accomplishing lower and upper limits.
func (s *WMailServer) processRequest(peer *whisper.Peer, lower, upper uint32, bloom []byte) []*whisper.Envelope {
	ret := make([]*whisper.Envelope, 0)
	var err error
	var zero common.Hash
	kl := NewDbKey(lower, zero)
	ku := NewDbKey(upper, zero)
	i := s.db.NewIterator(&util.Range{Start: kl.raw, Limit: ku.raw}, nil)
	defer i.Release()

	var (
		sentEnvelopes     int64
		sentEnvelopesSize int64
	)

	start := time.Now()

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
			sentEnvelopes++
			sentEnvelopesSize += whisper.EnvelopeHeaderLength + int64(len(envelope.Data))
		}
	}

	requestProcessTimer.UpdateSince(start)
	sentEnvelopesMeter.Mark(sentEnvelopes)
	sentEnvelopesSizeMeter.Mark(sentEnvelopesSize)

	err = i.Error()
	if err != nil {
		log.Error(fmt.Sprintf("Level DB iterator error: %s", err))
	}

	return ret
}

// validateRequest runs different validations on the current request.
func (s *WMailServer) validateRequest(peerID []byte, request *whisper.Envelope) (bool, uint32, uint32, []byte) {
	if s.pow > 0.0 && request.PoW() < s.pow {
		return false, 0, 0, nil
	}

	f := whisper.Filter{KeySym: s.key}
	decrypted := request.Open(&f)
	if decrypted == nil {
		log.Warn("Failed to decrypt p2p request")
		return false, 0, 0, nil
	}

	if err := s.checkMsgSignature(decrypted, peerID); err != nil {
		log.Warn(err.Error())
		return false, 0, 0, nil
	}

	bloom, err := s.bloomFromReceivedMessage(decrypted)
	if err != nil {
		log.Warn(err.Error())
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

// checkMsgSignature returns an error in case the message is not correcly signed
func (s *WMailServer) checkMsgSignature(msg *whisper.ReceivedMessage, id []byte) error {
	src := crypto.FromECDSAPub(msg.Src)
	if len(src)-len(id) == 1 {
		src = src[1:]
	}

	// if you want to check the signature, you can do it here. e.g.:
	// if !bytes.Equal(peerID, src) {
	if src == nil {
		return errors.New("Wrong signature of p2p request")
	}

	return nil
}

// bloomFromReceivedMessage gor a given whisper.ReceivedMessage it extracts the
// used bloom filter
func (s *WMailServer) bloomFromReceivedMessage(msg *whisper.ReceivedMessage) ([]byte, error) {
	payloadSize := len(msg.Payload)

	if payloadSize < 8 {
		return nil, errors.New("Undersized p2p request")
	} else if payloadSize == 8 {
		return whisper.MakeFullNodeBloom(), nil
	} else if payloadSize < 8+whisper.BloomFilterSize {
		return nil, errors.New("Undersized bloom filter in p2p request")
	}

	return msg.Payload[8 : 8+whisper.BloomFilterSize], nil
}
