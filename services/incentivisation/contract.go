package incentivisation

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/status-im/status-go/mailserver/registry"
	"math/big"
	"time"
)

type Contract interface {
	Vote(opts *bind.TransactOpts, joinNodes []gethcommon.Address, removeNodes []gethcommon.Address) (*types.Transaction, error)
	GetCurrentSession(opts *bind.CallOpts) (*big.Int, error)
	Registered(opts *bind.CallOpts, publicKey []byte) (bool, error)
	RegisterNode(opts *bind.TransactOpts, publicKey []byte, ip uint32, port uint16) (*types.Transaction, error)
	ActiveNodeCount(opts *bind.CallOpts) (*big.Int, error)
	InactiveNodeCount(opts *bind.CallOpts) (*big.Int, error)
	GetNode(opts *bind.CallOpts, index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error)
	GetInactiveNode(opts *bind.CallOpts, index *big.Int) ([]byte, uint32, uint16, uint32, uint32, error)
	VoteSync(opts *bind.TransactOpts, joinNodes []gethcommon.Address, removeNodes []gethcommon.Address) (*types.Transaction, error)
}

type ContractImpl struct {
	registry.NodesV2
	client *ethclient.Client
}

// VoteSync votes on the contract and wait until the transaction has been accepted, returns an error otherwise
func (c *ContractImpl) VoteSync(opts *bind.TransactOpts, joinNodes []gethcommon.Address, removeNodes []gethcommon.Address) (*types.Transaction, error) {
	tx, err := c.Vote(opts, joinNodes, removeNodes)
	if err != nil {
		return nil, err
	}

	for {
		receipt, _ := c.client.TransactionReceipt(context.TODO(), tx.Hash())
		if receipt != nil {
			if receipt.Status == 0 {
				return nil, errors.New("Invalid receipt")
			}
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	return tx, nil

}

// NewContract creates a new instance of Contract, bound to a specific deployed contract.
func NewContract(address gethcommon.Address, backend bind.ContractBackend, client *ethclient.Client) (Contract, error) {
	contract := &ContractImpl{}
	contract.client = client

	caller, err := registry.NewNodesV2Caller(address, backend)
	if err != nil {
		return nil, err
	}
	contract.NodesV2Caller = *caller

	transactor, err := registry.NewNodesV2Transactor(address, backend)
	if err != nil {
		return nil, err
	}
	contract.NodesV2Transactor = *transactor

	filterer, err := registry.NewNodesV2Filterer(address, backend)
	if err != nil {
		return nil, err
	}
	contract.NodesV2Filterer = *filterer

	return contract, nil
}
