package sdk

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PublicChannelTestSuite struct {
	suite.Suite
}

func (s *PublicChannelTestSuite) SetupTest() {
}

func (s *PublicChannelTestSuite) TestConnect() {
	c, err := Connect("My user name", "111222333")
	defer c.Close()
	s.Nil(err)
	s.NotNil(c)
}

func TestPublicChannelTestSuite(t *testing.T) {
	suite.Run(t, new(PublicChannelTestSuite))
}
