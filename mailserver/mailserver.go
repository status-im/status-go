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
	"github.com/status-im/status-go/db"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/shhext"
	whisper "github.com/status-im/whisper/whisperv6"
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
	requestProcessTimer            = metrics.NewRegisteredTimer("mailserver/requestProcessTime", nil)
	requestProcessNetTimer         = metrics.NewRegisteredTimer("mailserver/requestProcessNetTime", nil)
	requestsMeter                  = metrics.NewRegisteredMeter("mailserver/requests", nil)
	requestsBatchedCounter         = metrics.NewRegisteredCounter("mailserver/requestsBatched", nil)
	requestErrorsCounter           = metrics.NewRegisteredCounter("mailserver/requestErrors", nil)
	sentEnvelopesMeter             = metrics.NewRegisteredMeter("mailserver/sentEnvelopes", nil)
	sentEnvelopesSizeMeter         = metrics.NewRegisteredMeter("mailserver/sentEnvelopesSize", nil)
	archivedMeter                  = metrics.NewRegisteredMeter("mailserver/archivedEnvelopes", nil)
	archivedSizeMeter              = metrics.NewRegisteredMeter("mailserver/archivedEnvelopesSize", nil)
	archivedErrorsCounter          = metrics.NewRegisteredCounter("mailserver/archiveErrors", nil)
	requestValidationErrorsCounter = metrics.NewRegisteredCounter("mailserver/requestValidationErrors", nil)
	processRequestErrorsCounter    = metrics.NewRegisteredCounter("mailserver/processRequestErrors", nil)
	historicResponseErrorsCounter  = metrics.NewRegisteredCounter("mailserver/historicResponseErrors", nil)
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
	db         dbImpl
	w          *whisper.Whisper
	pow        float64
	symFilter  *whisper.Filter
	asymFilter *whisper.Filter

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

	if len(config.MailServerPassword) == 0 && len(config.MailServerAsymKey) == 0 {
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
	s.symFilter = nil
	s.asymFilter = nil

	if config.MailServerPassword != "" {
		keyID, err := s.w.AddSymKeyFromPassword(config.MailServerPassword)
		if err != nil {
			return fmt.Errorf("create symmetric key: %v", err)
		}

		symKey, err := s.w.GetSymKey(keyID)
		if err != nil {
			return fmt.Errorf("save symmetric key: %v", err)
		}

		s.symFilter = &whisper.Filter{KeySym: symKey}
	}

	if config.MailServerAsymKey != "" {
		keyAsym, err := crypto.HexToECDSA(config.MailServerAsymKey)
		if err != nil {
			return err
		}
		s.asymFilter = &whisper.Filter{KeyAsym: keyAsym}
	}

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

	log.Debug("Archiving envelope", "hash", env.Hash().Hex())

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
	log.Info("Delivering mail", "peerID", peerIDString(peer))
	requestsMeter.Mark(1)

	if peer == nil {
		requestErrorsCounter.Inc(1)
		log.Error("Whisper peer is nil")
		return
	}
	if s.exceedsPeerRequests(peer.ID()) {
		requestErrorsCounter.Inc(1)
		log.Error("Peer exceeded request per seconds limit", "peerID", peerIDString(peer))
		s.trySendHistoricMessageErrorResponse(peer, request, fmt.Errorf("rate limit exceeded"))
		return
	}

	defer recoverLevelDBPanics("DeliverMail")

	var (
		lower, upper uint32
		bloom        []byte
		limit        uint32
		cursor       cursorType
		batch        bool
		err          error
	)

	payload, err := s.decodeRequest(peer.ID(), request)
	if err == nil {
		lower, upper = payload.Lower, payload.Upper
		bloom = payload.Bloom
		cursor = cursorType(payload.Cursor)
		limit = payload.Limit
		batch = payload.Batch
	} else {
		log.Debug("Failed to decode request", "err", err, "peerID", peerIDString(peer))
		lower, upper, bloom, limit, cursor, err = s.validateRequest(peer.ID(), request)
	}

	if err != nil {
		requestValidationErrorsCounter.Inc(1)
		log.Error("Mailserver request failed validaton", "peerID", peerIDString(peer))
		s.trySendHistoricMessageErrorResponse(peer, request, err)
		return
	}

	log.Debug("Processing request",
		"lower", lower, "upper", upper,
		"bloom", bloom,
		"limit", limit,
		"cursor", cursor,
		"batch", batch)

	if batch {
		requestsBatchedCounter.Inc(1)
	}

	_, lastEnvelopeHash, nextPageCursor, err := s.processRequest(
		peer,
		lower, upper,
		bloom,
		limit,
		cursor,
		batch)
	if err != nil {
		processRequestErrorsCounter.Inc(1)
		log.Error("Error while processing mail server request", "err", err, "peerID", peerIDString(peer))
		s.trySendHistoricMessageErrorResponse(peer, request, err)
		return
	}

	log.Debug("Sending historic message response", "last", lastEnvelopeHash, "next", nextPageCursor)

	if err := s.sendHistoricMessageResponse(peer, request, lastEnvelopeHash, nextPageCursor); err != nil {
		historicResponseErrorsCounter.Inc(1)
		log.Error("Error sending historic message response", "err", err, "peerID", peerIDString(peer))
		// we still want to try to report error even it it is a p2p error and it is unlikely
		s.trySendHistoricMessageErrorResponse(peer, request, err)
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

func (s *WMailServer) createIterator(lower, upper uint32, cursor cursorType) iterator.Iterator {
	var (
		emptyHash common.Hash
		ku        []byte
		kl        []byte
	)

	kl = NewDbKey(lower, emptyHash).raw
	if len(cursor) == dbKeyLength {
		ku = cursor
	} else {
		ku = NewDbKey(upper+1, emptyHash).raw
	}

	i := s.db.NewIterator(&util.Range{Start: kl, Limit: ku}, nil)
	// seek to the end as we want to return envelopes in a descending order
	i.Seek(ku)

	return i
}

// processRequest processes the current request and re-sends all stored messages
// accomplishing lower and upper limits. The limit parameter determines the maximum number of
// messages to be sent back for the current request.
// The cursor parameter is used for pagination.
// After sending all the messages, a message of type p2pRequestCompleteCode is sent by the mailserver to
// the peer.
func (s *WMailServer) processRequest(
	peer *whisper.Peer,
	lower, upper uint32,
	bloom []byte,
	limit uint32,
	cursor cursorType,
	batch bool,
) (ret []*whisper.Envelope, lastEnvelopeHash common.Hash, nextPageCursor cursorType, err error) {
	// Recover from possible goleveldb panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic in processRequest: %v", r)
		}
	}()

	var (
		sentEnvelopes     uint32
		sentEnvelopesSize int64
	)

	i := s.createIterator(lower, upper, cursor)
	defer i.Release()

	var (
		bundle     []*whisper.Envelope
		bundleSize uint32
	)

	start := time.Now()

	for i.Prev() {
		var envelope whisper.Envelope
		decodeErr := rlp.DecodeBytes(i.Value(), &envelope)
		if decodeErr != nil {
			log.Error("failed to decode RLP", "err", decodeErr)
			continue
		}

		if !whisper.BloomFilterMatch(bloom, envelope.Bloom()) {
			continue
		}

		newSize := bundleSize + whisper.EnvelopeHeaderLength + uint32(len(envelope.Data))
		limitReached := limit != noLimits && (int(sentEnvelopes)+len(bundle)) == int(limit)
		if !limitReached && newSize < s.w.MaxMessageSize() {
			bundle = append(bundle, &envelope)
			bundleSize = newSize
			lastEnvelopeHash = envelope.Hash()
			continue
		}

		if peer == nil {
			// used for test purposes
			ret = append(ret, bundle...)
		} else {
			err = s.sendEnvelopes(peer, bundle, batch)
			if err != nil {
				return
			}
		}

		sentEnvelopes += uint32(len(bundle))
		sentEnvelopesSize += int64(bundleSize)

		if limitReached {
			bundle = nil
			bundleSize = 0

			// When the limit is reached, the current retrieved envelope
			// is not included in the response.
			// The nextPageCursor is a key used as a limit in a range and
			// is not included in the range, hence, we need to get
			// the previous iterator key.
			i.Next()
			nextPageCursor = i.Key()
			break
		} else {
			// Reset bundle information and add the last read envelope
			// which did not make in the last batch.
			bundle = []*whisper.Envelope{&envelope}
			bundleSize = whisper.EnvelopeHeaderLength + uint32(len(envelope.Data))
		}

		lastEnvelopeHash = envelope.Hash()
	}

	// Send any outstanding envelopes.
	if len(bundle) > 0 && bundleSize > 0 {
		if peer == nil {
			ret = append(ret, bundle...)
		} else {
			err = s.sendEnvelopes(peer, bundle, batch)
			if err != nil {
				return
			}
		}
	}

	requestProcessTimer.UpdateSince(start)
	sentEnvelopesMeter.Mark(int64(sentEnvelopes))
	sentEnvelopesSizeMeter.Mark(sentEnvelopesSize)

	err = i.Error()
	if err != nil {
		err = fmt.Errorf("levelDB iterator error: %v", err)
	}

	return
}

func (s *WMailServer) sendEnvelopes(peer *whisper.Peer, envelopes []*whisper.Envelope, batch bool) error {
	start := time.Now()
	defer requestProcessNetTimer.UpdateSince(start)

	if batch {
		return s.w.SendP2PDirect(peer, envelopes...)
	}

	for _, env := range envelopes {
		if err := s.w.SendP2PDirect(peer, env); err != nil {
			return err
		}
	}

	return nil
}

func (s *WMailServer) sendHistoricMessageResponse(peer *whisper.Peer, request *whisper.Envelope, lastEnvelopeHash common.Hash, cursor cursorType) error {
	payload := whisper.CreateMailServerRequestCompletedPayload(request.Hash(), lastEnvelopeHash, cursor)
	return s.w.SendHistoricMessageResponse(peer, payload)
}

// this method doesn't return an error because it is already in the error handling chain
func (s *WMailServer) trySendHistoricMessageErrorResponse(peer *whisper.Peer, request *whisper.Envelope, errorToReport error) {
	payload := whisper.CreateMailServerRequestFailedPayload(request.Hash(), errorToReport)

	err := s.w.SendHistoricMessageResponse(peer, payload)
	// if we can't report an error, probably something is wrong with p2p connection,
	// so we just print a log entry to document this sad fact
	if err != nil {
		log.Error("Error while reporting error response", "err", err, "peerID", peerIDString(peer))
	}
}

// openEnvelope tries to decrypt an envelope, first based on asymetric key (if
// provided) and second on the symetric key (if provided)
func (s *WMailServer) openEnvelope(request *whisper.Envelope) *whisper.ReceivedMessage {
	if s.asymFilter != nil {
		if d := request.Open(s.asymFilter); d != nil {
			return d
		}
	}
	if s.symFilter != nil {
		if d := request.Open(s.symFilter); d != nil {
			return d
		}
	}
	return nil
}

func (s *WMailServer) decodeRequest(peerID []byte, request *whisper.Envelope) (shhext.MessagesRequestPayload, error) {
	var payload shhext.MessagesRequestPayload

	if s.pow > 0.0 && request.PoW() < s.pow {
		return payload, errors.New("PoW too low")
	}

	decrypted := s.openEnvelope(request)
	if decrypted == nil {
		log.Warn("Failed to decrypt p2p request")
		return payload, errors.New("failed to decrypt p2p request")
	}

	if err := s.checkMsgSignature(decrypted, peerID); err != nil {
		log.Warn("Check message signature failed: %s", "err", err.Error())
		return payload, fmt.Errorf("check message signature failed: %v", err)
	}

	if err := rlp.DecodeBytes(decrypted.Payload, &payload); err != nil {
		return payload, fmt.Errorf("failed to decode data: %v", err)
	}

	if payload.Upper < payload.Lower {
		log.Error("Query range is invalid: lower > upper", "lower", payload.Lower, "upper", payload.Upper)
		return payload, errors.New("query range is invalid: lower > upper")
	}

	lowerTime := time.Unix(int64(payload.Lower), 0)
	upperTime := time.Unix(int64(payload.Upper), 0)
	if upperTime.Sub(lowerTime) > maxQueryRange {
		log.Warn("Query range too long", "peerID", peerIDBytesString(peerID), "length", upperTime.Sub(lowerTime), "max", maxQueryRange)
		return payload, fmt.Errorf("query range must be shorted than %d", maxQueryRange)
	}

	return payload, nil
}

// validateRequest runs different validations on the current request.
// DEPRECATED
func (s *WMailServer) validateRequest(
	peerID []byte,
	request *whisper.Envelope,
) (uint32, uint32, []byte, uint32, cursorType, error) {
	if s.pow > 0.0 && request.PoW() < s.pow {
		return 0, 0, nil, 0, nil, fmt.Errorf("PoW() is too low")
	}

	decrypted := s.openEnvelope(request)
	if decrypted == nil {
		return 0, 0, nil, 0, nil, fmt.Errorf("failed to decrypt p2p request")
	}

	if err := s.checkMsgSignature(decrypted, peerID); err != nil {
		return 0, 0, nil, 0, nil, err
	}

	bloom, err := s.bloomFromReceivedMessage(decrypted)
	if err != nil {
		return 0, 0, nil, 0, nil, err
	}

	lower := binary.BigEndian.Uint32(decrypted.Payload[:4])
	upper := binary.BigEndian.Uint32(decrypted.Payload[4:8])

	if upper < lower {
		err := fmt.Errorf("query range is invalid: from > to (%d > %d)", lower, upper)
		return 0, 0, nil, 0, nil, err
	}

	lowerTime := time.Unix(int64(lower), 0)
	upperTime := time.Unix(int64(upper), 0)
	if upperTime.Sub(lowerTime) > maxQueryRange {
		err := fmt.Errorf("query range too big for peer %s", string(peerID))
		return 0, 0, nil, 0, nil, err
	}

	var limit uint32
	if len(decrypted.Payload) >= requestTimeRangeLength+whisper.BloomFilterSize+requestLimitLength {
		limit = binary.BigEndian.Uint32(decrypted.Payload[requestTimeRangeLength+whisper.BloomFilterSize:])
	}

	var cursor cursorType
	if len(decrypted.Payload) == requestTimeRangeLength+whisper.BloomFilterSize+requestLimitLength+dbKeyLength {
		cursor = decrypted.Payload[requestTimeRangeLength+whisper.BloomFilterSize+requestLimitLength:]
	}

	err = nil
	return lower, upper, bloom, limit, cursor, err
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

// peerWithID is a generalization of whisper.Peer.
// whisper.Peer has all fields and methods, except for ID(), unexported.
// It makes it impossible to create an instance of it
// outside of whisper package and test properly.
type peerWithID interface {
	ID() []byte
}

func peerIDString(peer peerWithID) string {
	return fmt.Sprintf("%x", peer.ID())
}

func peerIDBytesString(id []byte) string {
	return fmt.Sprintf("%x", id)
}
