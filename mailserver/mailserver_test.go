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
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	waku "github.com/status-im/status-go/waku"
	wakucommon "github.com/status-im/status-go/waku/common"
)

const powRequirement = 0.00001

var keyID string
var seed = time.Now().Unix()
var testPayload = []byte("test payload")

type ServerTestParams struct {
	topic types.TopicType
	birth uint32
	low   uint32
	upp   uint32
	limit uint32
	key   *ecdsa.PrivateKey
}

func TestMailserverSuite(t *testing.T) {
	suite.Run(t, new(MailserverSuite))
}

type MailserverSuite struct {
	suite.Suite
	server  *WakuMailServer
	shh     *waku.Waku
	config  *params.WakuConfig
	dataDir string
}

func (s *MailserverSuite) SetupTest() {
	s.server = &WakuMailServer{}
	s.shh = waku.New(&waku.DefaultConfig, nil)
	s.shh.RegisterMailServer(s.server)

	tmpDir := s.T().TempDir()
	s.dataDir = tmpDir

	s.config = &params.WakuConfig{
		DataDir:            tmpDir,
		MailServerPassword: "testpassword",
	}
}

func (s *MailserverSuite) TestInit() {
	testCases := []struct {
		config        params.WakuConfig
		expectedError error
		info          string
	}{
		{
			config:        params.WakuConfig{DataDir: ""},
			expectedError: errDirectoryNotProvided,
			info:          "config with empty DataDir",
		},
		{
			config: params.WakuConfig{
				DataDir:            s.config.DataDir,
				MailServerPassword: "pwd",
			},
			expectedError: nil,
			info:          "config with correct DataDir and Password",
		},
		{
			config: params.WakuConfig{
				DataDir:             s.config.DataDir,
				MailServerPassword:  "pwd",
				MailServerRateLimit: 5,
			},
			expectedError: nil,
			info:          "config with rate limit",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.info, func(*testing.T) {
			mailServer := &WakuMailServer{}
			shh := waku.New(&waku.DefaultConfig, nil)
			shh.RegisterMailServer(mailServer)

			err := mailServer.Init(shh, &tc.config)
			s.Require().Equal(tc.expectedError, err)
			if err == nil {
				defer mailServer.Close()
			}

			// db should be open only if there was no error
			if tc.expectedError == nil {
				s.NotNil(mailServer.ms.db)
			} else {
				s.Nil(mailServer.ms)
			}

			if tc.config.MailServerRateLimit > 0 {
				s.NotNil(mailServer.ms.rateLimiter)
			}
		})
	}
}

func (s *MailserverSuite) TestArchive() {
	config := *s.config

	err := s.server.Init(s.shh, &config)
	s.Require().NoError(err)
	defer s.server.Close()

	env, err := generateEnvelope(time.Now())
	s.NoError(err)
	rawEnvelope, err := rlp.EncodeToBytes(env)
	s.NoError(err)

	s.server.Archive(env)
	key := NewDBKey(env.Expiry-env.TTL, types.TopicType(env.Topic), types.Hash(env.Hash()))
	archivedEnvelope, err := s.server.ms.db.GetEnvelope(key)
	s.NoError(err)

	s.Equal(rawEnvelope, archivedEnvelope)
}

func (s *MailserverSuite) TestManageLimits() {
	err := s.server.Init(s.shh, s.config)
	s.NoError(err)
	s.server.ms.rateLimiter = newRateLimiter(time.Duration(5) * time.Millisecond)
	s.False(s.server.ms.exceedsPeerRequests(types.BytesToHash([]byte("peerID"))))
	s.Equal(1, len(s.server.ms.rateLimiter.db))
	firstSaved := s.server.ms.rateLimiter.db["peerID"]

	// second call when limit is not accomplished does not store a new limit
	s.True(s.server.ms.exceedsPeerRequests(types.BytesToHash([]byte("peerID"))))
	s.Equal(1, len(s.server.ms.rateLimiter.db))
	s.Equal(firstSaved, s.server.ms.rateLimiter.db["peerID"])
}

func (s *MailserverSuite) TestDBKey() {
	var h types.Hash
	var emptyTopic types.TopicType
	i := uint32(time.Now().Unix())
	k := NewDBKey(i, emptyTopic, h)
	s.Equal(len(k.Bytes()), DBKeyLength, "wrong DB key length")
	s.Equal(byte(i%0x100), k.Bytes()[3], "raw representation should be big endian")
	s.Equal(byte(i/0x1000000), k.Bytes()[0], "big endian expected")
}

func (s *MailserverSuite) TestRequestPaginationLimit() {
	s.setupServer(s.server)
	defer s.server.Close()

	var (
		sentEnvelopes  []*wakucommon.Envelope
		sentHashes     []common.Hash
		receivedHashes []common.Hash
		archiveKeys    []string
	)

	now := time.Now()
	count := uint32(10)

	for i := count; i > 0; i-- {
		sentTime := now.Add(time.Duration(-i) * time.Second)
		env, err := generateEnvelope(sentTime)
		s.NoError(err)
		s.server.Archive(env)
		key := NewDBKey(env.Expiry-env.TTL, types.TopicType(env.Topic), types.Hash(env.Hash()))
		archiveKeys = append(archiveKeys, fmt.Sprintf("%x", key.Cursor()))
		sentEnvelopes = append(sentEnvelopes, env)
		sentHashes = append(sentHashes, env.Hash())
	}

	reqLimit := uint32(6)
	peerID, request, err := s.prepareRequest(sentEnvelopes, reqLimit)
	s.NoError(err)
	payload, err := s.server.decompositeRequest(peerID, request)
	s.NoError(err)
	s.Nil(payload.Cursor)
	s.Equal(reqLimit, payload.Limit)

	receivedHashes, cursor, _ := processRequestAndCollectHashes(s.server, payload)

	// 10 envelopes sent
	s.Equal(count, uint32(len(sentEnvelopes)))
	// 6 envelopes received
	s.Len(receivedHashes, int(payload.Limit))
	// the 6 envelopes received should be in forward order
	s.Equal(sentHashes[:payload.Limit], receivedHashes)
	// cursor should be the key of the last envelope of the last page
	s.Equal(archiveKeys[payload.Limit-1], fmt.Sprintf("%x", cursor))

	// second page
	payload.Cursor = cursor
	receivedHashes, cursor, _ = processRequestAndCollectHashes(s.server, payload)

	// 4 envelopes received
	s.Equal(int(count-payload.Limit), len(receivedHashes))
	// cursor is nil because there are no other pages
	s.Nil(cursor)
}

func (s *MailserverSuite) TestMailServer() {
	s.setupServer(s.server)
	defer s.server.Close()

	env, err := generateEnvelope(time.Now())
	s.NoError(err)

	s.server.Archive(env)

	testCases := []struct {
		params *ServerTestParams
		expect bool
		isOK   bool
		info   string
	}{
		{
			params: s.defaultServerParams(env),
			expect: true,
			isOK:   true,
			info:   "Processing a request where from and to are equal to an existing register, should provide results",
		},
		{
			params: func() *ServerTestParams {
				params := s.defaultServerParams(env)
				params.low = params.birth + 1
				params.upp = params.birth + 1

				return params
			}(),
			expect: false,
			isOK:   true,
			info:   "Processing a request where from and to are greater than any existing register, should not provide results",
		},
		{
			params: func() *ServerTestParams {
				params := s.defaultServerParams(env)
				params.upp = params.birth + 1
				params.topic[0] = 0xFF

				return params
			}(),
			expect: false,
			isOK:   true,
			info:   "Processing a request where to is greater than any existing register and with a specific topic, should not provide results",
		},
		{
			params: func() *ServerTestParams {
				params := s.defaultServerParams(env)
				params.low = params.birth
				params.upp = params.birth - 1

				return params
			}(),
			isOK: false,
			info: "Processing a request where to is lower than from should fail",
		},
		{
			params: func() *ServerTestParams {
				params := s.defaultServerParams(env)
				params.low = 0
				params.upp = params.birth + 24

				return params
			}(),
			isOK: false,
			info: "Processing a request where difference between from and to is > 24 should fail",
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.info, func(*testing.T) {
			request := s.createRequest(tc.params)
			src := crypto.FromECDSAPub(&tc.params.key.PublicKey)
			payload, err := s.server.decompositeRequest(src, request)
			s.Equal(tc.isOK, err == nil)
			if err == nil {
				s.Equal(tc.params.low, payload.Lower)
				s.Equal(tc.params.upp, payload.Upper)
				s.Equal(tc.params.limit, payload.Limit)
				s.Equal(types.TopicToBloom(tc.params.topic), payload.Bloom)
				s.Equal(tc.expect, s.messageExists(env, tc.params.low, tc.params.upp, payload.Bloom, tc.params.limit))

				src[0]++
				_, err = s.server.decompositeRequest(src, request)
				s.True(err == nil)
			}
		})
	}
}

func (s *MailserverSuite) TestDecodeRequest() {
	s.setupServer(s.server)
	defer s.server.Close()

	payload := MessagesRequestPayload{
		Lower:  50,
		Upper:  100,
		Bloom:  []byte{0x01},
		Topics: [][]byte{},
		Limit:  10,
		Cursor: []byte{},
		Batch:  true,
	}
	data, err := rlp.EncodeToBytes(payload)
	s.Require().NoError(err)

	id, err := s.shh.NewKeyPair()
	s.Require().NoError(err)
	srcKey, err := s.shh.GetPrivateKey(id)
	s.Require().NoError(err)

	env := s.createEnvelope(types.TopicType{0x01}, data, srcKey)

	decodedPayload, err := s.server.decodeRequest(nil, env)
	s.Require().NoError(err)
	s.Equal(payload, decodedPayload)
}

func (s *MailserverSuite) TestDecodeRequestNoUpper() {
	s.setupServer(s.server)
	defer s.server.Close()

	payload := MessagesRequestPayload{
		Lower:  50,
		Bloom:  []byte{0x01},
		Limit:  10,
		Cursor: []byte{},
		Batch:  true,
	}
	data, err := rlp.EncodeToBytes(payload)
	s.Require().NoError(err)

	id, err := s.shh.NewKeyPair()
	s.Require().NoError(err)
	srcKey, err := s.shh.GetPrivateKey(id)
	s.Require().NoError(err)

	env := s.createEnvelope(types.TopicType{0x01}, data, srcKey)

	decodedPayload, err := s.server.decodeRequest(nil, env)
	s.Require().NoError(err)
	s.NotEqual(0, decodedPayload.Upper)
}

func (s *MailserverSuite) TestProcessRequestDeadlockHandling() {
	s.setupServer(s.server)
	defer s.server.Close()

	var archievedEnvelopes []*wakucommon.Envelope

	now := time.Now()
	count := uint32(10)

	// Archieve some envelopes.
	for i := count; i > 0; i-- {
		sentTime := now.Add(time.Duration(-i) * time.Second)
		env, err := generateEnvelope(sentTime)
		s.NoError(err)
		s.server.Archive(env)
		archievedEnvelopes = append(archievedEnvelopes, env)
	}

	// Prepare a request.
	peerID, request, err := s.prepareRequest(archievedEnvelopes, 5)
	s.NoError(err)
	payload, err := s.server.decompositeRequest(peerID, request)
	s.NoError(err)

	testCases := []struct {
		Name    string
		Timeout time.Duration
		Verify  func(
			Iterator,
			time.Duration, // processRequestInBundles timeout
			chan []rlp.RawValue,
		)
	}{
		{
			Name:    "finish processing using `done` channel",
			Timeout: time.Second * 5,
			Verify: func(
				iter Iterator,
				timeout time.Duration,
				bundles chan []rlp.RawValue,
			) {
				done := make(chan struct{})
				processFinished := make(chan struct{})

				go func() {
					s.server.ms.processRequestInBundles(iter, payload.Bloom, payload.Topics, int(payload.Limit), timeout, "req-01", bundles, done)
					close(processFinished)
				}()
				go close(done)

				select {
				case <-processFinished:
				case <-time.After(time.Second):
					s.FailNow("waiting for processing finish timed out")
				}
			},
		},
		{
			Name:    "finish processing due to timeout",
			Timeout: time.Second,
			Verify: func(
				iter Iterator,
				timeout time.Duration,
				bundles chan []rlp.RawValue,
			) {
				done := make(chan struct{}) // won't be closed because we test timeout of `processRequestInBundles()`
				processFinished := make(chan struct{})

				go func() {
					s.server.ms.processRequestInBundles(iter, payload.Bloom, payload.Topics, int(payload.Limit), time.Second, "req-01", bundles, done)
					close(processFinished)
				}()

				select {
				case <-processFinished:
				case <-time.After(time.Second * 5):
					s.FailNow("waiting for processing finish timed out")
				}
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.Name, func(t *testing.T) {
			iter, err := s.server.ms.createIterator(payload)
			s.Require().NoError(err)

			defer func() { _ = iter.Release() }()

			// Nothing reads from this unbuffered channel which simulates a situation
			// when a connection between a peer and mail server was dropped.
			bundles := make(chan []rlp.RawValue)

			tc.Verify(iter, tc.Timeout, bundles)
		})
	}
}

func (s *MailserverSuite) messageExists(envelope *wakucommon.Envelope, low, upp uint32, bloom []byte, limit uint32) bool {
	receivedHashes, _, _ := processRequestAndCollectHashes(s.server, MessagesRequestPayload{
		Lower: low,
		Upper: upp,
		Bloom: bloom,
		Limit: limit,
	})
	for _, hash := range receivedHashes {
		if hash == envelope.Hash() {
			return true
		}
	}
	return false
}

func (s *MailserverSuite) setupServer(server *WakuMailServer) {
	const password = "password_for_this_test"

	s.shh = waku.New(&waku.DefaultConfig, nil)
	s.shh.RegisterMailServer(server)

	err := server.Init(s.shh, &params.WakuConfig{
		DataDir:            s.dataDir,
		MailServerPassword: password,
		MinimumPoW:         powRequirement,
	})
	if err != nil {
		s.T().Fatal(err)
	}

	keyID, err = s.shh.AddSymKeyFromPassword(password)
	if err != nil {
		s.T().Fatalf("failed to create symmetric key for mail request: %s", err)
	}
}

func (s *MailserverSuite) prepareRequest(envelopes []*wakucommon.Envelope, limit uint32) (
	[]byte, *wakucommon.Envelope, error,
) {
	if len(envelopes) == 0 {
		return nil, nil, errors.New("envelopes is empty")
	}

	now := time.Now()

	params := s.defaultServerParams(envelopes[0])
	params.low = uint32(now.Add(time.Duration(-len(envelopes)) * time.Second).Unix())
	params.upp = uint32(now.Unix())
	params.limit = limit

	request := s.createRequest(params)
	peerID := crypto.FromECDSAPub(&params.key.PublicKey)

	return peerID, request, nil
}

func (s *MailserverSuite) defaultServerParams(env *wakucommon.Envelope) *ServerTestParams {
	id, err := s.shh.NewKeyPair()
	if err != nil {
		s.T().Fatalf("failed to generate new key pair with seed %d: %s.", seed, err)
	}
	testPeerID, err := s.shh.GetPrivateKey(id)
	if err != nil {
		s.T().Fatalf("failed to retrieve new key pair with seed %d: %s.", seed, err)
	}
	birth := env.Expiry - env.TTL

	return &ServerTestParams{
		topic: types.TopicType(env.Topic),
		birth: birth,
		low:   birth - 1,
		upp:   birth + 1,
		limit: 0,
		key:   testPeerID,
	}
}

func (s *MailserverSuite) createRequest(p *ServerTestParams) *wakucommon.Envelope {
	bloom := types.TopicToBloom(p.topic)
	data := make([]byte, 8)
	binary.BigEndian.PutUint32(data, p.low)
	binary.BigEndian.PutUint32(data[4:], p.upp)
	data = append(data, bloom...)

	if p.limit != 0 {
		limitData := make([]byte, 4)
		binary.BigEndian.PutUint32(limitData, p.limit)
		data = append(data, limitData...)
	}

	return s.createEnvelope(p.topic, data, p.key)
}

func (s *MailserverSuite) createEnvelope(topic types.TopicType, data []byte, srcKey *ecdsa.PrivateKey) *wakucommon.Envelope {
	key, err := s.shh.GetSymKey(keyID)
	if err != nil {
		s.T().Fatalf("failed to retrieve sym key with seed %d: %s.", seed, err)
	}

	params := &wakucommon.MessageParams{
		KeySym:   key,
		Topic:    wakucommon.TopicType(topic),
		Payload:  data,
		PoW:      powRequirement * 2,
		WorkTime: 2,
		Src:      srcKey,
	}

	msg, err := wakucommon.NewSentMessage(params)
	if err != nil {
		s.T().Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}

	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		s.T().Fatalf("failed to wrap with seed %d: %s.", seed, err)
	}
	return env
}

func generateEnvelopeWithKeys(sentTime time.Time, keySym []byte, keyAsym *ecdsa.PublicKey) (*wakucommon.Envelope, error) {
	params := &wakucommon.MessageParams{
		Topic:    wakucommon.TopicType{0x1F, 0x7E, 0xA1, 0x7F},
		Payload:  testPayload,
		PoW:      powRequirement,
		WorkTime: 2,
	}

	if len(keySym) > 0 {
		params.KeySym = keySym
	} else if keyAsym != nil {
		params.Dst = keyAsym
	}

	msg, err := wakucommon.NewSentMessage(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create new message with seed %d: %s", seed, err)
	}
	env, err := msg.Wrap(params, sentTime)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap with seed %d: %s", seed, err)
	}

	return env, nil
}

func generateEnvelope(sentTime time.Time) (*wakucommon.Envelope, error) {
	h := crypto.Keccak256Hash([]byte("test sample data"))
	return generateEnvelopeWithKeys(sentTime, h[:], nil)
}

func processRequestAndCollectHashes(server *WakuMailServer, payload MessagesRequestPayload) ([]common.Hash, []byte, types.Hash) {
	iter, _ := server.ms.createIterator(payload)
	defer func() { _ = iter.Release() }()
	bundles := make(chan []rlp.RawValue, 10)
	done := make(chan struct{})

	var hashes []common.Hash
	go func() {
		for bundle := range bundles {
			for _, rawEnvelope := range bundle {
				var env *wakucommon.Envelope
				if err := rlp.DecodeBytes(rawEnvelope, &env); err != nil {
					panic(err)
				}
				hashes = append(hashes, env.Hash())
			}
		}
		close(done)
	}()

	cursor, lastHash := server.ms.processRequestInBundles(iter, payload.Bloom, payload.Topics, int(payload.Limit), time.Minute, "req-01", bundles, done)

	<-done

	return hashes, cursor, lastHash
}
