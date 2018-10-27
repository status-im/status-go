package devtests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/t/devtests/eventer"
	"github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestLogsSuite(t *testing.T) {
	suite.Run(t, new(LogsSuite))
}

type LogsSuite struct {
	DevNodeSuite
}

func (s *LogsSuite) testEmit(event *eventer.Eventer, opts *bind.TransactOpts, id string, topic [32]byte, expect int) {
	tx, err := event.Emit(opts, topic)
	s.NoError(err)
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = bind.WaitMined(timeout, s.Eth, tx)
	s.NoError(err)

	s.NoError(utils.Eventually(func() error {
		var logs []types.Log
		err := s.Local.Call(&logs, "eth_getFilterChanges", id)
		if err != nil {
			return err
		}
		if len(logs) == 0 {
			return errors.New("no logs")
		}
		if len(logs) > expect {
			return errors.New("more logs than expected")
		}
		return nil
	}, 10*time.Second, time.Second))
}

func (s *LogsSuite) TestLogsNewFilter() {
	var (
		fid, sid, tid  string
		ftopic, stopic = [32]byte{1}, [32]byte{2}
	)
	s.Require().NoError(s.Local.Call(&fid, "eth_newFilter", map[string]interface{}{
		"topics": [][]common.Hash{
			{},
			{common.BytesToHash(ftopic[:])},
		},
	}))
	s.Require().NoError(s.Local.Call(&sid, "eth_newFilter", map[string]interface{}{
		"topics": [][]common.Hash{
			{},
			{common.BytesToHash(stopic[:])},
		},
	}))
	s.Require().NoError(s.Local.Call(&tid, "eth_newFilter", map[string]interface{}{}))

	opts := bind.NewKeyedTransactor(s.DevAccount)
	_, tx, event, err := eventer.DeployEventer(opts, s.Eth)
	s.Require().NoError(err)
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addr, err := bind.WaitDeployed(timeout, s.Eth, tx)
	s.Require().NoError(err)
	s.Require().NotEmpty(addr)
	s.testEmit(event, opts, fid, ftopic, 1)
	s.testEmit(event, opts, sid, stopic, 1)
	s.testEmit(event, opts, fid, ftopic, 1)
}
