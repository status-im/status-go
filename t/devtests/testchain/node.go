package testchain

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

type Backend struct {
	Node     *node.Node
	Client   *ethclient.Client
	genesis  *core.Genesis
	Ethereum *eth.Ethereum
	Faucet   *ecdsa.PrivateKey
	Signer   types.Signer
}

func NewBackend() (*Backend, error) {
	faucet, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	config := params.AllEthashProtocolChanges
	genesis := &core.Genesis{
		Config:    config,
		Alloc:     core.GenesisAlloc{crypto.PubkeyToAddress(faucet.PublicKey): {Balance: big.NewInt(1e18)}},
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
	}
	var ethservice *eth.Ethereum
	n, err := node.New(&node.Config{})
	if err != nil {
		return nil, err
	}
	err = n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		config := &eth.Config{Genesis: genesis}
		config.Ethash.PowMode = ethash.ModeFake
		ethservice, err = eth.New(ctx, config)
		return ethservice, err
	})
	if err != nil {
		return nil, err
	}

	if err := n.Start(); err != nil {
		return nil, err
	}
	client, err := n.Attach()
	if err != nil {
		return nil, err
	}
	return &Backend{
		Node:     n,
		Client:   ethclient.NewClient(client),
		Ethereum: ethservice,
		Faucet:   faucet,
		Signer:   types.NewEIP155Signer(config.ChainID),
		genesis:  genesis,
	}, nil
}

// GenerateBlocks generate n blocks starting from genesis.
func (b *Backend) GenerateBlocks(n int, start uint64, gen func(int, *core.BlockGen)) []*types.Block {
	block := b.Ethereum.BlockChain().GetBlockByNumber(start)
	engine := ethash.NewFaker()
	blocks, _ := core.GenerateChain(b.genesis.Config, block, engine, b.Ethereum.ChainDb(), n, gen)
	return blocks
}

func (b *Backend) Stop() error {
	return b.Node.Stop()
}
