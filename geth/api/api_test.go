package api_test

import (
	"fmt"
	"testing"
	"time"

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

func (s *APITestSuite) TestStartStopRaces() {
	require := s.Require()

	nodeConfig, err := MakeTestNodeConfig(params.RinkebyNetworkID)
	require.NoError(err)

	progress := make(chan struct{}, 100)

	start := func() {
		fmt.Println("start node")
		s.api.StartNode(nodeConfig)
		progress <- struct{}{}
	}
	stop := func() {
		fmt.Println("stop node")
		s.api.StopNode()
		progress <- struct{}{}
	}

	// start one node and sync it a bit
	start()
	time.Sleep(5 * time.Second)

	for i := 0; i < 20; i++ {
		go start()
		go stop()
	}

	cnt := 0
	for range progress {
		cnt += 1
		if cnt >= 40 {
			break
		}
	}
}
