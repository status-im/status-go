package main

import (
  "testing"
  "io/ioutil"

  "github.com/stretchr/testify/suite"
)


func TestMainSuite(t *testing.T) {
  suite.Run(t, new(MainSuite))
}

type MainSuite struct {
	suite.Suite
}


func (s *MainSuite) SetupTest() {
  file1, err := ioutil.TempFile("/tmp", "status-go-test")
  s.Require().NoError(err)
  client1, err := StartClient("", file1.Name())
  s.Require().NoError(err)
  s.Require().NotNil(client1)

  file2, err := ioutil.TempFile("/tmp", "status-go-test")
  s.Require().NoError(err)
  client2, err := StartClient("", file2.Name())
  s.Require().NoError(err)
  s.Require().NotNil(client2)
}

func (s *MainSuite) TestBlah() {
  s.Require().NoError(nil)
}
