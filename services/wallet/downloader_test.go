package wallet

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/status-im/status-go/services/wallet/erc20"
	"github.com/status-im/status-go/t/devtests/miner"
)

func TestETHTransfers(t *testing.T) {
	suite.Run(t, new(ETHTransferSuite))
}

type ETHTransferSuite struct {
	suite.Suite

	ethclient           *ethclient.Client
	identity, secondary *ecdsa.PrivateKey
	faucet              *ecdsa.PrivateKey
	signer              types.Signer
	dbStop              func()

	downloader *ETHTransferDownloader
}

func (s *ETHTransferSuite) SetupTest() {
	var err error
	s.identity, err = crypto.GenerateKey()
	s.Require().NoError(err)
	s.faucet, err = crypto.GenerateKey()
	s.Require().NoError(err)
	s.secondary, err = crypto.GenerateKey()
	s.Require().NoError(err)

	node, err := miner.NewDevNode(crypto.PubkeyToAddress(s.faucet.PublicKey))
	s.Require().NoError(err)
	s.Require().NoError(miner.StartWithMiner(node))

	client, err := node.Attach()
	s.Require().NoError(err)
	s.ethclient = ethclient.NewClient(client)
	s.signer = types.NewEIP155Signer(big.NewInt(1337))
	db, stop := setupTestDB(s.Suite.T())
	s.dbStop = stop
	s.downloader = &ETHTransferDownloader{
		signer: s.signer,
		client: &walletClient{client: s.ethclient},
		db:     db,
		accounts: []common.Address{
			crypto.PubkeyToAddress(s.identity.PublicKey),
			crypto.PubkeyToAddress(s.secondary.PublicKey)},
	}
}

func (s *ETHTransferSuite) TearDownTest() {
	s.dbStop()
}

// signAndMineTx signs transaction with provided key and waits for it to be mined.
// uses configured faucet key if pkey is nil.
func (s *ETHTransferSuite) signAndMineTx(tx *types.Transaction, pkey *ecdsa.PrivateKey) {
	if pkey == nil {
		pkey = s.faucet
	}
	tx, err := types.SignTx(tx, s.signer, pkey)
	s.Require().NoError(err)
	s.Require().NoError(s.ethclient.SendTransaction(context.Background(), tx))
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	s.Require().NoError(err)
}

func (s *ETHTransferSuite) TestNoBalance() {
	ctx := context.TODO()
	tx := types.NewTransaction(0, common.Address{1}, big.NewInt(1e18), 1e6, big.NewInt(10), nil)
	s.signAndMineTx(tx, nil)

	header, err := s.ethclient.HeaderByNumber(ctx, nil)
	s.Require().NoError(err)
	transfers, err := s.downloader.GetTransfers(ctx, toDBHeader(header))
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
	transfers, err := s.downloader.GetTransfers(ctx, toDBHeader(header))
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
	transfers, err := s.downloader.GetTransfers(ctx, toDBHeader(header))
	s.Require().NoError(err)
	s.Require().Len(transfers, 1)
}

func (s *ETHTransferSuite) TestMultipleReferences() {
	tx := types.NewTransaction(0, crypto.PubkeyToAddress(s.identity.PublicKey), big.NewInt(1e18), 1e6, big.NewInt(10), nil)
	s.signAndMineTx(tx, nil)
	tx = types.NewTransaction(0, crypto.PubkeyToAddress(s.secondary.PublicKey), big.NewInt(1e17), 1e6, big.NewInt(10), nil)
	s.signAndMineTx(tx, s.identity)

	header, err := s.ethclient.HeaderByNumber(context.Background(), nil)
	s.Require().NoError(err)
	s.Require().Equal(big.NewInt(2), header.Number)
	transfers, err := s.downloader.GetTransfers(context.Background(), toDBHeader(header))
	s.Require().NoError(err)
	s.Require().Len(transfers, 2)
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
	s.signer = types.NewEIP155Signer(big.NewInt(1337))
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
	s.downloader = NewERC20TransfersDownloader(&walletClient{client: s.ethclient}, []common.Address{crypto.PubkeyToAddress(s.identity.PublicKey)}, s.signer)

	var (
		tx       *types.Transaction
		contract *erc20.ERC20Transfer
	)
	for i := 0; i <= 3; i++ {
		opts := bind.NewKeyedTransactor(s.faucet)
		_, tx, contract, err = erc20.DeployERC20Transfer(opts, s.ethclient)
		if err != nil {
			continue
		}
	}
	s.Require().NoError(err)
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	s.Require().NoError(err)
	s.contract = contract
}

func (s *ERC20TransferSuite) TestNoEvents() {
	header, err := s.ethclient.HeaderByNumber(context.TODO(), nil)
	s.Require().NoError(err)

	transfers, err := s.downloader.GetTransfers(context.TODO(), toDBHeader(header))
	s.Require().NoError(err)
	s.Require().Empty(transfers)
}

func (s *ERC20TransferSuite) TestInboundEvent() {
	opts := bind.NewKeyedTransactor(s.faucet)
	tx, err := s.contract.Transfer(opts, crypto.PubkeyToAddress(s.identity.PublicKey), big.NewInt(100))
	s.Require().NoError(err)
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	s.Require().NoError(err)

	header, err := s.ethclient.HeaderByNumber(context.TODO(), nil)
	s.Require().NoError(err)

	transfers, err := s.downloader.GetTransfers(context.TODO(), toDBHeader(header))
	s.Require().NoError(err)
	s.Require().Len(transfers, 1)
}

func (s *ERC20TransferSuite) TestOutboundEvent() {
	// give some eth to pay for gas
	ctx := context.TODO()
	tx := types.NewTransaction(4, crypto.PubkeyToAddress(s.identity.PublicKey), big.NewInt(1e18), 1e6, big.NewInt(10), nil)
	tx, err := types.SignTx(tx, s.signer, s.faucet)
	s.Require().NoError(err)
	s.Require().NoError(s.ethclient.SendTransaction(ctx, tx))
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	cancel()
	s.Require().NoError(err)

	opts := bind.NewKeyedTransactor(s.identity)
	tx, err = s.contract.Transfer(opts, common.Address{1}, big.NewInt(100))
	s.Require().NoError(err)
	timeout, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	_, err = bind.WaitMined(timeout, s.ethclient, tx)
	cancel()
	s.Require().NoError(err)

	header, err := s.ethclient.HeaderByNumber(context.TODO(), nil)
	s.Require().NoError(err)

	transfers, err := s.downloader.GetTransfers(context.TODO(), toDBHeader(header))
	s.Require().NoError(err)
	s.Require().Len(transfers, 1)
}

func (s *ERC20TransferSuite) TestInRange() {
	for i := 0; i < 5; i++ {
		tx, err := s.contract.Transfer(bind.NewKeyedTransactor(s.faucet), crypto.PubkeyToAddress(s.identity.PublicKey),
			big.NewInt(100))
		s.Require().NoError(err)
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = bind.WaitMined(timeout, s.ethclient, tx)
		s.Require().NoError(err)
	}
	transfers, err := s.downloader.GetHeadersInRange(context.TODO(), big.NewInt(1), nil)
	s.Require().NoError(err)
	s.Require().Len(transfers, 5)
}
