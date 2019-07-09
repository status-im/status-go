package wallet

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	zero = big.NewInt(0)
	one  = big.NewInt(1)
	two  = big.NewInt(2)
)

// Transfer stores information about transfer.
type Transfer struct {
	Type        TransferType       `json:"type"`
	ID          common.Hash        `json:"-"`
	Address     common.Address     `json:"address"`
	BlockNumber *big.Int           `json:"blockNumber"`
	BlockHash   common.Hash        `json:"blockhash"`
	Timestamp   uint64             `json:"timestamp"`
	Transaction *types.Transaction `json:"transaction"`
	// From is derived from tx signature in order to offload this computation from UI component.
	From    common.Address `json:"from"`
	Receipt *types.Receipt `json:"receipt"`
}

func (t Transfer) MarshalJSON() ([]byte, error) {
	m := transferMarshaling{}
	m.Type = t.Type
	m.Address = t.Address
	m.BlockNumber = (*hexutil.Big)(t.BlockNumber)
	m.BlockHash = t.BlockHash
	m.Timestamp = hexutil.Uint64(t.Timestamp)
	m.Transaction = t.Transaction
	m.From = t.From
	m.Receipt = t.Receipt
	return json.Marshal(m)
}

func (t *Transfer) UnmarshalJSON(input []byte) error {
	m := transferMarshaling{}
	err := json.Unmarshal(input, &m)
	if err != nil {
		return err
	}
	t.Type = m.Type
	t.Address = m.Address
	t.BlockNumber = (*big.Int)(m.BlockNumber)
	t.BlockHash = m.BlockHash
	t.Timestamp = uint64(m.Timestamp)
	t.Transaction = m.Transaction
	m.From = t.From
	m.Receipt = t.Receipt
	return nil
}

// transferMarshaling ensures that all integers will be marshalled with hexutil
// to be consistent with types.Transaction and types.Receipt.
type transferMarshaling struct {
	Type        TransferType       `json:"type"`
	Address     common.Address     `json:"address"`
	BlockNumber *hexutil.Big       `json:"blockNumber"`
	BlockHash   common.Hash        `json:"blockhash"`
	Timestamp   hexutil.Uint64     `json:"timestamp"`
	Transaction *types.Transaction `json:"transaction"`
	// From is derived from tx signature in order to offload this computation from UI component.
	From    common.Address `json:"from"`
	Receipt *types.Receipt `json:"receipt"`
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
		var address *common.Address
		from, err := types.Sender(d.signer, tx)
		if err != nil {
			return nil, err
		}
		if any(from, accounts) {
			address = &from
		} else if tx.To() != nil && any(*tx.To(), accounts) {
			address = tx.To()
		}
		if address != nil {
			receipt, err := d.client.TransactionReceipt(ctx, tx.Hash())
			if err != nil {
				return nil, err
			}
			if isTokenTransfer(receipt.Logs) {
				log.Debug("eth downloader found token transfer", "hash", tx.Hash())
				continue
			}
			rst = append(rst, Transfer{
				Type:        ethTransfer,
				ID:          tx.Hash(),
				Address:     *address,
				BlockNumber: blk.Number(),
				BlockHash:   blk.Hash(),
				Timestamp:   blk.Time(),
				Transaction: tx,
				From:        from,
				Receipt:     receipt})

		}
	}
	// TODO(dshulyak) test that balance difference was covered by transactions
	return rst, nil
}

// NewERC20TransfersDownloader returns new instance.
func NewERC20TransfersDownloader(client *ethclient.Client, accounts []common.Address, signer types.Signer) *ERC20TransfersDownloader {
	signature := crypto.Keccak256Hash([]byte(erc20TransferEventSignature))
	return &ERC20TransfersDownloader{
		client:    client,
		accounts:  accounts,
		signature: signature,
		signer:    signer,
	}
}

// ERC20TransfersDownloader is a downloader for erc20 tokens transfers.
type ERC20TransfersDownloader struct {
	client   *ethclient.Client
	accounts []common.Address

	// hash of the Transfer event signature
	signature common.Hash

	// signer is used to derive tx sender from tx signature
	signer types.Signer
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

func (d *ERC20TransfersDownloader) transferFromLog(parent context.Context, log types.Log, address common.Address) (Transfer, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	tx, _, err := d.client.TransactionByHash(ctx, log.TxHash)
	cancel()
	if err != nil {
		return Transfer{}, err
	}
	from, err := types.Sender(d.signer, tx)
	if err != nil {
		return Transfer{}, err
	}
	ctx, cancel = context.WithTimeout(parent, 3*time.Second)
	receipt, err := d.client.TransactionReceipt(ctx, log.TxHash)
	cancel()
	if err != nil {
		return Transfer{}, err
	}
	ctx, cancel = context.WithTimeout(parent, 3*time.Second)
	blk, err := d.client.BlockByHash(ctx, log.BlockHash)
	if err != nil {
		return Transfer{}, err
	}
	cancel()
	// TODO(dshulyak) what is the max number of logs?
	index := [4]byte{}
	binary.BigEndian.PutUint32(index[:], uint32(log.Index))
	id := crypto.Keccak256Hash(log.TxHash.Bytes(), index[:])
	return Transfer{
		Address:     address,
		ID:          id,
		Type:        erc20Transfer,
		BlockNumber: new(big.Int).SetUint64(log.BlockNumber),
		BlockHash:   log.BlockHash,
		Transaction: tx,
		From:        from,
		Receipt:     receipt,
		Timestamp:   blk.Time(),
	}, nil
}

func (d *ERC20TransfersDownloader) transfersFromLogs(parent context.Context, logs []types.Log, address common.Address) ([]Transfer, error) {
	concurrent := NewConcurrentDownloader(parent)
	for i := range logs {
		l := logs[i]
		concurrent.Add(func(ctx context.Context) error {
			transfer, err := d.transferFromLog(ctx, l, address)
			if err != nil {
				return err
			}
			concurrent.Push(transfer)
			return nil
		})
	}
	select {
	case <-concurrent.WaitAsync():
	case <-parent.Done():
		return nil, errors.New("logs downloader stuck")
	}
	return concurrent.Get(), nil
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

func any(address common.Address, compare []common.Address) bool {
	for _, c := range compare {
		if c == address {
			return true
		}
	}
	return false
}

func isTokenTransfer(logs []*types.Log) bool {
	signature := crypto.Keccak256Hash([]byte(erc20TransferEventSignature))
	for _, l := range logs {
		if len(l.Topics) > 0 && l.Topics[0] == signature {
			return true
		}
	}
	return false
}
