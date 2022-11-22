package main

import (
  "testing"
  "io/ioutil"

  "github.com/stretchr/testify/suite"

  "github.com/status-im/status-go/protocol/protobuf"
  "github.com/status-im/status-go/protocol/requests"
  "github.com/status-im/status-go/services/ext"
)


func TestMainSuite(t *testing.T) {
  suite.Run(t, new(MainSuite))
}

type MainSuite struct {
	suite.Suite

        client1 *ext.PublicAPI
        client2 *ext.PublicAPI
}


func (s *MainSuite) SetupTest() {
  file1, err := ioutil.TempFile("/tmp", "status-go-test")
  s.Require().NoError(err)
  client1, err := StartClient("", file1.Name())
  s.Require().NoError(err)
  s.Require().NotNil(client1)

  s.client1, err = client1.ExtPublicAPI()
  s.Require().NoError(err)
  s.Require().NotNil(s.client1)


  /*
  file2, err := ioutil.TempFile("/tmp", "status-go-test")
  s.Require().NoError(err)
  client2, err := StartClient("", file2.Name())
  s.Require().NoError(err)
  s.Require().NotNil(client2)


  s.client2, err = client2.ExtPublicAPI()
  s.Require().NoError(err)
  s.Require().NotNil(s.client2)*/
}

func (s *MainSuite) TestImportCommunity() {
  // Client 1 creates a community
  <-make(chan int)

  request := &requests.CreateCommunity{
    Name: "some-name",
    Color: "#cccccc",
    Description: "some-description",
    Membership: protobuf.CommunityPermissions_ON_REQUEST,
  }

  response, err := s.client1.CreateCommunity(request)
  s.Require().NoError(err)
  s.Require().NotNil(response)

}
