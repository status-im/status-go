package whisper

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/e2e"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/static"
	. "github.com/status-im/status-go/testing"
	"github.com/stretchr/testify/suite"
)

const (
	whisperMessage1 = `test message 1 (K1 -> K2, signed+encrypted, from us)`
	whisperMessage2 = `test message 3 (K1 -> "", signed broadcast)`
	whisperMessage3 = `test message 4 ("" -> "", anon broadcast)`
	whisperMessage4 = `test message 5 ("" -> K1, encrypted anon broadcast)`
	whisperMessage5 = `test message 6 (K2 -> K1, signed+encrypted, to us)`
	testChatID      = "testChat"
)

var (
	baseStatusJSCode = string(static.MustAsset("testdata/jail/status.js"))
)

func TestWhisperJailTestSuite(t *testing.T) {
	s := new(WhisperJailTestSuite)
	s.Timeout = time.Minute * 5
	suite.Run(t, s)
}

type WhisperJailTestSuite struct {
	e2e.BackendTestSuite

	Timeout    time.Duration
	WhisperAPI *whisper.PublicWhisperAPI
	Jail       common.JailManager
}

func (s *WhisperJailTestSuite) StartTestBackend(networkID int, opts ...e2e.TestNodeOption) {
	s.BackendTestSuite.StartTestBackend(networkID, opts...)

	s.WhisperAPI = whisper.NewPublicWhisperAPI(s.WhisperService())
	s.Jail = s.Backend.JailManager()
	s.NotNil(s.Jail)
	s.Jail.BaseJS(baseStatusJSCode)
}

func (s *WhisperJailTestSuite) GetAccountKey(account struct {
	Address  string
	Password string
}) (*keystore.Key, string, error) {
	accountManager := s.Backend.AccountManager()

	_, accountKey1, err := accountManager.AddressToDecryptedAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	if err != nil {
		return nil, "", err
	}
	accountKey1Hex := gethcommon.ToHex(crypto.FromECDSAPub(&accountKey1.PrivateKey.PublicKey))

	_, err = s.WhisperService().AddKeyPair(accountKey1.PrivateKey)
	if err != nil {
		return nil, "", err
	}

	if ok := s.WhisperAPI.HasKeyPair(context.Background(), accountKey1Hex); !ok {
		return nil, "", errors.New("KeyPair should be injected in Whisper")
	}

	return accountKey1, accountKey1Hex, nil
}

func (s *WhisperJailTestSuite) TestJailWhisper() {
	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	_, accountKey1Hex, err := s.GetAccountKey(TestConfig.Account1)
	s.NoError(err)

	_, accountKey2Hex, err := s.GetAccountKey(TestConfig.Account2)
	s.NoError(err)

	testCases := []struct {
		name      string
		code      string
		useFilter bool
	}{
		{
			"test 0: ensure correct version of Whisper is used",
			`
				var expectedVersion = '5.0';
				if (web3.version.whisper != expectedVersion) {
					throw 'unexpected shh version, expected: ' + expectedVersion + ', got: ' + web3.version.whisper;
				}
			`,
			false,
		},
		{
			"test 1: encrypted signed message from us (From != nil && To != nil)",
			`
				var identity1 = '` + accountKey1Hex + `';
				if (!shh.hasKeyPair(identity1)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}

				var identity2 = '` + accountKey2Hex + `';
				if (!shh.hasKeyPair(identity2)) {
					throw 'identitity "` + accountKey2Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage1 + `';

				// start watching for messages
				var filter = shh.newMessageFilter({
					sig: identity1,
					privateKeyID: identity2,
					topics: [topic]
				});

				// post message
				var message = {
					ttl: 20,
					powTarget: 0.01,
					powTime: 20,
					topic: topic,
					sig: identity1,
					pubKey: identity2,
			  		payload: web3.toHex(payload),
				};

				var sent = shh.post(message)
				if (!sent) {
					throw 'message not sent: ' + JSON.stringify(message);
				}
			`,
			true,
		},
		{
			"test 2: signed (known sender) broadcast (From != nil && To == nil)",
			`
				var identity = '` + accountKey1Hex + `';
				if (!shh.hasKeyPair(identity)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage2 + `';

				// generate symmetric key
				var keyid = shh.newSymKey();
				if (!shh.hasSymKey(keyid)) {
					throw new Error('key not found');
				}

				// start watching for messages
				var filter = shh.newMessageFilter({
					sig: identity,
					topics: [topic],
					symKeyID: keyid
				});

				// post message
				var message = {
					ttl: 20,
					powTarget: 0.01,
					powTime: 20,
					topic: topic,
					sig: identity,
					symKeyID: keyid,
			  		payload: web3.toHex(payload),
				};

				var sent = shh.post(message)
				if (!sent) {
					throw 'message not sent: ' + JSON.stringify(message);
				}
			`,
			true,
		},
		{
			"test 3: anonymous broadcast (From == nil && To == nil)",
			`
				var topic = makeTopic();
				var payload = '` + whisperMessage3 + `';

				// generate symmetric key
				var keyid = shh.newSymKey();
				if (!shh.hasSymKey(keyid)) {
					throw new Error('key not found');
				}

				// start watching for messages
				var filter = shh.newMessageFilter({
					topics: [topic],
					symKeyID: keyid
				});

				// post message
				var message = {
					ttl: 20,
					powTarget: 0.01,
					powTime: 20,
					topic: topic,
					symKeyID: keyid,
			  		payload: web3.toHex(payload),
				};

				var sent = shh.post(message)
				if (!sent) {
					throw 'message not sent: ' + JSON.stringify(message);
				}
			`,
			true,
		},
		// @TODO(adam): quarantined as always failing. Check out TestEncryptedAnonymousMessage
		// as an equivalent test in pure Go which passes. Bug in web3?
		// {
		// 	"test 4: encrypted anonymous message (From == nil && To != nil)",
		// 	`
		// 		var identity = '` + accountKey2Hex + `';
		// 		if (!shh.hasKeyPair(identity)) {
		// 			throw 'idenitity "` + accountKey2Hex + `" not found in whisper';
		// 		}

		// 		var topic = makeTopic();
		// 		var payload = '` + whisperMessage4 + `';

		// 		// start watching for messages
		// 		var filter = shh.newMessageFilter({
		// 			privateKeyID: identity,
		// 			topics: [topic],
		// 		});

		// 		// post message
		// 		var message = {
		// 			ttl: 20,
		// 			powTarget: 0.01,
		// 			powTime: 20,
		// 			topic: topic,
		// 			pubKey: identity,
		// 	  		payload: web3.toHex(payload),
		// 		};

		// 		var sent = shh.post(message)
		// 		if (!sent) {
		// 			throw 'message not sent: ' + JSON.stringify(message);
		// 		}
		// 	`,
		// 	true,
		// },
		{
			"test 5: encrypted signed response to us (From != nil && To != nil)",
			`
				var identity1 = '` + accountKey1Hex + `';
				if (!shh.hasKeyPair(identity1)) {
					throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
				}
				var identity2 = '` + accountKey2Hex + `';
				if (!shh.hasKeyPair(identity2)) {
					throw 'idenitity "` + accountKey2Hex + `" not found in whisper';
				}
				var topic = makeTopic();
				var payload = '` + whisperMessage5 + `';
				// start watching for messages
				var filter = shh.newMessageFilter({
					privateKeyID: identity1,
					sig: identity2,
					topics: [topic],
				});

				// post message
				var message = {
				  	sig: identity2,
				  	pubKey: identity1,
				  	topic: topic,
				  	payload: web3.toHex(payload),
					ttl: 20,
					powTime: 20,
					powTarget: 0.01,
				};

				var sent = shh.post(message)
				if (!sent) {
					throw 'message not sent: ' + message;
				}
			`,
			true,
		},
	}

	for _, tc := range testCases {
		s.T().Log(tc.name)

		chatID := crypto.Keccak256Hash([]byte(tc.name)).Hex()

		s.Jail.Parse(chatID, `
			var shh = web3.shh;
			// topic must be 4-byte long
			var makeTopic = function () {
				var topic = '0x';
				for (var i = 0; i < 8; i++) {
					topic += Math.floor(Math.random() * 16).toString(16);
				}
				return topic;
			};
		`)

		cell, err := s.Jail.Cell(chatID)
		s.NoError(err, "cannot get VM")

		// Setup filters and post messages.
		_, err = cell.Run(tc.code)
		s.NoError(err)

		if !tc.useFilter {
			continue
		}

		done := make(chan struct{})
		timedOut := make(chan struct{})
		go func() {
			select {
			case <-done:
			case <-time.After(s.Timeout):
				close(timedOut)
			}
		}()

	poll_loop:
		for {
			// Use polling because:
			//   (1) filterId is not assigned immediately,
			//   (2) messages propagate with some delay.
			select {
			case <-done:
				break poll_loop
			case <-timedOut:
				s.FailNow("polling for messages timed out")
			case <-time.After(time.Second):
			}

			filter, err := cell.Get("filter")
			s.NoError(err, "cannot get filter")
			filterID, err := filter.Object().Get("filterId")
			s.NoError(err, "cannot get filterId")

			// FilterID is not assigned yet.
			if filterID.IsNull() {
				continue
			}

			payload, err := cell.Get("payload")
			s.NoError(err, "cannot get payload")

			messages, err := s.WhisperAPI.GetFilterMessages(filterID.String())
			s.NoError(err)
			for _, m := range messages {
				s.Equal(payload.String(), string(m.Payload))
				close(done)
			}
		}
	}
}

func (s *WhisperJailTestSuite) TestEncryptedAnonymousMessage() {
	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	accountKey2, accountKey2Hex, err := s.GetAccountKey(TestConfig.Account2)
	s.NoError(err)

	topicSlice := make([]byte, whisper.TopicLength)
	_, err = rand.Read(topicSlice)
	s.NoError(err)

	topic := whisper.BytesToTopic(topicSlice)

	filter, err := s.WhisperAPI.NewMessageFilter(whisper.Criteria{
		PrivateKeyID: accountKey2Hex,
		Topics:       []whisper.TopicType{topic},
	})
	s.NoError(err)

	ok, err := s.WhisperAPI.Post(context.Background(), whisper.NewMessage{
		TTL:       20,
		PowTarget: 0.01,
		PowTime:   20,
		Topic:     topic,
		PublicKey: crypto.FromECDSAPub(&accountKey2.PrivateKey.PublicKey),
		Payload:   []byte(whisperMessage4),
	})
	s.NoError(err)
	s.True(ok)

	done := make(chan struct{})
	timedOut := make(chan struct{})
	go func() {
		select {
		case <-done:
		case <-time.After(s.Timeout):
			close(timedOut)
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-timedOut:
			s.FailNow("polling for messages timed out")
		case <-time.After(time.Second):
		}

		messages, err := s.WhisperAPI.GetFilterMessages(filter)
		s.NoError(err)
		for _, m := range messages {
			s.Equal(whisperMessage4, string(m.Payload))
			close(done)
		}
	}
}
