package api_test

import (
	"testing"

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
		s.api.StartNode(nodeConfig)
		progress <- struct{}{}
	}
	stop := func() {
		s.api.StopNode()
		progress <- struct{}{}
	}

	for i := 0; i < 50; i++ {
		go start()
		go stop()
	}

	cnt := 0
	for range progress {
		cnt += 1
		if cnt >= 100 {
			break
		}
	}
}
