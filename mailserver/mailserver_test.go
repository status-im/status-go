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
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/status-im/status-go/params"
	whisper "github.com/status-im/whisper/whisperv6"
	"github.com/stretchr/testify/suite"
)

const powRequirement = 0.00001

var keyID string
var seed = time.Now().Unix()
var testPayload = []byte("test payload")

type ServerTestParams struct {
	topic whisper.TopicType
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
	server  *WMailServer
	shh     *whisper.Whisper
	config  *params.WhisperConfig
	dataDir string
}

func (s *MailserverSuite) SetupTest() {
	s.server = &WMailServer{}
	s.shh = whisper.New(&whisper.DefaultConfig)
	s.shh.RegisterServer(s.server)

	tmpDir, err := ioutil.TempDir("", "mailserver-test")
	s.Require().NoError(err)
	s.dataDir = tmpDir

	// required files to validate mail server decryption method
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	s.config = &params.WhisperConfig{
		DataDir:            tmpDir,
		MailServerAsymKey:  hex.EncodeToString(crypto.FromECDSA(privateKey)),
		MailServerPassword: "testpassword",
	}
}

func (s *MailserverSuite) TearDownTest() {
	s.Require().NoError(os.RemoveAll(s.config.DataDir))
}

func (s *MailserverSuite) TestInit() {
	asymKey, err := crypto.GenerateKey()
	s.Require().NoError(err)

	testCases := []struct {
		config        params.WhisperConfig
		expectedError error
		info          string
	}{
		{
			config:        params.WhisperConfig{DataDir: ""},
			expectedError: errDirectoryNotProvided,
			info:          "config with empty DataDir",
		},
		{
			config: params.WhisperConfig{
				DataDir:            "/invalid-path",
				MailServerPassword: "pwd",
			},
			expectedError: errors.New("open DB: mkdir /invalid-path: permission denied"),
			info:          "config with an unexisting DataDir",
		},
		{
			config: params.WhisperConfig{
				DataDir:            s.config.DataDir,
				MailServerPassword: "",
				MailServerAsymKey:  "",
			},
			expectedError: errDecryptionMethodNotProvided,
			info:          "config with an empty password and empty asym key",
		},
		{
			config: params.WhisperConfig{
				DataDir:            s.config.DataDir,
				MailServerPassword: "pwd",
			},
			expectedError: nil,
			info:          "config with correct DataDir and Password",
		},
		{
			config: params.WhisperConfig{
				DataDir:           s.config.DataDir,
				MailServerAsymKey: hex.EncodeToString(crypto.FromECDSA(asymKey)),
			},
			expectedError: nil,
			info:          "config with correct DataDir and AsymKey",
		},
		{
			config: params.WhisperConfig{
				DataDir:            s.config.DataDir,
				MailServerAsymKey:  hex.EncodeToString(crypto.FromECDSA(asymKey)),
				MailServerPassword: "pwd",
			},
			expectedError: nil,
			info:          "config with both asym key and password",
		},
		{
			config: params.WhisperConfig{
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
			mailServer := &WMailServer{}
			shh := whisper.New(&whisper.DefaultConfig)
			shh.RegisterServer(mailServer)

			err := mailServer.Init(shh, &tc.config)
			s.Equal(tc.expectedError, err)
			defer mailServer.Close()

			// db should be open only if there was no error
			if tc.expectedError == nil {
				s.NotNil(mailServer.db)
			} else {
				s.Nil(mailServer.db)
			}

			if tc.config.MailServerRateLimit > 0 {
				s.NotNil(mailServer.rateLimiter)
			}
		})
	}
}

func (s *MailserverSuite) TestSetupRequestMessageDecryptor() {
	// without configured Password and AsymKey
	config := *s.config
	config.MailServerAsymKey = ""
	config.MailServerPassword = ""
	s.Error(errDecryptionMethodNotProvided, s.server.Init(s.shh, &config))

	// Password should work ok
	config = *s.config
	config.MailServerAsymKey = "" // clear asym key field
	s.NoError(s.server.Init(s.shh, &config))
	s.Require().NotNil(s.server.symFilter)
	s.NotNil(s.server.symFilter.KeySym)
	s.Nil(s.server.asymFilter)
	s.server.Close()

	// AsymKey can also be used
	config = *s.config
	config.MailServerPassword = "" // clear password field
	s.NoError(s.server.Init(s.shh, &config))
	s.Nil(s.server.symFilter) // important: symmetric filter should be nil
	s.Require().NotNil(s.server.asymFilter)
	s.Equal(config.MailServerAsymKey, hex.EncodeToString(crypto.FromECDSA(s.server.asymFilter.KeyAsym)))
	s.server.Close()

	// when Password and AsymKey are set, both are supported
	config = *s.config
	s.NoError(s.server.Init(s.shh, &config))
	s.Require().NotNil(s.server.symFilter)
	s.NotNil(s.server.symFilter.KeySym)
	s.NotNil(s.server.asymFilter.KeyAsym)
	s.server.Close()
}

func (s *MailserverSuite) TestOpenEnvelopeWithSymKey() {
	// Setup the server with a sym key
	config := *s.config
	config.MailServerAsymKey = "" // clear asym key
	s.NoError(s.server.Init(s.shh, &config))

	// Prepare a valid envelope
	s.Require().NotNil(s.server.symFilter)
	symKey := s.server.symFilter.KeySym
	env, err := generateEnvelopeWithKeys(time.Now(), symKey, nil)
	s.Require().NoError(err)

	// Test openEnvelope with a valid envelope
	d := s.server.openEnvelope(env)
	s.NotNil(d)
	s.Equal(testPayload, d.Payload)
	s.server.Close()
}

func (s *MailserverSuite) TestOpenEnvelopeWithAsymKey() {
	// Setup the server with an asymetric key
	config := *s.config
	config.MailServerPassword = "" // clear password field
	s.NoError(s.server.Init(s.shh, &config))

	// Prepare a valid envelope
	s.Require().NotNil(s.server.asymFilter)
	pubKey := s.server.asymFilter.KeyAsym.PublicKey
	env, err := generateEnvelopeWithKeys(time.Now(), nil, &pubKey)
	s.Require().NoError(err)

	// Test openEnvelope with a valid asymetric key
	d := s.server.openEnvelope(env)
	s.NotNil(d)
	s.Equal(testPayload, d.Payload)
	s.server.Close()
}

func (s *MailserverSuite) TestArchive() {
	config := *s.config
	config.MailServerAsymKey = "" // clear asym key

	err := s.server.Init(s.shh, &config)
	s.Require().NoError(err)
	defer s.server.Close()

	env, err := generateEnvelope(time.Now())
	s.NoError(err)
	rawEnvelope, err := rlp.EncodeToBytes(env)
	s.NoError(err)

	s.server.Archive(env)
	key := NewDBKey(env.Expiry-env.TTL, env.Hash())
	archivedEnvelope, err := s.server.db.Get(key.Bytes(), nil)
	s.NoError(err)

	s.Equal(rawEnvelope, archivedEnvelope)
}

func (s *MailserverSuite) TestManageLimits() {
	s.server.rateLimiter = newRateLimiter(time.Duration(5) * time.Millisecond)
	s.False(s.server.exceedsPeerRequests([]byte("peerID")))
	s.Equal(1, len(s.server.rateLimiter.db))
	firstSaved := s.server.rateLimiter.db["peerID"]

	// second call when limit is not accomplished does not store a new limit
	s.True(s.server.exceedsPeerRequests([]byte("peerID")))
	s.Equal(1, len(s.server.rateLimiter.db))
	s.Equal(firstSaved, s.server.rateLimiter.db["peerID"])
}

func (s *MailserverSuite) TestDBKey() {
	var h common.Hash
	i := uint32(time.Now().Unix())
	k := NewDBKey(i, h)
	s.Equal(len(k.Bytes()), DBKeyLength, "wrong DB key length")
	s.Equal(byte(i%0x100), k.Bytes()[3], "raw representation should be big endian")
	s.Equal(byte(i/0x1000000), k.Bytes()[0], "big endian expected")
}

func (s *MailserverSuite) TestRequestPaginationLimit() {
	s.setupServer(s.server)
	defer s.server.Close()

	var (
		sentEnvelopes     []*whisper.Envelope
		reverseSentHashes []common.Hash
		receivedHashes    []common.Hash
		archiveKeys       []string
	)

	now := time.Now()
	count := uint32(10)

	for i := count; i > 0; i-- {
		sentTime := now.Add(time.Duration(-i) * time.Second)
		env, err := generateEnvelope(sentTime)
		s.NoError(err)
		s.server.Archive(env)
		key := NewDBKey(env.Expiry-env.TTL, env.Hash())
		archiveKeys = append(archiveKeys, fmt.Sprintf("%x", key.Bytes()))
		sentEnvelopes = append(sentEnvelopes, env)
		reverseSentHashes = append([]common.Hash{env.Hash()}, reverseSentHashes...)
	}

	params := s.defaultServerParams(sentEnvelopes[0])
	params.low = uint32(now.Add(time.Duration(-count) * time.Second).Unix())
	params.upp = uint32(now.Unix())
	params.limit = 6
	request := s.createRequest(params)
	src := crypto.FromECDSAPub(&params.key.PublicKey)
	lower, upper, bloom, limit, cursor, err := s.server.validateRequest(src, request)
	s.True(err == nil)
	s.Nil(cursor)
	s.Equal(params.limit, limit)

	receivedHashes, cursor, _ = processRequestAndCollectHashes(
		s.server, lower, upper, cursor, bloom, int(limit),
	)

	// 10 envelopes sent
	s.Equal(count, uint32(len(sentEnvelopes)))
	// 6 envelopes received
	s.Equal(int(limit), len(receivedHashes))
	// the 6 envelopes received should be in descending order
	s.Equal(reverseSentHashes[:limit], receivedHashes)
	// cursor should be the key of the last envelope of the last page
	s.Equal(archiveKeys[count-limit], fmt.Sprintf("%x", cursor))

	// second page
	receivedHashes, cursor, _ = processRequestAndCollectHashes(
		s.server, lower, upper, cursor, bloom, int(limit),
	)

	// 4 envelopes received
	s.Equal(int(count-limit), len(receivedHashes))
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
			lower, upper, bloom, limit, _, err := s.server.validateRequest(src, request)
			s.Equal(tc.isOK, err == nil)
			if err == nil {
				s.Equal(tc.params.low, lower)
				s.Equal(tc.params.upp, upper)
				s.Equal(tc.params.limit, limit)
				s.Equal(whisper.TopicToBloom(tc.params.topic), bloom)
				s.Equal(tc.expect, s.messageExists(env, tc.params.low, tc.params.upp, bloom, tc.params.limit))

				src[0]++
				_, _, _, _, _, err = s.server.validateRequest(src, request)
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

	env := s.createEnvelope(whisper.TopicType{0x01}, data, srcKey)

	decodedPayload, err := s.server.decodeRequest(nil, env)
	s.Require().NoError(err)
	s.Equal(payload, decodedPayload)
}

func (s *MailserverSuite) messageExists(envelope *whisper.Envelope, low, upp uint32, bloom []byte, limit uint32) bool {
	receivedHashes, _, _ := processRequestAndCollectHashes(
		s.server, low, upp, nil, bloom, int(limit),
	)
	for _, hash := range receivedHashes {
		if hash == envelope.Hash() {
			return true
		}
	}
	return false
}

func (s *MailserverSuite) TestBloomFromReceivedMessage() {
	testCases := []struct {
		msg           whisper.ReceivedMessage
		expectedBloom []byte
		expectedErr   error
		info          string
	}{
		{
			msg:           whisper.ReceivedMessage{},
			expectedBloom: []byte(nil),
			expectedErr:   errors.New("Undersized p2p request"),
			info:          "getting bloom filter for an empty whisper message should produce an error",
		},
		{
			msg:           whisper.ReceivedMessage{Payload: []byte("hohohohoho")},
			expectedBloom: []byte(nil),
			expectedErr:   errors.New("Undersized bloom filter in p2p request"),
			info:          "getting bloom filter for a malformed whisper message should produce an error",
		},
		{
			msg:           whisper.ReceivedMessage{Payload: []byte("12345678")},
			expectedBloom: whisper.MakeFullNodeBloom(),
			expectedErr:   nil,
			info:          "getting bloom filter for a valid whisper message should be successful",
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.info, func(*testing.T) {
			bloom, err := s.server.bloomFromReceivedMessage(&tc.msg)
			s.Equal(tc.expectedErr, err)
			s.Equal(tc.expectedBloom, bloom)
		})
	}
}

func (s *MailserverSuite) setupServer(server *WMailServer) {
	const password = "password_for_this_test"

	s.shh = whisper.New(&whisper.DefaultConfig)
	s.shh.RegisterServer(server)

	err := server.Init(s.shh, &params.WhisperConfig{
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

func (s *MailserverSuite) defaultServerParams(env *whisper.Envelope) *ServerTestParams {
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
		topic: env.Topic,
		birth: birth,
		low:   birth - 1,
		upp:   birth + 1,
		limit: 0,
		key:   testPeerID,
	}
}

func (s *MailserverSuite) createRequest(p *ServerTestParams) *whisper.Envelope {
	bloom := whisper.TopicToBloom(p.topic)
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

func (s *MailserverSuite) createEnvelope(topic whisper.TopicType, data []byte, srcKey *ecdsa.PrivateKey) *whisper.Envelope {
	key, err := s.shh.GetSymKey(keyID)
	if err != nil {
		s.T().Fatalf("failed to retrieve sym key with seed %d: %s.", seed, err)
	}

	params := &whisper.MessageParams{
		KeySym:   key,
		Topic:    topic,
		Payload:  data,
		PoW:      powRequirement * 2,
		WorkTime: 2,
		Src:      srcKey,
	}

	msg, err := whisper.NewSentMessage(params)
	if err != nil {
		s.T().Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}

	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		s.T().Fatalf("failed to wrap with seed %d: %s.", seed, err)
	}
	return env
}

func generateEnvelopeWithKeys(sentTime time.Time, keySym []byte, keyAsym *ecdsa.PublicKey) (*whisper.Envelope, error) {
	params := &whisper.MessageParams{
		Topic:    whisper.TopicType{0x1F, 0x7E, 0xA1, 0x7F},
		Payload:  testPayload,
		PoW:      powRequirement,
		WorkTime: 2,
	}

	if len(keySym) > 0 {
		params.KeySym = keySym
	} else if keyAsym != nil {
		params.Dst = keyAsym
	}

	msg, err := whisper.NewSentMessage(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create new message with seed %d: %s", seed, err)
	}
	env, err := msg.Wrap(params, sentTime)
	if err != nil {
		return nil, fmt.Errorf("failed to wrap with seed %d: %s", seed, err)
	}

	return env, nil
}

func generateEnvelope(sentTime time.Time) (*whisper.Envelope, error) {
	h := crypto.Keccak256Hash([]byte("test sample data"))
	return generateEnvelopeWithKeys(sentTime, h[:], nil)
}

func processRequestAndCollectHashes(
	server *WMailServer, lower, upper uint32, cursor []byte, bloom []byte, limit int,
) ([]common.Hash, []byte, common.Hash) {
	iter := server.createIterator(lower, upper, cursor)
	defer iter.Release()
	bundles := make(chan []*whisper.Envelope, 10)
	done := make(chan struct{})

	var hashes []common.Hash
	go func() {
		for bundle := range bundles {
			for _, env := range bundle {
				hashes = append(hashes, env.Hash())
			}
		}
		close(done)
	}()

	cursor, lastHash := server.processRequestInBundles(iter, bloom, limit, bundles, done)

	<-done

	return hashes, cursor, lastHash
}

// mockPeerWithID is a struct that conforms to peerWithID interface.
type mockPeerWithID struct {
	id []byte
}

func (p mockPeerWithID) ID() []byte { return p.id }

func TestPeerIDString(t *testing.T) {
	a := []byte{0x01, 0x02, 0x03}
	require.Equal(t, "010203", peerIDString(&mockPeerWithID{a}))
	require.Equal(t, "010203", peerIDBytesString(a))
}
