package whisper

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/common"
	"github.com/status-im/status-go/static"
	e2e "github.com/status-im/status-go/t/e2e"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

const (
	//nolint: unused, varcheck
	whisperMessage1 = `test message 1 (K1 -> K2, signed+encrypted, from us)`
	whisperMessage2 = `test message 3 (K1 -> "", signed broadcast)`
	whisperMessage3 = `test message 4 ("" -> "", anon broadcast)`
	whisperMessage4 = `test message 5 ("" -> K1, encrypted anon broadcast)`
	whisperMessage5 = `test message 6 (K2 -> K1, signed+encrypted, to us)`
)

var (
	baseStatusJSCode = string(static.MustAsset("testdata/jail/status.js"))
)

func TestWhisperJailTestSuite(t *testing.T) {
	suite.Run(t, new(WhisperJailTestSuite))
}

type WhisperJailTestSuite struct {
	e2e.BackendTestSuite

	Timeout    time.Duration
	WhisperAPI *whisper.PublicWhisperAPI
	Jail       common.JailManager
}

func (s *WhisperJailTestSuite) StartTestBackend(opts ...e2e.TestNodeOption) {
	s.BackendTestSuite.StartTestBackend(opts...)

	s.Timeout = time.Minute * 5
	s.WhisperAPI = whisper.NewPublicWhisperAPI(s.WhisperService())
	s.Jail = s.Backend.JailManager()
	s.NotNil(s.Jail)
	s.Jail.SetBaseJS(baseStatusJSCode)
}

func (s *WhisperJailTestSuite) AddKeyPair(address, password string) (string, error) {
	accountManager := s.Backend.AccountManager()

	_, accountKey, err := accountManager.AddressToDecryptedAccount(address, password)
	if err != nil {
		return "", err
	}

	return s.WhisperService().AddKeyPair(accountKey.PrivateKey)
}

func (s *WhisperJailTestSuite) TestJailWhisper() {
	s.StartTestBackend()
	defer s.StopTestBackend()

	keyPairID1, err := s.AddKeyPair(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)

	keyPairID2, err := s.AddKeyPair(TestConfig.Account2.Address, TestConfig.Account2.Password)
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
				var identity1 = '` + keyPairID1 + `';
				if (!shh.hasKeyPair(identity1)) {
					throw 'identity "` + keyPairID1 + `" not found in whisper';
				}

				var identity2 = '` + keyPairID2 + `';
				if (!shh.hasKeyPair(identity2)) {
					throw 'identitity "` + keyPairID2 + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage1 + `';

				// start watching for messages
				var filter = shh.newMessageFilter({
					sig: shh.getPublicKey(identity1),
					privateKeyID: identity2,
					topics: [topic]
				});

				// post message
				var message = {
					ttl: 10,
					powTarget: 1.0,
					powTime: 20,
					topic: topic,
					sig: shh.getPublicKey(identity1),
					pubKey: shh.getPublicKey(identity2),
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
				var identity = '` + keyPairID1 + `';
				if (!shh.hasKeyPair(identity)) {
					throw 'identity "` + keyPairID1 + `" not found in whisper';
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
					sig: shh.getPublicKey(identity),
					topics: [topic],
					symKeyID: keyid
				});

				// post message
				var message = {
					ttl: 10,
					powTarget: 1.0,
					powTime: 20,
					topic: topic,
					sig: shh.getPublicKey(identity),
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
					ttl: 10,
					powTarget: 1.0,
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
		{
			"test 4: encrypted anonymous message (From == nil && To != nil)",
			`
				var identity = '` + keyPairID1 + `';
				if (!shh.hasKeyPair(identity)) {
					throw 'identity "` + keyPairID1 + `" not found in whisper';
				}

				var topic = makeTopic();
				var payload = '` + whisperMessage4 + `';

				// start watching for messages
				var filter = shh.newMessageFilter({
					privateKeyID: identity,
					topics: [topic],
				});

				// post message
				var message = {
					ttl: 20,
					powTarget: 0.01,
					powTime: 20,
					topic: topic,
					pubKey: shh.getPublicKey(identity),
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
			"test 5: encrypted signed response to us (From != nil && To != nil)",
			`
				var identity1 = '` + keyPairID1 + `';
				if (!shh.hasKeyPair(identity1)) {
					throw 'identity "` + keyPairID1 + `" not found in whisper';
				}
				var identity2 = '` + keyPairID2 + `';
				if (!shh.hasKeyPair(identity2)) {
					throw 'identity "` + keyPairID2 + `" not found in whisper';
				}
				var topic = makeTopic();
				var payload = '` + whisperMessage5 + `';
				// start watching for messages
				var filter = shh.newMessageFilter({
					privateKeyID: identity1,
					sig: shh.getPublicKey(identity2),
					topics: [topic],
				});

				// post message
				var message = {
					ttl: 10,
					powTarget: 1.0,
					powTime: 20,
				  	sig: shh.getPublicKey(identity2),
				  	pubKey: shh.getPublicKey(identity1),
				  	topic: topic,
				  	payload: web3.toHex(payload)
				};

				var sent = shh.post(message)
				if (!sent) {
					throw 'message not sent: ' + message;
				}
			`,
			true,
		},
	}

	makeTopicCode := `
		var shh = web3.shh;
		// topic must be 4-byte long
		var makeTopic = function () {
			var topic = '0x';
			for (var i = 0; i < 8; i++) {
				topic += Math.floor(Math.random() * 16).toString(16);
			}
			return topic;
		};
	`

	for _, tc := range testCases {
		chatID := crypto.Keccak256Hash([]byte(tc.name)).Hex()

		s.Jail.CreateAndInitCell(chatID, makeTopicCode)

		cell, err := s.Jail.Cell(chatID)
		s.NoError(err, "cannot get VM")

		// Run JS code that setups filters and sends messages.
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

		// Use polling because:
		//   (1) filterId is not assigned immediately,
		//   (2) messages propagate with some delay.
	poll_loop:
		for {
			filter, err := cell.Get("filter")
			s.NoError(err, "cannot get filter")
			filterID, err := filter.Object().Get("filterId")
			s.NoError(err, "cannot get filterId")

			select {
			case <-done:
				ok, err := s.WhisperAPI.DeleteMessageFilter(filterID.String())
				s.NoError(err)
				s.True(ok)
				break poll_loop
			case <-timedOut:
				s.FailNow("polling for messages timed out")
			case <-time.After(time.Second):
			}

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
