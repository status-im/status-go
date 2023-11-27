package protocol

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/status-im/status-go/services/browsers"
)

func TestBrowserSuite(t *testing.T) {
	suite.Run(t, new(BrowserSuite))
}

type BrowserSuite struct {
	MessengerBaseTestSuite
}

func (s *BrowserSuite) SetupTest() {
	s.MessengerBaseTestSuite.SetupTest()
	_, err := s.m.Start()
	s.Require().NoError(err)
}

func (s *MessengerBackupSuite) TestBrowsersOrderedNewestFirst() {
	msngr := s.newMessenger()
	testBrowsers := []*browsers.Browser{
		{
			ID:        "1",
			Name:      "first",
			Dapp:      true,
			Timestamp: 10,
		},
		{
			ID:        "2",
			Name:      "second",
			Dapp:      true,
			Timestamp: 50,
		},
		{
			ID:           "3",
			Name:         "third",
			Dapp:         true,
			Timestamp:    100,
			HistoryIndex: 0,
			History:      []string{"zero"},
		},
	}
	for i := 0; i < len(testBrowsers); i++ {
		s.Require().NoError(msngr.AddBrowser(context.TODO(), *testBrowsers[i]))
	}

	sort.Slice(testBrowsers, func(i, j int) bool {
		return testBrowsers[i].Timestamp > testBrowsers[j].Timestamp
	})

	rst, err := msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Equal(testBrowsers, rst)
}

func (s *MessengerBackupSuite) TestBrowsersHistoryIncluded() {
	msngr := s.newMessenger()
	browser := &browsers.Browser{
		ID:           "1",
		Name:         "first",
		Dapp:         true,
		Timestamp:    10,
		HistoryIndex: 1,
		History:      []string{"one", "two"},
	}
	s.Require().NoError(msngr.AddBrowser(context.TODO(), *browser))
	rst, err := msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Len(rst, 1)
	s.Require().Equal(browser, rst[0])
}

func (s *MessengerBackupSuite) TestBrowsersReplaceOnUpdate() {
	msngr := s.newMessenger()
	browser := &browsers.Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: 10,
		History:   []string{"one", "two"},
	}
	s.Require().NoError(msngr.AddBrowser(context.TODO(), *browser))
	browser.Dapp = false
	browser.History = []string{"one", "three"}
	browser.Timestamp = 107
	s.Require().NoError(msngr.AddBrowser(context.TODO(), *browser))
	rst, err := msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Len(rst, 1)
	s.Require().Equal(browser, rst[0])
}

func (s *MessengerBackupSuite) TestDeleteBrowser() {
	msngr := s.newMessenger()
	browser := &browsers.Browser{
		ID:        "1",
		Name:      "first",
		Dapp:      true,
		Timestamp: 10,
		History:   []string{"one", "two"},
	}

	s.Require().NoError(msngr.AddBrowser(context.TODO(), *browser))
	rst, err := msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Len(rst, 1)

	s.Require().NoError(msngr.DeleteBrowser(context.TODO(), browser.ID))
	rst, err = msngr.GetBrowsers(context.TODO())
	s.Require().NoError(err)
	s.Require().Len(rst, 0)
}
