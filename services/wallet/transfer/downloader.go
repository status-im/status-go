package transfer

import (
	"context"
	"encoding/binary"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/status-im/status-go/rpc/chain"
)

// Type type of the asset that was transferred.
type Type string
type MultiTransactionIDType int64

const (
	ethTransfer   Type = "eth"
	erc20Transfer Type = "erc20"

	erc20TransferEventSignature = "Transfer(address,address,uint256)"

	NoMultiTransactionID = MultiTransactionIDType(0)
)

var (
	zero = big.NewInt(0)
	one  = big.NewInt(1)
	two  = big.NewInt(2)
)

// Transfer stores information about transfer.
type Transfer struct {
	Type        Type               `json:"type"`
	ID          common.Hash        `json:"-"`
	Address     common.Address     `json:"address"`
	BlockNumber *big.Int           `json:"blockNumber"`
	BlockHash   common.Hash        `json:"blockhash"`
	Timestamp   uint64             `json:"timestamp"`
	Transaction *types.Transaction `json:"transaction"`
	Loaded      bool
	NetworkID   uint64
	// From is derived from tx signature in order to offload this computation from UI component.
	From    common.Address `json:"from"`
	Receipt *types.Receipt `json:"receipt"`
	// Log that was used to generate erc20 transfer. Nil for eth transfer.
	Log         *types.Log `json:"log"`
	BaseGasFees string
	// Internal field that is used to track multi-transaction transfers.
	MultiTransactionID MultiTransactionIDType `json:"multi_transaction_id"`
}

// ETHDownloader downloads regular eth transfers.
type ETHDownloader struct {
	chainClient *chain.ClientWithFallback
	accounts    []common.Address
	signer      types.Signer
	db          *Database
}

var errLogsDownloaderStuck = errors.New("logs downloader stuck")

// GetTransfers checks if the balance was changed between two blocks.
// If so it downloads transaction that transfer ethereum from that block.
func (d *ETHDownloader) GetTransfers(ctx context.Context, header *DBHeader) (rst []Transfer, err error) {
	// TODO(dshulyak) consider caching balance and reset it on reorg
	changed := d.accounts
	if len(changed) == 0 {
		return nil, nil
	}
	blk, err := d.chainClient.BlockByHash(ctx, header.Hash)
	if err != nil {
		return nil, err
	}
	rst, err = d.getTransfersInBlock(ctx, blk, changed)
	if err != nil {
		return nil, err
	}
	return rst, nil
}

func (d *ETHDownloader) GetTransfersByNumber(ctx context.Context, number *big.Int) ([]Transfer, error) {
	blk, err := d.chainClient.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	rst, err := d.getTransfersInBlock(ctx, blk, d.accounts)
	if err != nil {
		return nil, err
	}
	return rst, err
}

func getTransferByHash(ctx context.Context, client *chain.ClientWithFallback, signer types.Signer, address common.Address, hash common.Hash) (*Transfer, error) {
	transaction, _, err := client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	receipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	transactionLog := getTokenLog(receipt.Logs)

	transferType := ethTransfer
	if transactionLog != nil {
		transferType = erc20Transfer
	}

	from, err := types.Sender(signer, transaction)

	if err != nil {
		return nil, err
	}

	baseGasFee, err := client.GetBaseFeeFromBlock(big.NewInt(int64(transactionLog.BlockNumber)))
	if err != nil {
		return nil, err
	}

	transfer := &Transfer{Type: transferType,
		ID:          hash,
		Address:     address,
		BlockNumber: receipt.BlockNumber,
		BlockHash:   receipt.BlockHash,
		Timestamp:   uint64(time.Now().Unix()),
		Transaction: transaction,
		From:        from,
		Receipt:     receipt,
		Log:         transactionLog,
		BaseGasFees: baseGasFee,
	}

	return transfer, nil
}

func (d *ETHDownloader) getTransfersInBlock(ctx context.Context, blk *types.Block, accounts []common.Address) (rst []Transfer, err error) {
	for _, address := range accounts {
		preloadedTransfers, err := d.db.GetPreloadedTransactions(d.chainClient.ChainID, address, blk.Hash())
		if err != nil {
			return nil, err
		}
		for _, t := range preloadedTransfers {
			transfer, err := d.transferFromLog(ctx, *t.Log, address, t.ID)
			if err != nil {
				log.Error("can't fetch erc20 transfer from log", "error", err)
				return nil, err
			}
			rst = append(rst, transfer)
		}

		for _, tx := range blk.Transactions() {
			if tx.ChainId().Cmp(big.NewInt(0)) != 0 && tx.ChainId().Cmp(d.chainClient.ToBigInt()) != 0 {
				log.Info("chain id mismatch", "tx hash", tx.Hash(), "tx chain id", tx.ChainId(), "expected chain id", d.chainClient.ChainID)
				continue
			}
			from, err := types.Sender(d.signer, tx)

			if err != nil {
				if err == core.ErrTxTypeNotSupported {
					continue
				}
				return nil, err
			}

			if from == address || (tx.To() != nil && *tx.To() == address) {
				receipt, err := d.chainClient.TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					return nil, err
				}

				transactionLog := getTokenLog(receipt.Logs)

				baseGasFee, err := d.chainClient.GetBaseFeeFromBlock(blk.Number())
				if err != nil {
					return nil, err
				}

				if transactionLog == nil {
					rst = append(rst, Transfer{
						Type:        ethTransfer,
						ID:          tx.Hash(),
						Address:     address,
						BlockNumber: blk.Number(),
						BlockHash:   blk.Hash(),
						Timestamp:   blk.Time(),
						Transaction: tx,
						From:        from,
						Receipt:     receipt,
						Log:         transactionLog,
						BaseGasFees: baseGasFee})
				}
			}
		}
	}
	log.Debug("getTransfersInBlock found", "block", blk.Number(), "len", len(rst))
	// TODO(dshulyak) test that balance difference was covered by transactions
	return rst, nil
}

// NewERC20TransfersDownloader returns new instance.
func NewERC20TransfersDownloader(client *chain.ClientWithFallback, accounts []common.Address, signer types.Signer) *ERC20TransfersDownloader {
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
	client   *chain.ClientWithFallback
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

func (d *ETHDownloader) transferFromLog(parent context.Context, ethlog types.Log, address common.Address, id common.Hash) (Transfer, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	tx, _, err := d.chainClient.TransactionByHash(ctx, ethlog.TxHash)
	cancel()
	if err != nil {
		return Transfer{}, err
	}
	from, err := types.Sender(d.signer, tx)
	if err != nil {
		return Transfer{}, err
	}
	ctx, cancel = context.WithTimeout(parent, 3*time.Second)
	receipt, err := d.chainClient.TransactionReceipt(ctx, ethlog.TxHash)
	cancel()
	if err != nil {
		return Transfer{}, err
	}

	baseGasFee, err := d.chainClient.GetBaseFeeFromBlock(new(big.Int).SetUint64(ethlog.BlockNumber))
	if err != nil {
		return Transfer{}, err
	}

	ctx, cancel = context.WithTimeout(parent, 3*time.Second)
	blk, err := d.chainClient.BlockByHash(ctx, ethlog.BlockHash)
	cancel()
	if err != nil {
		return Transfer{}, err
	}
	return Transfer{
		Address:     address,
		ID:          id,
		Type:        erc20Transfer,
		BlockNumber: new(big.Int).SetUint64(ethlog.BlockNumber),
		BlockHash:   ethlog.BlockHash,
		Transaction: tx,
		From:        from,
		Receipt:     receipt,
		Timestamp:   blk.Time(),
		Log:         &ethlog,
		BaseGasFees: baseGasFee,
	}, nil
}

func (d *ERC20TransfersDownloader) transferFromLog(parent context.Context, ethlog types.Log, address common.Address) (Transfer, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	tx, _, err := d.client.TransactionByHash(ctx, ethlog.TxHash)
	cancel()
	if err != nil {
		return Transfer{}, err
	}
	from, err := types.Sender(d.signer, tx)
	if err != nil {
		return Transfer{}, err
	}
	ctx, cancel = context.WithTimeout(parent, 3*time.Second)
	receipt, err := d.client.TransactionReceipt(ctx, ethlog.TxHash)
	cancel()
	if err != nil {
		return Transfer{}, err
	}

	baseGasFee, err := d.client.GetBaseFeeFromBlock(new(big.Int).SetUint64(ethlog.BlockNumber))
	if err != nil {
		return Transfer{}, err
	}

	ctx, cancel = context.WithTimeout(parent, 3*time.Second)
	blk, err := d.client.BlockByHash(ctx, ethlog.BlockHash)
	cancel()
	if err != nil {
		return Transfer{}, err
	}
	index := [4]byte{}
	binary.BigEndian.PutUint32(index[:], uint32(ethlog.Index))
	id := crypto.Keccak256Hash(ethlog.TxHash.Bytes(), index[:])
	return Transfer{
		Address:     address,
		ID:          id,
		Type:        erc20Transfer,
		BlockNumber: new(big.Int).SetUint64(ethlog.BlockNumber),
		BlockHash:   ethlog.BlockHash,
		Transaction: tx,
		From:        from,
		Receipt:     receipt,
		Timestamp:   blk.Time(),
		Log:         &ethlog,
		BaseGasFees: baseGasFee,
	}, nil
}

func (d *ERC20TransfersDownloader) transfersFromLogs(parent context.Context, logs []types.Log, address common.Address) ([]Transfer, error) {
	concurrent := NewConcurrentDownloader(parent)
	for i := range logs {
		l := logs[i]
		if l.Removed {
			continue
		}
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
		return nil, errLogsDownloaderStuck
	}
	return concurrent.Get(), concurrent.Error()
}

func (d *ERC20TransfersDownloader) blocksFromLogs(parent context.Context, logs []types.Log, address common.Address) ([]*DBHeader, error) {
	concurrent := NewConcurrentDownloader(parent)
	for i := range logs {
		l := logs[i]

		if l.Removed {
			continue
		}

		index := [4]byte{}
		binary.BigEndian.PutUint32(index[:], uint32(l.Index))
		id := crypto.Keccak256Hash(l.TxHash.Bytes(), index[:])

		baseGasFee, err := d.client.GetBaseFeeFromBlock(new(big.Int).SetUint64(l.BlockNumber))
		if err != nil {
			return nil, err
		}

		header := &DBHeader{
			Number: big.NewInt(int64(l.BlockNumber)),
			Hash:   l.BlockHash,
			Erc20Transfers: []*Transfer{{
				Address:     address,
				BlockNumber: big.NewInt(int64(l.BlockNumber)),
				BlockHash:   l.BlockHash,
				ID:          id,
				From:        address,
				Loaded:      false,
				Type:        erc20Transfer,
				Log:         &l,
				BaseGasFees: baseGasFee,
			}}}

		concurrent.Add(func(ctx context.Context) error {
			concurrent.PushHeader(header)
			return nil
		})
	}
	select {
	case <-concurrent.WaitAsync():
	case <-parent.Done():
		return nil, errLogsDownloaderStuck
	}
	return concurrent.GetHeaders(), concurrent.Error()
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

// GetHeadersInRange returns transfers between two blocks.
// time to get logs for 100000 blocks = 1.144686979s. with 249 events in the result set.
func (d *ERC20TransfersDownloader) GetHeadersInRange(parent context.Context, from, to *big.Int) ([]*DBHeader, error) {
	start := time.Now()
	log.Debug("get erc20 transfers in range", "from", from, "to", to)
	headers := []*DBHeader{}
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
		rst, err := d.blocksFromLogs(parent, logs, address)
		if err != nil {
			return nil, err
		}
		headers = append(headers, rst...)
	}
	log.Debug("found erc20 transfers between two blocks", "from", from, "to", to, "headers", len(headers), "took", time.Since(start))
	return headers, nil
}

func IsTokenTransfer(logs []*types.Log) bool {
	signature := crypto.Keccak256Hash([]byte(erc20TransferEventSignature))
	for _, l := range logs {
		if len(l.Topics) > 0 && l.Topics[0] == signature {
			return true
		}
	}
	return false
}

func getTokenLog(logs []*types.Log) *types.Log {
	signature := crypto.Keccak256Hash([]byte(erc20TransferEventSignature))
	for _, l := range logs {
		if len(l.Topics) > 0 && l.Topics[0] == signature {
			return l
		}
	}
	return nil
}
