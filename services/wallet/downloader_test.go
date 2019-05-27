package wallet

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/status-im/status-go/services/wallet/erc20"
	"github.com/status-im/status-go/t/devtests/miner"
	"github.com/stretchr/testify/suite"
)

func TestETHTransfers(t *testing.T) {
	suite.Run(t, new(ETHTransferSuite))
}

type ETHTransferSuite struct {
	suite.Suite

	ethclient *ethclient.Client
	identity  *ecdsa.PrivateKey
	faucet    *ecdsa.PrivateKey
	signer    types.Signer

	downloader *ETHTransferDownloader
}

func (s *ETHTransferSuite) SetupTest() {
	var err error
	s.identity, err = crypto.GenerateKey()
	s.Require().NoError(err)
	s.faucet, err = crypto.GenerateKey()
	s.Require().NoError(err)

	node, err := miner.NewDevNode(crypto.PubkeyToAddress(s.faucet.PublicKey))
	s.Require().NoError(err)
	s.Require().NoError(miner.StartWithMiner(node))

	client, err := node.Attach()
	s.Require().NoError(err)
	s.ethclient = ethclient.NewClient(client)
	s.signer = types.NewEIP155Signer(big.NewInt(1337))
	s.downloader = &ETHTransferDownloader{
		signer:  s.signer,
		client:  s.ethclient,
		address: crypto.PubkeyToAddress(s.identity.PublicKey),
	}
}

func (s *ETHTransferSuite) TestNoBalance() {
	ctx := context.TODO()
	tx := types.NewTransaction(0, common.Address{1}, big.NewInt(1e18), 1e6, big.NewInt(10), nil)
	tx, err := types.SignTx(tx, s.signer, s.faucet)
	s.Require().NoError(err)
	s.Require().NoError(s.ethclient.SendTransaction(ctx, tx))
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	s.Require().NoError(err)

	header, err := s.ethclient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err)
	transfers, err := s.downloader.GetTransfers(ctx, header)
	s.Require().NoError(err)
	s.Require().Empty(transfers)
}

func (s *ETHTransferSuite) TestBalanceUpdatedOnInbound() {
	ctx := context.TODO()
	tx := types.NewTransaction(0, crypto.PubkeyToAddress(s.identity.PublicKey), big.NewInt(1e18), 1e6, big.NewInt(10), nil)
	tx, err := types.SignTx(tx, s.signer, s.faucet)
	s.Require().NoError(err)
	s.Require().NoError(s.ethclient.SendTransaction(ctx, tx))
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	s.Require().NoError(err)

	header, err := s.ethclient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(1), header.Number)
	transfers, err := s.downloader.GetTransfers(ctx, header)
	s.Require().NoError(err)
	s.Require().Len(transfers, 1)
}

func (s *ETHTransferSuite) TestBalanceUpdatedOnOutbound() {
	ctx := context.TODO()
	tx := types.NewTransaction(0, crypto.PubkeyToAddress(s.identity.PublicKey), big.NewInt(1e18), 1e6, big.NewInt(10), nil)
	tx, err := types.SignTx(tx, s.signer, s.faucet)
	s.Require().NoError(err)
	s.Require().NoError(s.ethclient.SendTransaction(ctx, tx))
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	cancel()
	s.Require().NoError(err)

	tx = types.NewTransaction(0, common.Address{1}, big.NewInt(5e17), 1e6, big.NewInt(10), nil)
	tx, err = types.SignTx(tx, s.signer, s.identity)
	s.Require().NoError(err)
	s.Require().NoError(s.ethclient.SendTransaction(ctx, tx))
	timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	cancel()
	s.Require().NoError(err)

	header, err := s.ethclient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(2), header.Number)
	transfers, err := s.downloader.GetTransfers(ctx, header)
	s.Require().NoError(err)
	s.Require().Len(transfers, 1)
}

func TestERC20Transfers(t *testing.T) {
	suite.Run(t, new(ERC20TransferSuite))
}

type ERC20TransferSuite struct {
	suite.Suite

	ethclient *ethclient.Client
	identity  *ecdsa.PrivateKey
	faucet    *ecdsa.PrivateKey
	signer    types.Signer

	downloader *ERC20TransfersDownloader

	contract *erc20.ERC20Transfer
}

func (s *ERC20TransferSuite) SetupTest() {
	var err error
	s.identity, err = crypto.GenerateKey()
	s.Require().NoError(err)
	s.faucet, err = crypto.GenerateKey()
	s.Require().NoError(err)

	node, err := miner.NewDevNode(crypto.PubkeyToAddress(s.faucet.PublicKey))
	s.Require().NoError(err)
	s.Require().NoError(miner.StartWithMiner(node))

	client, err := node.Attach()
	s.Require().NoError(err)
	s.ethclient = ethclient.NewClient(client)
	s.downloader = NewERC20TransfersDownloader(s.ethclient, crypto.PubkeyToAddress(s.identity.PublicKey))

	_, tx, contract, err := erc20.DeployERC20Transfer(bind.NewKeyedTransactor(s.faucet), s.ethclient)
	s.Require().NoError(err)
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	s.Require().NoError(err)
	s.contract = contract
	s.signer = types.NewEIP155Signer(big.NewInt(1337))
}

func (s *ERC20TransferSuite) TestNoEvents() {
	header, err := s.ethclient.HeaderByNumber(context.TODO(), nil)
	s.Require().NoError(err)

	transfers, err := s.downloader.GetTransfers(context.TODO(), header)
	s.Require().NoError(err)
	s.Require().Empty(transfers)
}

func (s *ERC20TransferSuite) TestInboundEvent() {
	tx, err := s.contract.Transfer(bind.NewKeyedTransactor(s.faucet), crypto.PubkeyToAddress(s.identity.PublicKey),
		big.NewInt(100))
	s.Require().NoError(err)
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	s.Require().NoError(err)

	header, err := s.ethclient.HeaderByNumber(context.TODO(), nil)
	s.Require().NoError(err)

	transfers, err := s.downloader.GetTransfers(context.TODO(), header)
	s.Require().NoError(err)
	s.Require().Len(transfers, 1)
}

func (s *ERC20TransferSuite) TestOutboundEvent() {
	// give some eth to pay for gas
	ctx := context.TODO()
	// nonce is 1 - contact with nonce 0 was deployed in setup
	// FIXME request nonce
	tx := types.NewTransaction(1, crypto.PubkeyToAddress(s.identity.PublicKey), big.NewInt(1e18), 1e6, big.NewInt(10), nil)
	tx, err := types.SignTx(tx, s.signer, s.faucet)
	s.Require().NoError(err)
	s.Require().NoError(s.ethclient.SendTransaction(ctx, tx))
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	cancel()
	s.Require().NoError(err)

	tx, err = s.contract.Transfer(bind.NewKeyedTransactor(s.identity), common.Address{1}, big.NewInt(100))
	s.Require().NoError(err)
	timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	cancel()
	s.Require().NoError(err)

	header, err := s.ethclient.HeaderByNumber(context.TODO(), nil)
	s.Require().NoError(err)

	transfers, err := s.downloader.GetTransfers(context.TODO(), header)
	s.Require().NoError(err)
	s.Require().Len(transfers, 1)
}
