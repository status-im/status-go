package api_test

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/signal"
	. "github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/storage"
)

const (
	testChatID = "testChat"
)

func TestAPI(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

type APITestSuite struct {
	suite.Suite
	api *api.StatusAPI
}

func (s *APITestSuite) ensureNodeStopped() {
	if err := s.api.StopNode(); err != node.ErrNoRunningNode && err != nil {
		s.NoError(err, "unexpected error")
	}
}

func (s *APITestSuite) SetupTest() {
	s.api = api.NewStatusAPI()
	s.NotNil(s.api)
}

func (s *APITestSuite) TestCHTUpdate() {
	tmpDir, err := ioutil.TempDir(os.TempDir(), "cht-updates")
	s.NoError(err)
	defer os.RemoveAll(tmpDir) //nolint: errcheck

	configJSON := `{
		"NetworkId": ` + strconv.Itoa(params.RopstenNetworkID) + `,
		"DataDir": "` + tmpDir + `",
		"LogLevel": "INFO",
		"RPCEnabled": true
	}`

	_, err = params.LoadNodeConfig(configJSON)
	s.NoError(err)
	// TODO(tiabc): Test that CHT is really updated.
}

func (s *APITestSuite) TestRaceConditions() {
	cnt := 25
	progress := make(chan struct{}, cnt)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	nodeConfig1, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	nodeConfig2, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	nodeConfigs := []*params.NodeConfig{nodeConfig1, nodeConfig2}

	var funcsToTest = []func(*params.NodeConfig){
		func(config *params.NodeConfig) {
			log.Info("StartNodeAsync()")
			s.api.StartNodeAsync(config)
			s.T().Logf("StartNodeAsync() for network: %d", config.NetworkID)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("StopNodeAsync()")
			s.api.StopNodeAsync()
			s.T().Logf("StopNodeAsync()")
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("RestartNodeAsync()")
			s.api.RestartNodeAsync()
			s.T().Logf("RestartNodeAsync()")
			progress <- struct{}{}
		},
		// TODO(adam): quarantined until it uses a different datadir
		// as otherwise it wipes out cached blockchain data.
		// func(config *params.NodeConfig) {
		//	log.Info("ResetChainDataAsync()")
		//	_, err := s.api.ResetChainDataAsync()
		//	s.T().Logf("ResetChainDataAsync(), error: %v", err)
		//	progress <- struct{}{}
		// },
	}

	// increase StartNode()/StopNode() population
	for i := 0; i < 5; i++ {
		funcsToTest = append(funcsToTest, funcsToTest[0], funcsToTest[1])
	}

	for i := 0; i < cnt; i++ {
		randConfig := nodeConfigs[rnd.Intn(len(nodeConfigs))]
		randFunc := funcsToTest[rnd.Intn(len(funcsToTest))]

		if rnd.Intn(100) > 75 { // introduce random delays
			time.Sleep(500 * time.Millisecond)
		}
		go randFunc(randConfig)
	}

	for range progress {
		cnt--
		if cnt <= 0 {
			break
		}
	}

	time.Sleep(2 * time.Second) // so that we see some logs
	// just in case we have a node running
	s.ensureNodeStopped()
}

func (s *APITestSuite) TestCellsRemovedAfterSwitchAccount() {
	const itersCount = 5
	var (
		require   = s.Require()
		getChatID = func(id int) string {
			return testChatID + strconv.Itoa(id)
		}
	)

	config, err := MakeTestNodeConfig(GetNetworkID())
	require.NoError(err)
	err = s.api.StartNode(config)
	require.NoError(err)
	defer s.api.StopNode() //nolint: errcheck

	address1, _, _, err := s.api.AccountManager().CreateAccount(TestConfig.Account1.Password)
	require.NoError(err)

	address2, _, _, err := s.api.AccountManager().CreateAccount(TestConfig.Account2.Password)
	require.NoError(err)

	err = s.api.SelectAccount(address1, TestConfig.Account1.Password)
	require.NoError(err)

	for i := 0; i < itersCount; i++ {
		_, e := s.api.JailManager().CreateCell(getChatID(i))
		require.NoError(e)
	}

	err = s.api.SelectAccount(address2, TestConfig.Account2.Password)
	require.NoError(err)

	for i := 0; i < itersCount; i++ {
		_, e := s.api.JailManager().Cell(getChatID(i))
		require.Error(e)
	}
}

// TestLogoutRemovesCells we want be sure that
// cells will be removed after the API call "Logout"
func (s *APITestSuite) TestLogoutRemovesCells() {
	var (
		err     error
		require = s.Require()
	)

	config, err := MakeTestNodeConfig(GetNetworkID())
	require.NoError(err)
	err = s.api.StartNode(config)
	require.NoError(err)
	defer s.api.StopNode() //nolint: errcheck

	address1, _, _, err := s.api.AccountManager().CreateAccount(TestConfig.Account1.Password)
	require.NoError(err)

	err = s.api.SelectAccount(address1, TestConfig.Account1.Password)
	require.NoError(err)

	s.api.JailManager().CreateAndInitCell(testChatID)

	err = s.api.Logout()
	require.NoError(err)

	_, err = s.api.JailManager().Cell(testChatID)
	require.Error(err, "Expected that cells was removed")
}

func (s *APITestSuite) TestEventsNodeStartStop() {
	envelopes := make(chan signal.Envelope, 3)
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err)
		// whitelist types that we are interested in
		switch envelope.Type {
		case signal.EventNodeStarted:
		case signal.EventNodeStopped:
		case signal.EventNodeReady:
		default:
			return
		}
		envelopes <- envelope
	})

	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)
	s.NoError(s.api.StartNode(nodeConfig))
	s.NoError(s.api.StopNode())
	s.verifyEnvelopes(envelopes, signal.EventNodeStarted, signal.EventNodeReady, signal.EventNodeStopped)
	s.NoError(s.api.StartNode(nodeConfig))
	s.verifyEnvelopes(envelopes, signal.EventNodeStarted, signal.EventNodeReady)
	s.NoError(s.api.RestartNode())
	s.verifyEnvelopes(envelopes, signal.EventNodeStopped, signal.EventNodeStarted, signal.EventNodeReady)
	s.NoError(s.api.StopNode())
	s.verifyEnvelopes(envelopes, signal.EventNodeStopped)
}

func (s *APITestSuite) verifyEnvelopes(envelopes chan signal.Envelope, envelopeTypes ...string) {
	for _, envelopeType := range envelopeTypes {
		select {
		case env := <-envelopes:
			s.Equal(envelopeType, env.Type)
		case <-time.After(1 * time.Second):
			s.Fail("timeout waiting for envelope")
		}
	}
}

func (s *APITestSuite) TestNodeStartCrash() {
	// let's listen for node.crashed signal
	signalReceived := make(chan struct{})
	signal.SetDefaultNodeNotificationHandler(func(jsonEvent string) {
		var envelope signal.Envelope
		err := json.Unmarshal([]byte(jsonEvent), &envelope)
		s.NoError(err)

		if envelope.Type == signal.EventNodeCrashed {
			close(signalReceived)
		}
	})
	defer signal.ResetDefaultNodeNotificationHandler()

	nodeConfig, err := MakeTestNodeConfig(GetNetworkID())
	s.NoError(err)

	db, err := leveldb.Open(storage.NewMemStorage(), nil)
	s.NoError(err)
	defer func() { s.NoError(db.Close()) }()

	// start node outside the manager (on the same port), so that manager node.Start() method fails
	outsideNode, err := node.MakeNode(nodeConfig, db)
	s.NoError(err)
	err = outsideNode.Start()
	s.NoError(err)

	// now try starting using node manager, it should fail (error is irrelevant as it is implementation detail)
	s.Error(<-s.api.StartNodeAsync(nodeConfig))

	select {
	case <-time.After(500 * time.Millisecond):
		s.FailNow("timed out waiting for signal")
	case <-signalReceived:
	}

	// stop outside node, and re-try
	s.NoError(outsideNode.Stop())
	signalReceived = make(chan struct{})
	s.NoError(<-s.api.StartNodeAsync(nodeConfig))

	select {
	case <-time.After(500 * time.Millisecond):
	case <-signalReceived:
		s.FailNow("signal should not be received")
	}

	// cleanup
	s.NoError(s.api.StopNode())
}
