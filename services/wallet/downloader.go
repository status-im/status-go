package wallet

import (
	"context"
	"errors"
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
	rst, err = d.getTransfersInBlock(ctx, blk)
	if err != nil {
		return nil, err
	}
	if len(rst) == 0 {
		return nil, errors.New("balance changed but no new transactions were found")
	}
	return rst, nil
}

func (d *ETHTransferDownloader) getTransfersInBlock(ctx context.Context, blk *types.Block) (rst []Transfer, err error) {
	for _, tx := range blk.Transactions() {
		if *tx.To() == d.address {
			receipt, err := d.client.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				return nil, err
			}
			rst = append(rst, Transfer{Type: ethTransfer, Header: blk.Header(), Transaction: tx, Receipt: receipt})
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
			rst = append(rst, Transfer{Type: ethTransfer, Header: blk.Header(), Transaction: tx, Receipt: receipt})
			continue
		}
	}
	// TODO(dshulyak) test that balance difference was covered by transactions
	return rst, nil
}

func (d *ETHTransferDownloader) GetTransfersInRange(ctx context.Context, from, to *types.Header) (rst []Transfer, err error) {
	older, err := d.client.BalanceAt(ctx, d.address, from.Number)
	if err != nil {
		return nil, err
	}
	newer, err := d.client.BalanceAt(ctx, d.address, to.Number)
	if err != nil {
		return nil, err
	}
	// need better name
	num := new(big.Int).Set(to.Number)
	// on every iteration newer will get one step closer to odler.
	// once balance is the same we consider that all possible transfers were found
	for older.Cmp(newer) != 0 {
		num = num.Sub(to.Number, one)
		update, err := d.client.BalanceAt(ctx, d.address, num)
		if err != nil {
			return nil, err
		}
		if update.Cmp(newer) != 0 {
			// FIXME store both
			blk, err := d.client.BlockByNumber(ctx, new(big.Int).Add(num, one))
			if err != nil {
				return nil, err
			}
			transfers, err := d.getTransfersInBlock(ctx, blk)
			if err != nil {
				return nil, err
			}
			rst = append(rst, transfers...)
		}
		newer = update
	}
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

func (d *ERC20TransfersDownloader) inboundTopics() [][]common.Hash {
	return [][]common.Hash{{d.signature}, {}, {d.target}}
}

func (d *ERC20TransfersDownloader) outboundTopics() [][]common.Hash {
	return [][]common.Hash{{d.signature}, {d.target}, {}}
}

func (d *ERC20TransfersDownloader) transfersFromLogs(ctx context.Context, logs []types.Log) ([]Transfer, error) {
	rst := make([]Transfer, len(logs))
	for i, l := range logs {
		header, err := d.client.HeaderByHash(ctx, l.BlockHash)
		if err != nil {
			return nil, err
		}
		tx, err := d.client.TransactionInBlock(ctx, l.BlockHash, l.TxIndex)
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

// GetTransfers for erc20 uses eth_getLogs rpc with Transfer event signature and our address acount.
func (d *ERC20TransfersDownloader) GetTransfers(ctx context.Context, header *types.Header) ([]Transfer, error) {
	hash := header.Hash()
	outbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
		BlockHash: &hash,
		Topics:    d.outboundTopics(),
	})
	if err != nil {
		return nil, err
	}
	inbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
		BlockHash: &hash,
		Topics:    d.inboundTopics(),
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

// GetTransfersInRange returns transfers between two blocks.
// time to get logs for 100000 blocks = 1.144686979s. with 249 events in the result set.
func (d *ERC20TransfersDownloader) GetTransfersInRange(ctx context.Context, from, to *types.Header) ([]Transfer, error) {
	outbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: from.Number,
		ToBlock:   to.Number,
		Topics:    d.outboundTopics(),
	})
	if err != nil {
		return nil, err
	}
	inbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: from.Number,
		ToBlock:   to.Number,
		Topics:    d.inboundTopics(),
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
	return d.transfersFromLogs(ctx, all)
}
