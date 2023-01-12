package collectibles

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/account"
	"github.com/status-im/status-go/contracts/collectibles"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/rpc"
	"github.com/status-im/status-go/transactions"
)

func NewAPI(rpcClient *rpc.Client, accountsManager *account.GethManager, config *params.NodeConfig) *API {
	return &API{
		RPCClient:       rpcClient,
		accountsManager: accountsManager,
		config:          config,
	}
}

type API struct {
	RPCClient       *rpc.Client
	accountsManager *account.GethManager
	config          *params.NodeConfig
}

func (api *API) getSigner(chainID uint64, from types.Address, password string) bind.SignerFn {
	return func(addr common.Address, tx *ethTypes.Transaction) (*ethTypes.Transaction, error) {
		selectedAccount, err := api.accountsManager.VerifyAccountPassword(api.config.KeyStoreDir, from.Hex(), password)
		if err != nil {
			return nil, err
		}
		s := ethTypes.NewLondonSigner(new(big.Int).SetUint64(chainID))
		return ethTypes.SignTx(tx, s, selectedAccount.PrivateKey)
	}
}

func (api *API) Deploy(ctx context.Context, chainID uint64, txArgs transactions.SendTxArgs, password string) error {

	// TODO use txArgs.toTransactOpts ?
	transactOpts := bind.TransactOpts{
		From:    common.Address(txArgs.From),
		Signer:  api.getSigner(chainID, txArgs.From, password),
		Context: context.Background(),
	}

	ethClient, err := api.RPCClient.EthClient(chainID)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	// TODO set gas options ?
	//transactOpts.Nonce = big.NewInt(int64(nonce))
	//transactOpts.Value = big.NewInt(0)     // in wei
	//transactOpts.GasLimit = uint64(300000) // in units
	//transactOpts.GasPrice = gasPrice

	// TODO add context ?
	address, tx, instance, err := collectibles.DeployCollectibles(&transactOpts, ethClient)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	fmt.Println("Contract deployed to:", address.Hex())
	fmt.Println("Transaction hash:", tx.Hash().Hex())
	fmt.Printf("Instance %+v\n", instance)
	return nil
}
