package peer

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

func TestPeerSuite(t *testing.T) {
	suite.Run(t, new(PeerSuite))
}

type PeerSuite struct {
	suite.Suite
	api *PublicAPI
	s   *Service
	d   *MockDiscoverer
}

func (s *PeerSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.d = NewMockDiscoverer(ctrl)
	s.s = New()
	s.api = NewAPI(s.s)
}

var discovertests = []struct {
	name                string
	expectedError       error
	prepareExpectations func(*PeerSuite)
	request             DiscoverRequest
}{
	{
		name:          "success discover",
		expectedError: nil,
		prepareExpectations: func(s *PeerSuite) {
			s.d.EXPECT().Discover("topic", 10, 1).Return(nil)
		},
		request: DiscoverRequest{
			Topic: "topic",
			Max:   10,
			Min:   1,
		},
	},
	{
		name:          "range 0",
		expectedError: nil,
		prepareExpectations: func(s *PeerSuite) {
			s.d.EXPECT().Discover("topic", 10, 10).Return(nil)
		},
		request: DiscoverRequest{
			Topic: "topic",
			Max:   10,
			Min:   10,
		},
	},
	{
		name:                "invalid topic",
		expectedError:       ErrInvalidTopic,
		prepareExpectations: func(s *PeerSuite) {},
		request: DiscoverRequest{
			Topic: "",
			Max:   10,
			Min:   1,
		},
	},
	{
		name:                "invalid range",
		expectedError:       ErrInvalidRange,
		prepareExpectations: func(s *PeerSuite) {},
		request: DiscoverRequest{
			Topic: "topic",
			Max:   1,
			Min:   10,
		},
	},
	{
		name:          "success discover",
		expectedError: nil,
		prepareExpectations: func(s *PeerSuite) {
			s.d.EXPECT().Discover("topic", 10, 1).Return(nil)
		},
		request: DiscoverRequest{
			Topic: "topic",
			Max:   10,
			Min:   1,
		},
	},
	{
		name:          "errored discover",
		expectedError: errors.New("could not create the specified account : foo"),
		prepareExpectations: func(s *PeerSuite) {
			s.d.EXPECT().Discover("topic", 10, 1).Return(errors.New("could not create the specified account : foo"))
		},
		request: DiscoverRequest{
			Topic: "topic",
			Max:   10,
			Min:   1,
		},
	},
}

func (s *PeerSuite) TestDiscover() {
	for _, tc := range discovertests {
		s.T().Run(tc.name, func(t *testing.T) {
			s.s.SetDiscoverer(s.d)
			tc.prepareExpectations(s)

			var ctx context.Context
			err := s.api.Discover(ctx, tc.request)
			s.Equal(tc.expectedError, err, "failed scenario : "+tc.name)
		})
	}
}

func (s *PeerSuite) TestDiscoverWihEmptyDiscoverer() {
	var ctx context.Context
	s.Equal(ErrDiscovererNotProvided, s.api.Discover(ctx, DiscoverRequest{
		Topic: "topic",
		Max:   10,
		Min:   1,
	}))
}
