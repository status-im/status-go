package wallet

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/status-im/status-go/services/wallet/erc20"
	"github.com/status-im/status-go/t/devtests/miner"
)

func TestBalancesSuite(t *testing.T) {
	suite.Run(t, new(BalancesSuite))
}

type BalancesSuite struct {
	suite.Suite

	tokens   []common.Address
	accounts []common.Address

	client *walletClient

	faucet *ecdsa.PrivateKey
}

func (s *BalancesSuite) SetupTest() {
	var err error
	s.faucet, err = crypto.GenerateKey()
	s.Require().NoError(err)

	node, err := miner.NewDevNode(crypto.PubkeyToAddress(s.faucet.PublicKey))
	s.Require().NoError(err)
	s.Require().NoError(miner.StartWithMiner(node))

	client, err := node.Attach()
	s.Require().NoError(err)
	s.client = &walletClient{ethclient.NewClient(client)}

	s.tokens = make([]common.Address, 3)
	s.accounts = make([]common.Address, 5)
	for i := range s.accounts {
		key, err := crypto.GenerateKey()
		s.Require().NoError(err)
		s.accounts[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	for i := range s.tokens {
		token, tx, _, err := erc20.DeployERC20Transfer(bind.NewKeyedTransactor(s.faucet), s.client.client)
		s.Require().NoError(err)
		_, err = bind.WaitMined(context.Background(), s.client, tx)
		s.Require().NoError(err)
		s.tokens[i] = token
	}
}

func (s *BalancesSuite) TestBalanceEqualPerToken() {
	base := big.NewInt(10)
	expected := map[common.Address]map[common.Address]*hexutil.Big{}
	for _, account := range s.accounts {
		expected[account] = map[common.Address]*hexutil.Big{}
		for i, token := range s.tokens {
			balance := new(big.Int).Add(base, big.NewInt(int64(i)))
			transactor, err := erc20.NewERC20Transfer(token, s.client.client)
			s.Require().NoError(err)
			tx, err := transactor.Transfer(bind.NewKeyedTransactor(s.faucet), account, balance)
			s.Require().NoError(err)
			_, err = bind.WaitMined(context.Background(), s.client, tx)
			s.Require().NoError(err)
			expected[account][token] = (*hexutil.Big)(balance)
		}
	}
	result, err := GetTokensBalances(context.Background(), s.client, s.accounts, s.tokens)
	s.Require().NoError(err)
	s.Require().Equal(expected, result)
}

func (s *BalancesSuite) TestBalanceEqualPerAccount() {
	base := big.NewInt(10)
	expected := map[common.Address]map[common.Address]*hexutil.Big{}
	for i, account := range s.accounts {
		expected[account] = map[common.Address]*hexutil.Big{}
		for _, token := range s.tokens {
			balance := new(big.Int).Add(base, big.NewInt(int64(i)))
			transactor, err := erc20.NewERC20Transfer(token, s.client.client)
			s.Require().NoError(err)
			tx, err := transactor.Transfer(bind.NewKeyedTransactor(s.faucet), account, balance)
			s.Require().NoError(err)
			_, err = bind.WaitMined(context.Background(), s.client, tx)
			s.Require().NoError(err)
			expected[account][token] = (*hexutil.Big)(balance)
		}
	}
	result, err := GetTokensBalances(context.Background(), s.client, s.accounts, s.tokens)
	s.Require().NoError(err)
	s.Require().Equal(expected, result)
}

func (s *BalancesSuite) TestNoBalances() {
	result, err := GetTokensBalances(context.Background(), s.client, s.accounts, s.tokens)
	s.Require().NoError(err)
	for _, account := range s.accounts {
		for _, token := range s.tokens {
			s.Require().Equal(zero.Int64(), result[account][token].ToInt().Int64())
		}
	}
}

func (s *BalancesSuite) TestNoTokens() {
	expected := map[common.Address]map[common.Address]*hexutil.Big{}
	result, err := GetTokensBalances(context.Background(), s.client, s.accounts, nil)
	s.Require().NoError(err)
	s.Require().Equal(expected, result)
}

func (s *BalancesSuite) TestNoAccounts() {
	expected := map[common.Address]map[common.Address]*hexutil.Big{}
	result, err := GetTokensBalances(context.Background(), s.client, nil, s.tokens)
	s.Require().NoError(err)
	s.Require().Equal(expected, result)
}

func (s *BalancesSuite) TestTokenNotDeployed() {
	_, err := GetTokensBalances(context.Background(), s.client, s.accounts, []common.Address{{0x01}})
	s.Require().NoError(err)
}

func (s *BalancesSuite) TestInterrupted() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := GetTokensBalances(ctx, s.client, s.accounts, s.tokens)
	s.Require().EqualError(err, "context canceled")
}
