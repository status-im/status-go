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
	"math/rand"
	"sync"

	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/status-go/params"
	whisper "github.com/status-im/whisper/whisperv6"
)

const (
	maxQueryRange = 24 * time.Hour
	defaultLimit  = 2000
	// When we default the upper limit, we want to extend the range a bit
	// to accommodate for envelopes with slightly higher timestamp, in seconds
	whisperTTLSafeThreshold = 60
)

var (
	errDirectoryNotProvided        = errors.New("data directory not provided")
	errDecryptionMethodNotProvided = errors.New("decryption method is not provided")
)

const (
	timestampLength        = 4
	requestLimitLength     = 4
	requestTimeRangeLength = timestampLength * 2
	processRequestTimeout  = time.Minute
)

// WMailServer whisper mailserver.
type WMailServer struct {
	db         DB
	w          *whisper.Whisper
	pow        float64
	symFilter  *whisper.Filter
	asymFilter *whisper.Filter

	muRateLimiter sync.RWMutex
	rateLimiter   *rateLimiter

	cleaner *dbCleaner // removes old envelopes
}

// Init initializes mailServer.
func (s *WMailServer) Init(shh *whisper.Whisper, config *params.WhisperConfig) error {
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

	if config.MailServerRateLimit > 0 {
		s.setupRateLimiter(time.Duration(config.MailServerRateLimit) * time.Second)
	}

	// Open database in the last step in order not to init with error
	// and leave the database open by accident.
	if config.DatabaseConfig.PGConfig.Enabled {
		log.Info("Connecting to postgres database")
		database, err := NewPostgresDB(config)
		if err != nil {
			return fmt.Errorf("open DB: %s", err)
		}
		s.db = database
		log.Info("Connected to postgres database")
	} else {
		// Defaults to LevelDB
		database, err := NewLevelDB(config)
		if err != nil {
			return fmt.Errorf("open DB: %s", err)
		}
		s.db = database
	}

	if config.MailServerDataRetention > 0 {
		// MailServerDataRetention is a number of days.
		s.setupCleaner(time.Duration(config.MailServerDataRetention) * time.Hour * 24)
	}

	return nil
}

// setupRateLimiter in case limit is bigger than 0 it will setup an automated
// limit db cleanup.
func (s *WMailServer) setupRateLimiter(limit time.Duration) {
	s.rateLimiter = newRateLimiter(limit)
	s.rateLimiter.Start()
}

func (s *WMailServer) setupCleaner(retention time.Duration) {
	s.cleaner = newDBCleaner(s.db, retention)
	s.cleaner.Start()
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

// Close the mailserver and its associated db connection.
func (s *WMailServer) Close() {
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			log.Error(fmt.Sprintf("s.db.Close failed: %s", err))
		}
	}
	if s.rateLimiter != nil {
		s.rateLimiter.Stop()
	}
	if s.cleaner != nil {
		s.cleaner.Stop()
	}
}

// Archive a whisper envelope.
func (s *WMailServer) Archive(env *whisper.Envelope) {
	if err := s.db.SaveEnvelope(env); err != nil {
		log.Error("Could not save envelope", "hash", env.Hash())
	}
}

// DeliverMail sends mail to specified whisper peer.
func (s *WMailServer) DeliverMail(peer *whisper.Peer, request *whisper.Envelope) {

	startMethod := time.Now()
	defer deliverMailTimer.UpdateSince(startMethod)

	requestsMeter.Mark(1)

	if peer == nil {
		requestErrorsCounter.Inc(1)
		log.Error("[mailserver:DeliverMail] peer is nil")
		return
	}

	requestID := request.Hash().String()
	peerID := peerIDString(peer)

	log.Info("[mailserver:DeliverMail] delivering mail",
		"peerID", peerID,
		"requestID", requestID)

	if s.exceedsPeerRequests(peer.ID()) {
		requestErrorsCounter.Inc(1)
		log.Error("[mailserver:DeliverMail] peer exceeded the limit",
			"peerID", peerID,
			"requestID", requestID)
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
		log.Debug("[mailserver:DeliverMail] failed to decode request",
			"err", err,
			"peerID", peerID,
			"requestID", requestID)
		lower, upper, bloom, limit, cursor, err = s.validateRequest(peer.ID(), request)
	}

	if limit == 0 {
		limit = defaultLimit
	}

	if err != nil {
		requestValidationErrorsCounter.Inc(1)
		log.Error("[mailserver:DeliverMail] request failed validaton",
			"peerID", peerID,
			"requestID", requestID)
		s.trySendHistoricMessageErrorResponse(peer, request, err)
		return
	}

	log.Info("[mailserver:DeliverMail] processing request",
		"peerID", peerID,
		"requestID", requestID,
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

	iter, err := s.createIterator(lower, upper, cursor, bloom, limit)
	if err != nil {
		log.Error("[mailserver:DeliverMail] request failed",
			"peerID", peerID,
			"requestID", requestID)

		return
	}

	defer iter.Release()

	bundles := make(chan []rlp.RawValue, 5)
	errCh := make(chan error)
	cancelProcessing := make(chan struct{})

	go func() {
		counter := 0
		for bundle := range bundles {
			if err := s.sendRawEnvelopes(peer, bundle, batch); err != nil {
				close(cancelProcessing)
				errCh <- err
				break
			}
			counter++
		}
		close(errCh)
		log.Info("[mailserver:DeliverMail] finished sending bundles",
			"peerID", peerID,
			"requestID", requestID,
			"counter", counter)
	}()

	start := time.Now()
	nextPageCursor, lastEnvelopeHash := s.processRequestInBundles(
		iter,
		bloom,
		int(limit),
		processRequestTimeout,
		requestID,
		bundles,
		cancelProcessing,
	)
	requestProcessTimer.UpdateSince(start)

	// Wait for the goroutine to finish the work. It may return an error.
	if err := <-errCh; err != nil {
		processRequestErrorsCounter.Inc(1)
		log.Error("[mailserver:DeliverMail] error while processing",
			"err", err,
			"peerID", peerID,
			"requestID", requestID)
		s.trySendHistoricMessageErrorResponse(peer, request, err)
		return
	}

	// Processing of the request could be finished earlier due to iterator error.
	if err := iter.Error(); err != nil {
		processRequestErrorsCounter.Inc(1)
		log.Error("[mailserver:DeliverMail] iterator failed",
			"err", err,
			"peerID", peerID,
			"requestID", requestID)
		s.trySendHistoricMessageErrorResponse(peer, request, err)
		return
	}

	log.Info("[mailserver:DeliverMail] sending historic message response",
		"peerID", peerID,
		"requestID", requestID,
		"last", lastEnvelopeHash,
		"next", nextPageCursor)

	if err := s.sendHistoricMessageResponse(peer, request, lastEnvelopeHash, nextPageCursor); err != nil {
		historicResponseErrorsCounter.Inc(1)
		log.Error("[mailserver:DeliverMail] error sending historic message response",
			"err", err,
			"peerID", peerID,
			"requestID", requestID)
		// we still want to try to report error even it it is a p2p error and it is unlikely
		s.trySendHistoricMessageErrorResponse(peer, request, err)
	}
}

// SyncMail syncs mail servers between two Mail Servers.
func (s *WMailServer) SyncMail(peer *whisper.Peer, request whisper.SyncMailRequest) error {
	log.Info("Started syncing envelopes", "peer", peerIDString(peer), "req", request)

	requestID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(1000))

	syncRequestsMeter.Mark(1)

	// Check rate limiting for a requesting peer.
	if s.exceedsPeerRequests(peer.ID()) {
		requestErrorsCounter.Inc(1)
		log.Error("Peer exceeded request per seconds limit", "peerID", peerIDString(peer))
		return fmt.Errorf("requests per seconds limit exceeded")
	}

	if err := request.Validate(); err != nil {
		return fmt.Errorf("request is invalid: %v", err)
	}

	iter, err := s.createIterator(request.Lower, request.Upper, request.Cursor, nil, 0)
	if err != nil {
		return err
	}
	defer iter.Release()

	bundles := make(chan []rlp.RawValue, 5)
	errCh := make(chan error)
	cancelProcessing := make(chan struct{})

	go func() {
		for bundle := range bundles {
			resp := whisper.RawSyncResponse{Envelopes: bundle}
			if err := s.w.SendRawSyncResponse(peer, resp); err != nil {
				close(cancelProcessing)
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
		processRequestTimeout,
		requestID,
		bundles,
		cancelProcessing,
	)
	requestProcessTimer.UpdateSince(start)

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
		return fmt.Errorf("LevelDB iterator failed: %v", err)
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
	s.muRateLimiter.RLock()
	defer s.muRateLimiter.RUnlock()

	if s.rateLimiter == nil {
		return false
	}

	peerID := string(peer)
	if s.rateLimiter.IsAllowed(peerID) {
		s.rateLimiter.Add(peerID)
		return false
	}

	log.Info("peerID exceeded the number of requests per second")
	return true
}

func (s *WMailServer) createIterator(lower, upper uint32, cursor []byte, bloom []byte, limit uint32) (Iterator, error) {
	var (
		emptyHash  common.Hash
		emptyTopic whisper.TopicType
		ku, kl     *DBKey
	)

	ku = NewDBKey(upper+1, emptyTopic, emptyHash)
	kl = NewDBKey(lower, emptyTopic, emptyHash)

	query := CursorQuery{
		start:  kl.Bytes(),
		end:    ku.Bytes(),
		cursor: cursor,
		bloom:  bloom,
		limit:  limit,
	}
	return s.db.BuildIterator(query)
}

// processRequestInBundles processes envelopes using an iterator and passes them
// to the output channel in bundles.
func (s *WMailServer) processRequestInBundles(
	iter Iterator,
	bloom []byte,
	limit int,
	timeout time.Duration,
	requestID string,
	output chan<- []rlp.RawValue,
	cancel <-chan struct{},
) ([]byte, common.Hash) {
	var (
		bundle                 []rlp.RawValue
		bundleSize             uint32
		batches                [][]rlp.RawValue
		processedEnvelopes     int
		processedEnvelopesSize int64
		nextCursor             []byte
		lastEnvelopeHash       common.Hash
	)

	log.Info("[mailserver:processRequestInBundles] processing request",
		"requestID", requestID,
		"limit", limit)

	// We iterate over the envelopes.
	// We collect envelopes in batches.
	// If there still room and we haven't reached the limit
	// append and continue.
	// Otherwise publish what you have so far, reset the bundle to the
	// current envelope, and leave if we hit the limit
	for iter.Next() {

		rawValue, err := iter.GetEnvelope(bloom)

		if err != nil {
			log.Error("Failed to get envelope from iterator",
				"err", err,
				"requestID", requestID)
			continue

		}

		if rawValue == nil {
			continue
		}

		key, err := iter.DBKey()
		if err != nil {
			log.Error("[mailserver:processRequestInBundles] failed getting key",
				"requestID", requestID)
			break

		}

		lastEnvelopeHash = key.EnvelopeHash()
		processedEnvelopes++
		envelopeSize := uint32(len(rawValue))
		limitReached := processedEnvelopes == limit
		newSize := bundleSize + envelopeSize

		// If we still have some room for messages, add and continue
		if !limitReached && newSize < s.w.MaxMessageSize() {
			bundle = append(bundle, rawValue)
			bundleSize = newSize
			continue
		}

		// Publish if anything is in the bundle (there should always be
		// something unless limit = 1)
		if len(bundle) != 0 {
			batches = append(batches, bundle)
			processedEnvelopesSize += int64(bundleSize)
		}

		// Reset the bundle with the current envelope
		bundle = []rlp.RawValue{rawValue}
		bundleSize = envelopeSize

		// Leave if we reached the limit
		if limitReached {
			nextCursor = key.Cursor()
			break
		}
	}

	if len(bundle) > 0 {
		batches = append(batches, bundle)
		processedEnvelopesSize += int64(bundleSize)
	}

	log.Info("[mailserver:processRequestInBundles] publishing envelopes",
		"requestID", requestID,
		"batchesCount", len(batches),
		"envelopeCount", processedEnvelopes,
		"processedEnvelopesSize", processedEnvelopesSize,
		"cursor", nextCursor)

	// Publish
	for _, batch := range batches {
		select {
		case output <- batch:
		// It might happen that during producing the batches,
		// the connection with the peer goes down and
		// the consumer of `output` channel exits prematurely.
		// In such a case, we should stop pushing batches and exit.
		case <-cancel:
			log.Info("[mailserver:processRequestInBundles] failed to push all batches",
				"requestID", requestID)
			break
		case <-time.After(timeout):
			log.Error("[mailserver:processRequestInBundles] timed out pushing a batch",
				"requestID", requestID)
			break
		}
	}

	sentEnvelopesMeter.Mark(int64(processedEnvelopes))
	sentEnvelopesSizeMeter.Mark(processedEnvelopesSize)

	log.Info("[mailserver:processRequestInBundles] envelopes published",
		"requestID", requestID)
	close(output)

	return nextCursor, lastEnvelopeHash
}

func (s *WMailServer) sendRawEnvelopes(peer *whisper.Peer, envelopes []rlp.RawValue, batch bool) error {
	start := time.Now()
	defer requestProcessNetTimer.UpdateSince(start)

	if batch {
		return s.w.SendRawP2PDirect(peer, envelopes...)
	}

	for _, env := range envelopes {
		if err := s.w.SendRawP2PDirect(peer, env); err != nil {
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

	if payload.Upper == 0 {
		payload.Upper = uint32(time.Now().Unix() + whisperTTLSafeThreshold)
	}

	if payload.Upper < payload.Lower {
		log.Error("Query range is invalid: lower > upper", "lower", payload.Lower, "upper", payload.Upper)
		return payload, errors.New("query range is invalid: lower > upper")
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

// bloomFromReceivedMessage for a given whisper.ReceivedMessage it extracts the
// used bloom filter.
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

func extractBloomFromEncodedEnvelope(rawValue rlp.RawValue) ([]byte, error) {
	var envelope whisper.Envelope
	decodeErr := rlp.DecodeBytes(rawValue, &envelope)
	if decodeErr != nil {
		return nil, decodeErr
	}
	return envelope.Bloom(), nil
}
