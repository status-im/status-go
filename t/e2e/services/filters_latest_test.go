package services

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/suite"

	. "github.com/status-im/status-go/t/utils"
)

func TestFiltersAPISuite(t *testing.T) {
	s := new(FiltersAPISuite)
	s.upstream = false
	suite.Run(t, s)
}

func TestFiltersAPISuiteUpstream(t *testing.T) {
	s := new(FiltersAPISuite)
	s.upstream = true

	if s.upstream && GetNetworkID() == params.StatusChainNetworkID {
		t.Skip()
		return
	}

	suite.Run(t, s)
}

type FiltersAPISuite struct {
	BaseJSONRPCSuite
	upstream bool
}

func (s *FiltersAPISuite) TestFilters() {
	err := s.SetupTest(s.upstream, false, false)
	s.NoError(err)
	defer func() {
		err := s.Backend.StopNode()
		s.NoError(err)
	}()

	basicCall := `{"jsonrpc":"2.0","method":"eth_newBlockFilter","params":[],"id":67}`

	response := s.Backend.CallRPC(basicCall)
	filterID := s.filterIDFromRPCResponse(response)

	// we don't check new blocks on private network, because no one mines them
	if GetNetworkID() != params.StatusChainNetworkID {

		timeout := time.After(time.Minute)
		newBlocksChannel := s.getFirstFilterChange(filterID)

		select {
		case hash := <-newBlocksChannel:
			s.True(len(hash) > 0, "received hash isn't empty")
		case <-timeout:
			s.Fail("timeout while waiting for filter results")
		}

	}

	basicCall = fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_uninstallFilter","params":["%s"],"id":67}`, filterID)

	response = s.Backend.CallRPC(basicCall)
	result := s.boolFromRPCResponse(response)

	s.True(result, "filter expected to be removed successfully")
}

func (s *FiltersAPISuite) getFirstFilterChange(filterID string) chan string {

	result := make(chan string)

	go func() {
		timeout := time.Now().Add(time.Minute)
		for time.Now().Before(timeout) {
			basicCall := fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getFilterChanges","params":["%s"],"id":67}`, filterID)
			response := s.Backend.CallRPC(basicCall)
			filterChanges := s.arrayFromRPCResponse(response)
			if len(filterChanges) > 0 {
				result <- filterChanges[0]
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return result
}

func (s *FiltersAPISuite) filterIDFromRPCResponse(response string) string {
	var r struct {
		Result string `json:"result"`
	}
	s.NoError(json.Unmarshal([]byte(response), &r))

	return r.Result
}

func (s *FiltersAPISuite) arrayFromRPCResponse(response string) []string {
	var r struct {
		Result []string `json:"result"`
	}
	s.NoError(json.Unmarshal([]byte(response), &r))

	return r.Result
}

func (s *FiltersAPISuite) boolFromRPCResponse(response string) bool {
	var r struct {
		Result bool `json:"result"`
	}
	s.NoError(json.Unmarshal([]byte(response), &r))

	return r.Result
}
