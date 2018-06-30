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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/suite"
)

const powRequirement = 0.00001

var keyID string
var seed = time.Now().Unix()

type ServerTestParams struct {
	topic whisper.TopicType
	birth uint32
	low   uint32
	upp   uint32
	key   *ecdsa.PrivateKey
}

func TestMailserverSuite(t *testing.T) {
	suite.Run(t, new(MailserverSuite))
}

type MailserverSuite struct {
	suite.Suite
	server *WMailServer
	shh    *whisper.Whisper
	config *params.WhisperConfig
}

func (s *MailserverSuite) SetupTest() {
	s.server = &WMailServer{}
	s.shh = whisper.New(&whisper.DefaultConfig)
	s.shh.RegisterServer(s.server)

	tmpDir, err := ioutil.TempDir("", "mailserver-test")
	s.Require().NoError(err)

	// required files to validate mail server decryption method
	asymKeyFile := filepath.Join(tmpDir, "asymkey")
	passwordFile := filepath.Join(tmpDir, "password")
	privateKey, err := crypto.GenerateKey()
	s.Require().NoError(err)
	err = crypto.SaveECDSA(asymKeyFile, privateKey)
	s.Require().NoError(err)
	err = ioutil.WriteFile(passwordFile, []byte("testpassword"), os.ModePerm)
	s.Require().NoError(err)

	s.config = &params.WhisperConfig{
		DataDir:      tmpDir,
		AsymKeyFile:  asymKeyFile,
		PasswordFile: passwordFile,
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
			config:        params.WhisperConfig{DataDir: "/invalid-path"},
			expectedError: errors.New("open DB: mkdir /invalid-path: permission denied"),
			info:          "config with an unexisting DataDir",
		},
		{
			config: params.WhisperConfig{
				DataDir:  s.config.DataDir,
				Password: "",
				AsymKey:  nil,
			},
			expectedError: errDecryptionMethodNotProvided,
			info:          "config with an empty password and empty asym key",
		},
		{
			config:        params.WhisperConfig{DataDir: s.config.DataDir},
			expectedError: nil,
			info:          "config with correct DataDir",
		},
		{
			config: params.WhisperConfig{
				DataDir: s.config.DataDir,
				AsymKey: asymKey,
			},
			expectedError: nil,
			info:          "config with only asym key file",
		},
		{
			config: params.WhisperConfig{
				DataDir:  s.config.DataDir,
				AsymKey:  asymKey,
				Password: "pwd",
			},
			expectedError: nil,
			info:          "config with both asym key and password",
		},
		{
			config: params.WhisperConfig{
				DataDir:             s.config.DataDir,
				MailServerRateLimit: 5,
			},
			expectedError: nil,
			info:          "config witih rate limit",
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

			s.NotNil(mailServer.db)
			s.NotNil(mailServer.filter)
			if tc.config.MailServerRateLimit > 0 {
				s.NotNil(mailServer.limit)
			}
		})
	}
}

func (s *MailserverSuite) TestSetupRequestMessageDecryptor() {
	var config params.WhisperConfig

	// without configured Password and AsymKey
	config = *s.config
	s.Error(errDecryptionMethodNotProvided, s.server.Init(s.shh, &config))

	// Password should work ok
	config = *s.config
	s.NoError(config.ReadPasswordFile())
	s.NoError(s.server.Init(s.shh, &config))
	s.NotNil(s.server.filter.KeySym)
	s.Nil(s.server.filter.KeyAsym)
	s.server.Close()

	// AsymKey can also be used
	config = *s.config
	s.NoError(config.ReadAsymKeyFile())
	s.NoError(s.server.Init(s.shh, &config))
	s.Nil(s.server.filter.KeySym) // important: symmetric key should be nil
	s.Equal(config.AsymKey, s.server.filter.KeyAsym)
	s.server.Close()

	// when both Password and AsymKey are set, Password has a preference
	config = *s.config
	s.NoError(config.ReadPasswordFile())
	s.NoError(config.ReadAsymKeyFile())
	s.NoError(s.server.Init(s.shh, &config))
	s.NotNil(s.server.filter.KeySym)
	s.Nil(s.server.filter.KeyAsym)
	s.server.Close()
}

func (s *MailserverSuite) TestArchive() {
	err := s.server.Init(s.shh, s.config)
	s.server.tick = nil
	s.NoError(err)
	defer s.server.Close()

	env, err := generateEnvelope(time.Now())
	s.NoError(err)
	rawEnvelope, err := rlp.EncodeToBytes(env)
	s.NoError(err)

	s.server.Archive(env)
	key := NewDbKey(env.Expiry-env.TTL, env.Hash())
	archivedEnvelope, err := s.server.db.Get(key.raw, nil)
	s.NoError(err)

	s.Equal(rawEnvelope, archivedEnvelope)
}

func (s *MailserverSuite) TestManageLimits() {
	s.server.limit = newLimiter(time.Duration(5) * time.Millisecond)
	s.False(s.server.exceedsPeerRequests([]byte("peerID")))
	s.Equal(1, len(s.server.limit.db))
	firstSaved := s.server.limit.db["peerID"]

	// second call when limit is not accomplished does not store a new limit
	s.True(s.server.exceedsPeerRequests([]byte("peerID")))
	s.Equal(1, len(s.server.limit.db))
	s.Equal(firstSaved, s.server.limit.db["peerID"])
}

func (s *MailserverSuite) TestDBKey() {
	var h common.Hash
	i := uint32(time.Now().Unix())
	k := NewDbKey(i, h)
	s.Equal(len(k.raw), common.HashLength+4, "wrong DB key length")
	s.Equal(byte(i%0x100), k.raw[3], "raw representation should be big endian")
	s.Equal(byte(i/0x1000000), k.raw[0], "big endian expected")
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
			ok, lower, upper, bloom := s.server.validateRequest(src, request)
			s.Equal(tc.isOK, ok)
			if ok {
				s.Equal(tc.params.low, lower)
				s.Equal(tc.params.upp, upper)
				s.Equal(whisper.TopicToBloom(tc.params.topic), bloom)
				s.Equal(tc.expect, s.messageExists(env, tc.params.low, tc.params.upp, bloom))

				src[0]++
				ok, _, _, _ = s.server.validateRequest(src, request)
				s.True(ok)
			}
		})
	}
}

func (s *MailserverSuite) messageExists(envelope *whisper.Envelope, low, upp uint32, bloom []byte) bool {
	var exist bool
	mail, err := s.server.processRequest(nil, low, upp, bloom)
	s.NoError(err)
	for _, msg := range mail {
		if msg.Hash() == envelope.Hash() {
			exist = true
			break
		}
	}
	return exist
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
	const dbPath = "whisper-server-test"

	dir, err := ioutil.TempDir("", dbPath)
	if err != nil {
		s.T().Fatal(err)
	}

	s.shh = whisper.New(&whisper.DefaultConfig)
	s.shh.RegisterServer(server)

	err = server.Init(s.shh, &params.WhisperConfig{DataDir: dir, Password: password, MinimumPoW: powRequirement})
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
		key:   testPeerID,
	}
}

func (s *MailserverSuite) createRequest(p *ServerTestParams) *whisper.Envelope {
	bloom := whisper.TopicToBloom(p.topic)
	data := make([]byte, 8)
	binary.BigEndian.PutUint32(data, p.low)
	binary.BigEndian.PutUint32(data[4:], p.upp)
	data = append(data, bloom...)

	key, err := s.shh.GetSymKey(keyID)
	if err != nil {
		s.T().Fatalf("failed to retrieve sym key with seed %d: %s.", seed, err)
	}

	params := &whisper.MessageParams{
		KeySym:   key,
		Topic:    p.topic,
		Payload:  data,
		PoW:      powRequirement * 2,
		WorkTime: 2,
		Src:      p.key,
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

func generateEnvelope(sentTime time.Time) (*whisper.Envelope, error) {
	h := crypto.Keccak256Hash([]byte("test sample data"))
	params := &whisper.MessageParams{
		KeySym:   h[:],
		Topic:    whisper.TopicType{0x1F, 0x7E, 0xA1, 0x7F},
		Payload:  []byte("test payload"),
		PoW:      powRequirement,
		WorkTime: 2,
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
