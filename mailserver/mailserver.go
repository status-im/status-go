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
	syncRequestsMeter              = metrics.NewRegisteredMeter("mailserver/syncRequests", nil)
)

const (
	// DBKeyLength is a size of the envelope key.
	DBKeyLength = common.HashLength + timestampLength

	timestampLength        = 4
	requestLimitLength     = 4
	requestTimeRangeLength = timestampLength * 2
)

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

// Bytes returns a bytes representation of the DBKey.
func (k *DBKey) Bytes() []byte {
	return k.raw
}

// NewDBKey creates a new DBKey with the given values.
func NewDBKey(t uint32, h common.Hash) *DBKey {
	var k DBKey
	k.timestamp = t
	k.hash = h
	k.raw = make([]byte, DBKeyLength)
	binary.BigEndian.PutUint32(k.raw, k.timestamp)
	copy(k.raw[4:], k.hash[:])
	return &k
}

// NewDBKeyFromBytes creates a DBKey from a byte slice.
func NewDBKeyFromBytes(b []byte) *DBKey {
	return &DBKey{
		raw:       b,
		timestamp: binary.BigEndian.Uint32(b),
		hash:      common.BytesToHash(b[4:]),
	}
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

	key := NewDBKey(env.Expiry-env.TTL, env.Hash())
	rawEnvelope, err := rlp.EncodeToBytes(env)
	if err != nil {
		log.Error(fmt.Sprintf("rlp.EncodeToBytes failed: %s", err))
		archivedErrorsCounter.Inc(1)
	} else {
		if err = s.db.Put(key.Bytes(), rawEnvelope, nil); err != nil {
			log.Error(fmt.Sprintf("Writing to DB failed: %s", err))
			archivedErrorsCounter.Inc(1)
		}
		archivedMeter.Mark(1)
		archivedSizeMeter.Mark(int64(whisper.EnvelopeHeaderLength + len(env.Data)))
	}
}

// DeliverMail sends mail to specified whisper peer.
func (s *WMailServer) DeliverMail(peer *whisper.Peer, request *whisper.Envelope) {
	defer recoverLevelDBPanics("DeliverMail")

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

	var (
		lower, upper uint32
		bloom        []byte
		limit        uint32
		cursor       []byte
		batch        bool
		err          error
	)

	payload, err := s.decodeRequest(peer.ID(), request)
	if err == nil {
		lower, upper = payload.Lower, payload.Upper
		bloom = payload.Bloom
		cursor = payload.Cursor
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
		"lower", lower,
		"upper", upper,
		"bloom", bloom,
		"limit", limit,
		"cursor", cursor,
		"batch", batch,
	)

	if batch {
		requestsBatchedCounter.Inc(1)
	}

	iter := s.createIterator(lower, upper, cursor)
	defer iter.Release()

	bundles := make(chan []*whisper.Envelope, 5)
	errCh := make(chan error)

	go func() {
		for bundle := range bundles {
			if err := s.sendEnvelopes(peer, bundle, batch); err != nil {
				errCh <- err
				break
			}
		}
		close(errCh)
	}()

	start := time.Now()
	nextPageCursor, lastEnvelopeHash := s.processRequestInBundles(
		iter,
		bloom,
		int(limit),
		bundles,
	)
	requestProcessTimer.UpdateSince(start)

	// bundles channel can be closed now because the processing finished.
	close(bundles)

	// Wait for the goroutine to finish the work. It may return an error.
	if err := <-errCh; err != nil {
		processRequestErrorsCounter.Inc(1)
		log.Error("Error while processing mail server request", "err", err, "peerID", peerIDString(peer))
		s.trySendHistoricMessageErrorResponse(peer, request, err)
		return
	}

	// Processing of the request could be finished earlier due to iterator error.
	if err := iter.Error(); err != nil {
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

// SyncMail syncs mail servers between two Mail Servers.
func (s *WMailServer) SyncMail(peer *whisper.Peer, request whisper.SyncMailRequest) error {
	log.Info("Started syncing envelopes", "peer", peerIDString(peer), "req", request)

	defer recoverLevelDBPanics("SyncMail")

	syncRequestsMeter.Mark(1)

	// Check rate limiting for a requesting peer.
	if s.exceedsPeerRequests(peer.ID()) {
		requestErrorsCounter.Inc(1)
		log.Error("Peer exceeded request per seconds limit", "peerID", peerIDString(peer))
		return fmt.Errorf("requests per seconds limit exceeded")
	}

	iter := s.createIterator(request.Lower, request.Upper, request.Cursor)
	defer iter.Release()

	bundles := make(chan []*whisper.Envelope, 5)
	errCh := make(chan error)

	go func() {
		for bundle := range bundles {
			resp := whisper.SyncResponse{Envelopes: bundle}
			if err := s.w.SendSyncResponse(peer, resp); err != nil {
				errCh <- fmt.Errorf("failed to send sync response: %v", err)
				break
			}
		}
		close(errCh)
	}()

	start := time.Now()
	nextCursor, _ := s.processRequestInBundles(
		iter,
		request.Bloom,
		int(request.Limit),
		bundles,
	)
	requestProcessTimer.UpdateSince(start)

	// bundles channel can be closed now because the processing finished.
	close(bundles)

	// Wait for the goroutine to finish the work. It may return an error.
	if err := <-errCh; err != nil {
		_ = s.w.SendSyncResponse(
			peer,
			whisper.SyncResponse{Error: "failed to send a response"},
		)
		return err
	}

	// Processing of the request could be finished earlier due to iterator error.
	if err := iter.Error(); err != nil {
		_ = s.w.SendSyncResponse(
			peer,
			whisper.SyncResponse{Error: "failed to process all envelopes"},
		)
		return fmt.Errorf("levelDB iterator failed: %v", err)
	}

	log.Info("Finished syncing envelopes", "peer", peerIDString(peer))

	if err := s.w.SendSyncResponse(peer, whisper.SyncResponse{
		Cursor: nextCursor,
		Final:  true,
	}); err != nil {
		return fmt.Errorf("failed to send the final sync response: %v", err)
	}

	return nil
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

func (s *WMailServer) createIterator(lower, upper uint32, cursor []byte) iterator.Iterator {
	var (
		emptyHash common.Hash
		ku, kl    *DBKey
	)

	kl = NewDBKey(lower, emptyHash)
	if len(cursor) == DBKeyLength {
		ku = NewDBKeyFromBytes(cursor)
	} else {
		ku = NewDBKey(upper+1, emptyHash)
	}

	i := s.db.NewIterator(&util.Range{Start: kl.Bytes(), Limit: ku.Bytes()}, nil)
	// seek to the end as we want to return envelopes in a descending order
	i.Seek(ku.Bytes())

	return i
}

// processRequestInBundles processes envelopes using an iterator and passes them
// to the output channel in bundles.
func (s *WMailServer) processRequestInBundles(
	iter iterator.Iterator, bloom []byte, limit int, output chan<- []*whisper.Envelope,
) ([]byte, common.Hash) {
	var (
		bundle                 []*whisper.Envelope
		bundleSize             uint32
		processedEnvelopes     int
		processedEnvelopesSize int64
		nextCursor             []byte
		lastEnvelopeHash       common.Hash
	)

	for iter.Prev() {
		var envelope whisper.Envelope

		decodeErr := rlp.DecodeBytes(iter.Value(), &envelope)
		if decodeErr != nil {
			log.Error("failed to decode RLP", "err", decodeErr)
			continue
		}

		if !whisper.BloomFilterMatch(bloom, envelope.Bloom()) {
			continue
		}

		newSize := bundleSize + whisper.EnvelopeHeaderLength + uint32(len(envelope.Data))
		limitReached := limit != noLimits && (processedEnvelopes+len(bundle)) == limit
		if !limitReached && newSize < s.w.MaxMessageSize() {
			bundle = append(bundle, &envelope)
			bundleSize = newSize
			lastEnvelopeHash = envelope.Hash()
			continue
		}

		output <- bundle
		bundle = bundle[:0]
		bundleSize = 0

		processedEnvelopes += len(bundle)
		processedEnvelopesSize += int64(bundleSize)

		if limitReached {
			// When the limit is reached, the current retrieved envelope
			// is not included in the response.
			// The nextCursor is a key used as a limit in a range and
			// is not included in the range, hence, we need to get
			// the previous iterator key.
			// Because we iterate backwards, we use Next().
			iter.Next()
			nextCursor = iter.Key()
			break
		} else {
			// Reset bundle information and add the last read envelope
			// which did not make in the last batch.
			bundle = []*whisper.Envelope{&envelope}
			bundleSize = whisper.EnvelopeHeaderLength + uint32(len(envelope.Data))
		}

		lastEnvelopeHash = envelope.Hash()
	}

	// There might be some outstanding elements in the bundle.
	if len(bundle) > 0 && bundleSize > 0 {
		output <- bundle
	}

	sentEnvelopesMeter.Mark(int64(processedEnvelopes))
	sentEnvelopesSizeMeter.Mark(processedEnvelopesSize)

	return nextCursor, lastEnvelopeHash
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

func (s *WMailServer) sendHistoricMessageResponse(peer *whisper.Peer, request *whisper.Envelope, lastEnvelopeHash common.Hash, cursor []byte) error {
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

func (s *WMailServer) decodeRequest(peerID []byte, request *whisper.Envelope) (MessagesRequestPayload, error) {
	var payload MessagesRequestPayload

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
) (uint32, uint32, []byte, uint32, []byte, error) {
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

	var cursor []byte
	if len(decrypted.Payload) == requestTimeRangeLength+whisper.BloomFilterSize+requestLimitLength+DBKeyLength {
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
