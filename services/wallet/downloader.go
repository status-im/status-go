package wallet

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TransferType type of the asset that was transferred.
type TransferType string

const (
	ethTransfer   TransferType = "eth"
	erc20Transfer TransferType = "erc20"

	erc20TransferEventSignature = "Transfer(address,address,uint256)"
)

var (
	one = big.NewInt(1)
)

// Transfer stores information about transfer.
type Transfer struct {
	Type        TransferType
	Header      *types.Header
	Transaction *types.Transaction
	Receipt     *types.Receipt
}

// ETHTransferDownloader downloads regular eth transfers.
type ETHTransferDownloader struct {
	client  *ethclient.Client
	address common.Address
	signer  types.Signer
}

// GetTransfers checks if the balance was changed between two blocks.
// If so it downloads transaction that transfer ethereum from that block.
func (d *ETHTransferDownloader) GetTransfers(ctx context.Context, header *types.Header) (rst []Transfer, err error) {
	// TODO(dshulyak) consider caching balance and reset it on reorg
	num := new(big.Int).Sub(header.Number, one)
	balance, err := d.client.BalanceAt(ctx, d.address, num)
	if err != nil {
		return nil, err
	}
	current, err := d.client.BalanceAt(ctx, d.address, header.Number)
	if err != nil {
		return nil, err
	}
	if current.Cmp(balance) == 0 {
		return nil, nil
	}
	blk, err := d.client.BlockByHash(ctx, header.Hash())
	if err != nil {
		return nil, err
	}
	for _, tx := range blk.Transactions() {
		if *tx.To() == d.address {
			receipt, err := d.client.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				return nil, err
			}
			rst = append(rst, Transfer{Type: ethTransfer, Header: header, Transaction: tx, Receipt: receipt})
			continue
		}
		from, err := types.Sender(d.signer, tx)
		if err != nil {
			return nil, err
		}
		// payload is empty for eth transfers
		if from == d.address && len(tx.Data()) == 0 {
			receipt, err := d.client.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				return nil, err
			}
			rst = append(rst, Transfer{Type: ethTransfer, Header: header, Transaction: tx, Receipt: receipt})
			continue
		}
	}
	// TODO(dshulyak) test that balance difference was covered by transactions
	return rst, nil
}

// NewERC20TransfersDownloader returns new instance.
func NewERC20TransfersDownloader(client *ethclient.Client, address common.Address) *ERC20TransfersDownloader {
	signature := crypto.Keccak256Hash([]byte(erc20TransferEventSignature))
	target := common.Hash{}
	copy(target[12:], address[:])
	return &ERC20TransfersDownloader{
		client:    client,
		address:   address,
		signature: signature,
		target:    target,
	}
}

// ERC20TransfersDownloader is a downloader for erc20 tokens transfers.
type ERC20TransfersDownloader struct {
	client  *ethclient.Client
	address common.Address

	// hash of the Transfer event signature
	signature common.Hash
	// padded address
	target common.Hash
}

// GetTransfers for erc20 uses eth_getLogs rpc with Transfer event signature and our address acount.
func (d *ERC20TransfersDownloader) GetTransfers(ctx context.Context, header *types.Header) ([]Transfer, error) {
	hash := header.Hash()
	outbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
		BlockHash: &hash,
		Topics:    [][]common.Hash{{d.signature}, {d.target}, {}},
	})
	if err != nil {
		return nil, err
	}
	inbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
		BlockHash: &hash,
		Topics:    [][]common.Hash{{d.signature}, {}, {d.target}},
	})
	if err != nil {
		return nil, err
	}
	lth := len(outbound) + len(inbound)
	if lth == 0 {
		return nil, nil
	}
	all := make([]types.Log, lth)
	copy(all, outbound)
	copy(all[len(outbound):], inbound)
	rst := make([]Transfer, lth)
	for i, l := range all {
		tx, err := d.client.TransactionInBlock(ctx, hash, l.TxIndex)
		if err != nil {
			return nil, err
		}
		receipt, err := d.client.TransactionReceipt(ctx, l.TxHash)
		if err != nil {
			return nil, err
		}
		rst[i] = Transfer{
			Type:        erc20Transfer,
			Header:      header,
			Transaction: tx,
			Receipt:     receipt,
		}
	}
	return rst, nil
}
