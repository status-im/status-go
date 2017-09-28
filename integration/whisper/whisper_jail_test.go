package whisper

import (
	"context"
	"testing"
	"time"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/params"
	"github.com/status-im/status-go/integration"
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
	suite.Run(t, new(WhisperJailTestSuite))
}

type WhisperJailTestSuite struct {
	integration.BackendTestSuite
}

func (s *WhisperJailTestSuite) TestJailWhisper() {
	s.StartTestBackend(params.RopstenNetworkID)
	defer s.StopTestBackend()

	jail := s.Backend.JailManager()
	s.NotNil(jail)

	jail.BaseJS(baseStatusJSCode)

	whisperService := s.WhisperService()
	whisperAPI := whisper.NewPublicWhisperAPI(whisperService)
	accountManager := s.Backend.AccountManager()

	// account1
	_, accountKey1, err := accountManager.AddressToDecryptedAccount(TestConfig.Account1.Address, TestConfig.Account1.Password)
	s.NoError(err)
	accountKey1Hex := gethcommon.ToHex(crypto.FromECDSAPub(&accountKey1.PrivateKey.PublicKey))

	_, err = whisperService.AddKeyPair(accountKey1.PrivateKey)
	s.NoError(err, "identity not injected: %v", accountKey1Hex)

	if ok := whisperAPI.HasKeyPair(context.Background(), accountKey1Hex); !ok {
		s.FailNow("identity not injected: %v", accountKey1Hex)
	}

	// account2
	_, accountKey2, err := accountManager.AddressToDecryptedAccount(TestConfig.Account2.Address, TestConfig.Account2.Password)
	s.NoError(err)
	accountKey2Hex := gethcommon.ToHex(crypto.FromECDSAPub(&accountKey2.PrivateKey.PublicKey))

	_, err = whisperService.AddKeyPair(accountKey2.PrivateKey)
	s.NoError(err, "identity not injected: %v", accountKey2Hex)

	if ok := whisperAPI.HasKeyPair(context.Background(), accountKey2Hex); !ok {
		s.FailNow("identity not injected: %v", accountKey2Hex)
	}

	passedTests := map[string]bool{
		whisperMessage1: false,
		whisperMessage2: false,
		whisperMessage3: false,
		whisperMessage4: false,
		whisperMessage5: false,
	}
	installedFilters := map[string]string{
		whisperMessage1: "",
		whisperMessage2: "",
		whisperMessage3: "",
		whisperMessage4: "",
		whisperMessage5: "",
	}

	testCases := []struct {
		name      string
		testCode  string
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

				var filterName = '` + whisperMessage1 + `';
				var filterId = filter.filterId;
				if (!filterId) {
					throw 'filter not installed properly';
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

				var filterName = '` + whisperMessage2 + `';
				var filterId = filter.filterId;
				if (!filterId) {
					throw 'filter not installed properly';
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

				var filterName = '` + whisperMessage3 + `';
				var filterId = filter.filterId;
				if (!filterId) {
					throw 'filter not installed properly';
				}
			`,
			true,
		},
		// @TODO(adam): fix in #336
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

		// 		var filterName = '` + whisperMessage4 + `';
		// 		var filterId = filter.filterId;
		// 		if (!filterId) {
		// 			throw 'filter not installed properly';
		// 		}
		// 	`,
		// 	true,
		// },
		// {
		// 	"test 5: encrypted signed response to us (From != nil && To != nil)",
		// 	`
		// 		var identity1 = '` + accountKey1Hex + `';
		// 		if (!shh.hasKeyPair(identity1)) {
		// 			throw 'idenitity "` + accountKey1Hex + `" not found in whisper';
		// 		}
		// 		var identity2 = '` + accountKey2Hex + `';
		// 		if (!shh.hasKeyPair(identity2)) {
		// 			throw 'idenitity "` + accountKey2Hex + `" not found in whisper';
		// 		}
		// 		var topic = makeTopic();
		// 		var payload = '` + whisperMessage5 + `';
		// 		// start watching for messages
		// 		var filter = shh.newMessageFilter({
		// 			privateKeyID: identity1,
		// 			sig: identity2,
		// 			topics: [topic],
		// 		});

		// 		// post message
		// 		var message = {
		// 		  	sig: identity2,
		// 		  	pubKey: identity1,
		// 		  	topic: topic,
		// 		  	payload: web3.toHex(payload),
		// 			ttl: 20,
		// 			powTime: 20,
		// 			powTarget: 0.01,
		// 		};
		// 		var sent = shh.post(message)
		// 		if (!sent) {
		// 			throw 'message not sent: ' + message;
		// 		}
		// 		var filterName = '` + whisperMessage5 + `';
		// 		var filterId = filter.filterId;
		// 		if (!filterId) {
		// 			throw 'filter not installed properly';
		// 		}
		// 	`,
		// 	true,
		// },
	}

	for _, testCase := range testCases {
		s.T().Log(testCase.name)
		testCaseKey := crypto.Keccak256Hash([]byte(testCase.name)).Hex()

		jail.Parse(testCaseKey, `
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

		cell, err := jail.Cell(testCaseKey)
		s.NoError(err, "cannot get VM")

		// post messages
		_, err = cell.Run(testCase.testCode)
		s.NoError(err)

		if !testCase.useFilter {
			continue
		}

		// update installed filters
		filterID, err := cell.Get("filterId")
		s.NoError(err, "cannot get filterId")

		filterName, err := cell.Get("filterName")
		s.NoError(err, "cannot get filterName")

		_, ok := installedFilters[filterName.String()]
		s.True(ok, "unrecognized filter")

		installedFilters[filterName.String()] = filterID.String()
	}

	time.Sleep(2 * time.Second) // allow whisper to poll

	for testKey, filter := range installedFilters {
		if filter != "" {
			s.T().Logf("filter found: %v", filter)
			messages, err := whisperAPI.GetFilterMessages(filter)
			s.NoError(err)
			for _, message := range messages {
				s.T().Logf("message found: %s", string(message.Payload))
				passedTests[testKey] = true
			}
		}
	}

	// TODO(adam): qurantined because test case 4th and 5th are commented out
	// for testName, passedTest := range passedTests {
	// 	s.True(passedTest, "test not passed: %v", testName)
	// }
}
