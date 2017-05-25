package api_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/geth/api"
	"github.com/status-im/status-go/geth/params"
	. "github.com/status-im/status-go/geth/testing"
	"github.com/stretchr/testify/suite"
)

func TestAPI(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

type APITestSuite struct {
	suite.Suite
	api *api.StatusAPI
}

func (s *APITestSuite) SetupTest() {
	require := s.Require()
	statusAPI := api.NewStatusAPI()
	require.NotNil(statusAPI)
	require.IsType(&api.StatusAPI{}, statusAPI)
	s.api = statusAPI
}
func (s *APITestSuite) TestRaceConditions() {
	require := s.Require()
	require.NotNil(s.api)

	cnt := 25
	progress := make(chan struct{}, cnt)
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	nodeConfig1, err := MakeTestNodeConfig(params.RopstenNetworkID)
	require.NoError(err)

	nodeConfig2, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	nodeConfigs := []*params.NodeConfig{nodeConfig1, nodeConfig2}

	var funcsToTest = []func(*params.NodeConfig){
		func(config *params.NodeConfig) {
			log.Info("StartNodeAsync()")
			_, err := s.api.StartNodeAsync(config)
			s.T().Logf("StartNodeAsync() for network: %d, error: %v", config.NetworkID, err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("StopNodeAsync()")
			_, err := s.api.StopNodeAsync()
			s.T().Logf("StopNodeAsync(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("RestartNodeAsync()")
			_, err := s.api.RestartNodeAsync()
			s.T().Logf("RestartNodeAsync(), error: %v", err)
			progress <- struct{}{}
		},
		func(config *params.NodeConfig) {
			log.Info("ResetChainDataAsync()")
			_, err := s.api.ResetChainDataAsync()
			s.T().Logf("ResetChainDataAsync(), error: %v", err)
			progress <- struct{}{}
		},
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
		cnt -= 1
		if cnt <= 0 {
			break
		}
	}

	time.Sleep(2 * time.Second) // so that we see some logs
	s.api.StopNode()            // just in case we have a node running
}
