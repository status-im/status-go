package wallet

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// TransferType type of the asset that was transferred.
type TransferType string

const (
	ethTransfer   TransferType = "eth"
	erc20Transfer TransferType = "erc20"

	erc20TransferEventSignature = "Transfer(address,address,uint256)"
)

var (
	one  = big.NewInt(1)
	zero = big.NewInt(0)
	two  = big.NewInt(2)
)

// Transfer stores information about transfer.
type Transfer struct {
	Type        TransferType       `json:"type"`
	Address     common.Address     `json:"address"`
	BlockNumber *big.Int           `json:"blockNumber"`
	BlockHash   common.Hash        `json:"blockhash"`
	Transaction *types.Transaction `json:"transaction"`
	Receipt     *types.Receipt     `json:"receipt"`
}

// ETHTransferDownloader downloads regular eth transfers.
type ETHTransferDownloader struct {
	client   *ethclient.Client
	accounts []common.Address
	signer   types.Signer
}

// GetTransfers checks if the balance was changed between two blocks.
// If so it downloads transaction that transfer ethereum from that block.
func (d *ETHTransferDownloader) GetTransfers(ctx context.Context, header *DBHeader) (rst []Transfer, err error) {
	// TODO(dshulyak) consider caching balance and reset it on reorg
	num := new(big.Int).Sub(header.Number, one)
	changed := []common.Address{}
	for _, address := range d.accounts {
		balance, err := d.client.BalanceAt(ctx, address, num)
		if err != nil {
			return nil, err
		}
		current, err := d.client.BalanceAt(ctx, address, header.Number)
		if err != nil {
			return nil, err
		}
		if current.Cmp(balance) != 0 {
			changed = append(changed, address)
		}
	}
	if len(changed) == 0 {
		return nil, nil
	}
	blk, err := d.client.BlockByHash(ctx, header.Hash)
	if err != nil {
		return nil, err
	}
	rst, err = d.getTransfersInBlock(ctx, blk, changed)
	if err != nil {
		return nil, err
	}
	return rst, nil
}

func (d *ETHTransferDownloader) GetTransfersByNumber(ctx context.Context, number *big.Int) ([]Transfer, error) {
	blk, err := d.client.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	rst, err := d.getTransfersInBlock(ctx, blk, d.accounts)
	if err != nil {
		return nil, err
	}
	return rst, err
}

func (d *ETHTransferDownloader) getTransfersInBlock(ctx context.Context, blk *types.Block, accounts []common.Address) (rst []Transfer, err error) {
	for _, tx := range blk.Transactions() {
		from, err := types.Sender(d.signer, tx)
		if err != nil {
			return nil, err
		}
		// payload is empty for eth transfers
		if any(from, accounts) {
			receipt, err := d.client.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				return nil, err
			}
			rst = append(rst, Transfer{Type: ethTransfer,
				Address:     from,
				BlockNumber: blk.Number(),
				BlockHash:   blk.Hash(),
				Transaction: tx, Receipt: receipt})
			continue
		}
		if tx.To() == nil {
			continue
		}
		if any(*tx.To(), accounts) {
			receipt, err := d.client.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				return nil, err
			}
			rst = append(rst, Transfer{Type: ethTransfer,
				Address:     *tx.To(),
				BlockNumber: blk.Number(),
				BlockHash:   blk.Hash(),
				Transaction: tx, Receipt: receipt})
			continue
		}
	}
	// TODO(dshulyak) test that balance difference was covered by transactions
	return rst, nil
}

// NewERC20TransfersDownloader returns new instance.
func NewERC20TransfersDownloader(client *ethclient.Client, accounts []common.Address) *ERC20TransfersDownloader {
	signature := crypto.Keccak256Hash([]byte(erc20TransferEventSignature))
	return &ERC20TransfersDownloader{
		client:    client,
		accounts:  accounts,
		signature: signature,
	}
}

// ERC20TransfersDownloader is a downloader for erc20 tokens transfers.
type ERC20TransfersDownloader struct {
	client   *ethclient.Client
	accounts []common.Address

	// hash of the Transfer event signature
	signature common.Hash
}

func (d *ERC20TransfersDownloader) paddedAddress(address common.Address) common.Hash {
	rst := common.Hash{}
	copy(rst[12:], address[:])
	return rst
}

func (d *ERC20TransfersDownloader) inboundTopics(address common.Address) [][]common.Hash {
	return [][]common.Hash{{d.signature}, {}, {d.paddedAddress(address)}}
}

func (d *ERC20TransfersDownloader) outboundTopics(address common.Address) [][]common.Hash {
	return [][]common.Hash{{d.signature}, {d.paddedAddress(address)}, {}}
}

func (d *ERC20TransfersDownloader) transfersFromLogs(parent context.Context, logs []types.Log, address common.Address) ([]Transfer, error) {
	rst := make([]Transfer, len(logs))
	for i, l := range logs {
		// TODO(dshulyak) use TransactionInBlock after it is fixed
		ctx, cancel := context.WithTimeout(parent, 3*time.Second)
		tx, _, err := d.client.TransactionByHash(ctx, l.TxHash)
		cancel()
		if err != nil {
			return nil, err
		}
		ctx, cancel = context.WithTimeout(parent, 3*time.Second)
		receipt, err := d.client.TransactionReceipt(ctx, l.TxHash)
		cancel()
		if err != nil {
			return nil, err
		}
		rst[i] = Transfer{
			Address:     address,
			Type:        erc20Transfer,
			BlockNumber: new(big.Int).SetUint64(l.BlockNumber),
			BlockHash:   l.BlockHash,
			Transaction: tx,
			Receipt:     receipt,
		}
	}
	return rst, nil
}

func any(address common.Address, compare []common.Address) bool {
	for _, c := range compare {
		if c == address {
			return true
		}
	}
	return false
}

// GetTransfers for erc20 uses eth_getLogs rpc with Transfer event signature and our address acount.
func (d *ERC20TransfersDownloader) GetTransfers(ctx context.Context, header *DBHeader) ([]Transfer, error) {
	hash := header.Hash
	transfers := []Transfer{}
	for _, address := range d.accounts {
		outbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
			BlockHash: &hash,
			Topics:    d.outboundTopics(address),
		})
		if err != nil {
			return nil, err
		}
		inbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
			BlockHash: &hash,
			Topics:    d.inboundTopics(address),
		})
		if err != nil {
			return nil, err
		}
		logs := append(outbound, inbound...)
		if len(logs) == 0 {
			continue
		}
		rst, err := d.transfersFromLogs(ctx, logs, address)
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, rst...)
	}
	return transfers, nil
}

// GetTransfersInRange returns transfers between two blocks.
// time to get logs for 100000 blocks = 1.144686979s. with 249 events in the result set.
func (d *ERC20TransfersDownloader) GetTransfersInRange(parent context.Context, from, to *big.Int) ([]Transfer, error) {
	start := time.Now()
	log.Debug("get erc20 transfers in range", "from", from, "to", to)
	transfers := []Transfer{}
	for _, address := range d.accounts {
		ctx, cancel := context.WithTimeout(parent, 5*time.Second)
		outbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: from,
			ToBlock:   to,
			Topics:    d.outboundTopics(address),
		})
		cancel()
		if err != nil {
			return nil, err
		}
		ctx, cancel = context.WithTimeout(parent, 5*time.Second)
		inbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: from,
			ToBlock:   to,
			Topics:    d.inboundTopics(address),
		})
		cancel()
		if err != nil {
			return nil, err
		}
		logs := append(outbound, inbound...)
		if len(logs) == 0 {
			continue
		}
		rst, err := d.transfersFromLogs(parent, logs, address)
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, rst...)
	}
	log.Debug("found erc20 transfers between two blocks", "from", from, "to", to, "lth", len(transfers), "took", time.Since(start))
	return transfers, nil
}
