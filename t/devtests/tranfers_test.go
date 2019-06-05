package devtests

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/services/wallet"
	"github.com/status-im/status-go/t/utils"
	"github.com/stretchr/testify/suite"
)

func TestTransfersSuite(t *testing.T) {
	suite.Run(t, new(TransfersSuite))
}

type TransfersSuite struct {
	DevNodeSuite

	Password string
	Info     account.Info
	Address  common.Address
}

func (s *TransfersSuite) SelectAccount() {
	s.Require().NoError(s.backend.SelectAccount(s.Info.WalletAddress, s.Info.ChatAddress, s.Password))
	_, err := s.backend.AccountManager().SelectedWalletAccount()
	s.Require().NoError(err)
}

func (s *TransfersSuite) SetupTest() {
	s.DevNodeSuite.SetupTest()
	s.Password = "test"
	info, _, err := s.backend.AccountManager().CreateAccount(s.Password)
	s.Require().NoError(err)
	s.Info = info
	s.Address = common.HexToAddress(info.WalletAddress)
}

func (s *TransfersSuite) TearDownTest() {
	s.Require().NoError(s.backend.Logout())
	s.DevNodeSuite.TearDownTest()
}

func (s *TransfersSuite) getAllTranfers() (rst []wallet.Transfer, err error) {
	return rst, s.Local.Call(&rst, "wallet_getTransfersByAddress", s.Address, (*hexutil.Big)(big.NewInt(0)))
}

func (s *TransfersSuite) sendTx(nonce uint64, to common.Address) {
	tx := types.NewTransaction(nonce, to, big.NewInt(1e18), 1e6, big.NewInt(10), nil)
	// TODO move signer to DevNodeSuite
	tx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(1337)), s.DevAccount)
	s.Require().NoError(err)
	s.Require().NoError(s.Eth.SendTransaction(context.Background(), tx))
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = bind.WaitMined(timeout, s.Eth, tx)
	cancel()
	s.Require().NoError(err)
}

func (s *TransfersSuite) TestNewTransfers() {
	s.SelectAccount()
	s.sendTx(0, s.Address)
	s.Require().NoError(utils.Eventually(func() error {
		all, err := s.getAllTranfers()
		if err != nil {
			return err
		}
		if len(all) != 1 {
			return fmt.Errorf("waiting for one transfer")
		}
		return nil
	}, 10*time.Second, 1*time.Second))

	go func() {
		for i := 1; i < 10; i++ {
			s.sendTx(uint64(i), s.Address)
		}
	}()
	s.Require().NoError(utils.Eventually(func() error {
		all, err := s.getAllTranfers()
		if err != nil {
			return err
		}
		if len(all) != 10 {
			return fmt.Errorf("waiting for 10 transfers")
		}
		return nil
	}, 30*time.Second, 1*time.Second))
}

func (s *TransfersSuite) TestHistoricalTransfers() {
	for i := 0; i < 30; i++ {
		s.sendTx(uint64(i), s.Address)
	}
	s.SelectAccount()
	s.Require().NoError(utils.Eventually(func() error {
		all, err := s.getAllTranfers()
		if err != nil {
			return err
		}
		if len(all) >= 30 {
			return fmt.Errorf("waiting for atleast 30 transfers")
		}
		return nil
	}, 30*time.Second, 1*time.Second))
}
