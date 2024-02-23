package protocol

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/protocol/tt"
)

type MessengerCommunityForMobileTestingTestSuite struct {
	MessengerBaseTestSuite
}

func TestMessengerCommunityForMobileTesting(t *testing.T) {
	suite.Run(t, new(MessengerCommunityForMobileTestingTestSuite))
}

func (s *MessengerCommunityForMobileTestingTestSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()
}

func (s *MessengerCommunityForMobileTestingTestSuite) TearDownTest() {
	s.MessengerBaseTestSuite.TearDownTest()
}

func (s *MessengerCommunityForMobileTestingTestSuite) TestCreateClosedCommunity() {
	var wg sync.WaitGroup
	wg.Add(1)
	// simulate invoking `HandleCommunityDescription`
	go func() {
		err := tt.RetryWithBackOff(func() error {
			r, err := s.m.RetrieveAll()
			s.Require().NoError(err)
			if len(r.Communities()) > 0 {
				return nil
			}
			return errors.New("not receive enough messages relate to community")
		})
		wg.Done()
		s.Require().NoError(err)
	}()

	wg.Add(1)
	var communityID types.HexBytes
	// simulate frontend(mobile) invoking `CreateClosedCommunity`
	go func() {
		response, err := s.m.CreateClosedCommunity()
		s.Require().NoError(err)
		s.Require().Len(response.Communities(), 1)
		s.Require().Len(response.Communities()[0].Categories(), 2)
		s.Require().Len(response.Chats(), 4)
		s.Require().Len(response.Communities()[0].Description().Chats, 4)
		communityID = response.Communities()[0].ID()
		wg.Done()
	}()

	timeout := time.After(20 * time.Second)
	done := make(chan bool)
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-timeout:
		s.Fail("TestCreateClosedCommunity timed out")
	case <-done:
		// validate concurrent updating community result
		lastCommunity, err := s.m.GetCommunityByID(communityID)
		s.Require().NoError(err)
		s.Require().Len(lastCommunity.Categories(), 2)
		s.Require().Len(lastCommunity.Description().Chats, 4)
	}
}
