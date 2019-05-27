package miner

import (
	"io/ioutil"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
)

// NewDevNode returns node with clieque engine and prefunded accounts.
func NewDevNode(faucet common.Address) (*node.Node, error) {
	cfg := node.DefaultConfig
	ipc, err := ioutil.TempFile("", "devnode-ipc-")
	if err != nil {
		return nil, err
	}
	cfg.IPCPath = ipc.Name()
	cfg.HTTPModules = []string{"eth"}
	cfg.DataDir = ""
	cfg.NoUSB = true
	cfg.P2P.MaxPeers = 0
	cfg.P2P.ListenAddr = ":0"
	cfg.P2P.NoDiscovery = true
	cfg.P2P.DiscoveryV5 = false

	stack, err := node.New(&cfg)
	if err != nil {
		return nil, err
	}

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	// ensure that etherbase is added to an account manager
	etherbase, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	acc, err := ks.ImportECDSA(etherbase, "")
	if err != nil {
		return nil, err
	}
	err = ks.Unlock(acc, "")
	if err != nil {
		return nil, err
	}

	ethcfg := eth.DefaultConfig
	ethcfg.NetworkId = 1337
	// 0 - mine only if transaction pending
	ethcfg.Genesis = core.DeveloperGenesisBlock(0, faucet)
	extra := make([]byte, 32) // extraVanity
	extra = append(extra, acc.Address[:]...)
	extra = append(extra, make([]byte, 65)...) // extraSeal
	ethcfg.Genesis.ExtraData = extra
	ethcfg.MinerGasPrice = big.NewInt(1)
	ethcfg.Etherbase = acc.Address

	return stack, stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return eth.New(ctx, &ethcfg)
	})

}

// StartWithMiner starts node with eth service and a miner.
func StartWithMiner(stack *node.Node) error {
	err := stack.Start()
	if err != nil {
		return err
	}
	var ethereum *eth.Ethereum
	err = stack.Service(&ethereum)
	if err != nil {
		return err
	}
	ethereum.TxPool().SetGasPrice(big.NewInt(1))
	return ethereum.StartMining(0)
}
