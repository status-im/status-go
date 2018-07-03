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
	"sync"

	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/db"
	"github.com/status-im/status-go/params"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

const (
	maxQueryRange = 24 * time.Hour
	noLimits      = 0
)

var (
	errDirectoryNotProvided        = errors.New("data directory not provided")
	errDecryptionMethodNotProvided = errors.New("decryption method is not provided")
	// By default go-ethereum/metrics creates dummy metrics that don't register anything.
	// Real metrics are collected only if -metrics flag is set
	requestProcessTimer    = metrics.NewRegisteredTimer("mailserver/requestProcessTime", nil)
	requestsMeter          = metrics.NewRegisteredMeter("mailserver/requests", nil)
	requestErrorsCounter   = metrics.NewRegisteredCounter("mailserver/requestErrors", nil)
	sentEnvelopesMeter     = metrics.NewRegisteredMeter("mailserver/sentEnvelopes", nil)
	sentEnvelopesSizeMeter = metrics.NewRegisteredMeter("mailserver/sentEnvelopesSize", nil)
	archivedMeter          = metrics.NewRegisteredMeter("mailserver/archivedEnvelopes", nil)
	archivedSizeMeter      = metrics.NewRegisteredMeter("mailserver/archivedEnvelopesSize", nil)
	archivedErrorsCounter  = metrics.NewRegisteredCounter("mailserver/archiveErrors", nil)
)

const (
	timestampLength        = 4
	dbKeyLength            = common.HashLength + timestampLength
	requestLimitLength     = 4
	requestTimeRangeLength = timestampLength * 2
)

type cursorType []byte

// dbImpl is an interface introduced to be able to test some unexpected
// panics from leveldb that are difficult to reproduce.
// normally the db implementation is leveldb.DB, but in TestMailServerDBPanicSuite
// we use panicDB to test panics from the db.
// more info about the panic errors:
// https://github.com/syndtr/goleveldb/issues/224
type dbImpl interface {
	Close() error
	Write(*leveldb.Batch, *opt.WriteOptions) error
	Put([]byte, []byte, *opt.WriteOptions) error
	Get([]byte, *opt.ReadOptions) ([]byte, error)
	NewIterator(*util.Range, *opt.ReadOptions) iterator.Iterator
}

// WMailServer whisper mailserver.
type WMailServer struct {
	db     dbImpl
	w      *whisper.Whisper
	pow    float64
	filter *whisper.Filter

	muLimiter sync.RWMutex
	limiter   *limiter
	tick      *ticker
}

// DBKey key to be stored on db.
type DBKey struct {
	timestamp uint32
	hash      common.Hash
	raw       []byte
}

// NewDbKey creates a new DBKey with the given values.
func NewDbKey(t uint32, h common.Hash) *DBKey {
	var k DBKey
	k.timestamp = t
	k.hash = h
	k.raw = make([]byte, dbKeyLength)
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

	if len(config.MailServerPassword) == 0 && config.MailServerAsymKey == nil {
		return errDecryptionMethodNotProvided
	}

	s.w = shh
	s.pow = config.MinimumPoW

	if err := s.setupRequestMessageDecryptor(config); err != nil {
		return err
	}
	s.setupLimiter(time.Duration(config.MailServerRateLimit) * time.Second)

	// Open database in the last step in order not to init with error
	// and leave the database open by accident.
	database, err := db.Open(config.DataDir, nil)
	if err != nil {
		return fmt.Errorf("open DB: %s", err)
	}
	s.db = database

	return nil
}

// setupLimiter in case limit is bigger than 0 it will setup an automated
// limit db cleanup.
func (s *WMailServer) setupLimiter(limit time.Duration) {
	if limit > 0 {
		s.limiter = newLimiter(limit)
		s.setupMailServerCleanup(limit)
	}
}

// setupRequestMessageDecryptor setup a Whisper filter to decrypt
// incoming Whisper requests.
func (s *WMailServer) setupRequestMessageDecryptor(config *params.WhisperConfig) error {
	var filter whisper.Filter

	if config.MailServerPassword != "" {
		keyID, err := s.w.AddSymKeyFromPassword(config.MailServerPassword)
		if err != nil {
			return fmt.Errorf("create symmetric key: %v", err)
		}

		symKey, err := s.w.GetSymKey(keyID)
		if err != nil {
			return fmt.Errorf("save symmetric key: %v", err)
		}

		filter = whisper.Filter{KeySym: symKey}
	} else if config.MailServerAsymKey != nil {
		filter = whisper.Filter{KeyAsym: config.MailServerAsymKey}
	}

	s.filter = &filter

	return nil
}

// setupMailServerCleanup periodically runs an expired entries deleteion for
// stored limits.
func (s *WMailServer) setupMailServerCleanup(period time.Duration) {
	if s.tick == nil {
		s.tick = &ticker{}
	}
	go s.tick.run(period, s.limiter.deleteExpired)
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

func recoverLevelDBPanics(calleMethodName string) {
	// Recover from possible goleveldb panics
	if r := recover(); r != nil {
		if errString, ok := r.(string); ok {
			log.Error(fmt.Sprintf("recovered from panic in %s: %s", calleMethodName, errString))
		}
	}
}

// Archive a whisper envelope.
func (s *WMailServer) Archive(env *whisper.Envelope) {
	defer recoverLevelDBPanics("Archive")

	key := NewDbKey(env.Expiry-env.TTL, env.Hash())
	rawEnvelope, err := rlp.EncodeToBytes(env)
	if err != nil {
		log.Error(fmt.Sprintf("rlp.EncodeToBytes failed: %s", err))
		archivedErrorsCounter.Inc(1)
	} else {
		if err = s.db.Put(key.raw, rawEnvelope, nil); err != nil {
			log.Error(fmt.Sprintf("Writing to DB failed: %s", err))
			archivedErrorsCounter.Inc(1)
		}
		archivedMeter.Mark(1)
		archivedSizeMeter.Mark(int64(whisper.EnvelopeHeaderLength + len(env.Data)))
	}
}

// DeliverMail sends mail to specified whisper peer.
func (s *WMailServer) DeliverMail(peer *whisper.Peer, request *whisper.Envelope) {
	log.Info("Delivering mail", "peer", peer.ID)
	requestsMeter.Mark(1)

	if peer == nil {
		requestErrorsCounter.Inc(1)
		log.Error("Whisper peer is nil")
		return
	}
	if s.exceedsPeerRequests(peer.ID()) {
		requestErrorsCounter.Inc(1)
		return
	}

	defer recoverLevelDBPanics("DeliverMail")

	if ok, lower, upper, bloom, limit, cursor := s.validateRequest(peer.ID(), request); ok {
		_, lastEnvelopeHash, nextPageCursor, err := s.processRequest(peer, lower, upper, bloom, limit, cursor)
		if err != nil {
			log.Error(fmt.Sprintf("error in DeliverMail: %s", err))
			return
		}

		if err := s.sendHistoricMessageResponse(peer, request, lastEnvelopeHash, nextPageCursor); err != nil {
			log.Error(fmt.Sprintf("SendHistoricMessageResponse error: %s", err))
		}
	}
}

// exceedsPeerRequests in case limit its been setup on the current server and limit
// allows the query, it will store/update new query time for the current peer.
func (s *WMailServer) exceedsPeerRequests(peer []byte) bool {
	s.muLimiter.RLock()
	defer s.muLimiter.RUnlock()

	if s.limiter != nil {
		peerID := string(peer)
		if !s.limiter.isAllowed(peerID) {
			log.Info("peerID exceeded the number of requests per second")
			return true
		}
		s.limiter.add(peerID)
	}
	return false
}

// processRequest processes the current request and re-sends all stored messages
// accomplishing lower and upper limits. The limit parameter determines the maximum number of
// messages to be sent back for the current request.
// The cursor parameter is used for pagination.
// After sending all the messages, a message of type p2pRequestCompleteCode is sent by the mailserver to
// the peer.
func (s *WMailServer) processRequest(peer *whisper.Peer, lower, upper uint32, bloom []byte, limit uint32, cursor cursorType) (ret []*whisper.Envelope, lastEnvelopeHash common.Hash, nextPageCursor cursorType, err error) {
	// Recover from possible goleveldb panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic in processRequest: %v", r)
		}
	}()

	var (
		sentEnvelopes     uint32
		sentEnvelopesSize int64
		zero              common.Hash
		ku                []byte
		kl                []byte
	)

	kl = NewDbKey(lower, zero).raw
	if cursor != nil {
		ku = cursor
	} else {
		ku = NewDbKey(upper+1, zero).raw
	}

	i := s.db.NewIterator(&util.Range{Start: kl, Limit: ku}, nil)
	i.Seek(ku)
	defer i.Release()

	start := time.Now()

	for i.Prev() {
		var envelope whisper.Envelope
		decodeErr := rlp.DecodeBytes(i.Value(), &envelope)
		if decodeErr != nil {
			log.Error(fmt.Sprintf("RLP decoding failed: %s", decodeErr))
			continue
		}

		if whisper.BloomFilterMatch(bloom, envelope.Bloom()) {
			if peer == nil {
				// used for test purposes
				ret = append(ret, &envelope)
			} else {
				err = s.w.SendP2PDirect(peer, &envelope)
				if err != nil {
					log.Error(fmt.Sprintf("Failed to send direct message to peer: %s", err))
					return
				}
				lastEnvelopeHash = envelope.Hash()
			}
			sentEnvelopes++
			sentEnvelopesSize += whisper.EnvelopeHeaderLength + int64(len(envelope.Data))

			if limit != noLimits && sentEnvelopes == limit {
				nextPageCursor = i.Key()
				break
			}
		}
	}

	requestProcessTimer.UpdateSince(start)
	sentEnvelopesMeter.Mark(int64(sentEnvelopes))
	sentEnvelopesSizeMeter.Mark(sentEnvelopesSize)

	err = i.Error()
	if err != nil {
		log.Error(fmt.Sprintf("Level DB iterator error: %s", err))
	}

	return
}

func (s *WMailServer) sendHistoricMessageResponse(peer *whisper.Peer, request *whisper.Envelope, lastEnvelopeHash common.Hash, cursor cursorType) error {
	requestID := request.Hash()
	payload := append(requestID[:], lastEnvelopeHash[:]...)
	payload = append(payload, cursor...)
	return s.w.SendHistoricMessageResponse(peer, payload)
}

// validateRequest runs different validations on the current request.
func (s *WMailServer) validateRequest(peerID []byte, request *whisper.Envelope) (bool, uint32, uint32, []byte, uint32, cursorType) {
	if s.pow > 0.0 && request.PoW() < s.pow {
		return false, 0, 0, nil, 0, nil
	}

	decrypted := request.Open(s.filter)
	if decrypted == nil {
		log.Warn("Failed to decrypt p2p request")
		return false, 0, 0, nil, 0, nil
	}

	if err := s.checkMsgSignature(decrypted, peerID); err != nil {
		log.Warn(err.Error())
		return false, 0, 0, nil, 0, nil
	}

	bloom, err := s.bloomFromReceivedMessage(decrypted)
	if err != nil {
		log.Warn(err.Error())
		return false, 0, 0, nil, 0, nil
	}

	lower := binary.BigEndian.Uint32(decrypted.Payload[:4])
	upper := binary.BigEndian.Uint32(decrypted.Payload[4:8])

	if upper < lower {
		log.Error(fmt.Sprintf("Query range is invalid: from > to (%d > %d)", lower, upper))
		return false, 0, 0, nil, 0, nil
	}

	lowerTime := time.Unix(int64(lower), 0)
	upperTime := time.Unix(int64(upper), 0)
	if upperTime.Sub(lowerTime) > maxQueryRange {
		log.Warn(fmt.Sprintf("Query range too big for peer %s", string(peerID)))
		return false, 0, 0, nil, 0, nil
	}

	var limit uint32
	if len(decrypted.Payload) >= requestTimeRangeLength+whisper.BloomFilterSize+requestLimitLength {
		limit = binary.BigEndian.Uint32(decrypted.Payload[requestTimeRangeLength+whisper.BloomFilterSize:])
	}

	var cursor cursorType
	if len(decrypted.Payload) == requestTimeRangeLength+whisper.BloomFilterSize+requestLimitLength+dbKeyLength {
		cursor = decrypted.Payload[requestTimeRangeLength+whisper.BloomFilterSize+requestLimitLength:]
	}

	return true, lower, upper, bloom, limit, cursor
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
