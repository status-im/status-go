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
	w_common "github.com/status-im/status-go/services/wallet/common"
)

type MultiTransactionIDType int64

const (
	NoMultiTransactionID = MultiTransactionIDType(0)
)

func getLogSubTxID(log types.Log) common.Hash {
	// Get unique ID by using TxHash and log index
	index := [4]byte{}
	binary.BigEndian.PutUint32(index[:], uint32(log.Index))
	return crypto.Keccak256Hash(log.TxHash.Bytes(), index[:])
}

var (
	zero = big.NewInt(0)
	one  = big.NewInt(1)
	two  = big.NewInt(2)
)

// Partial transaction info obtained by ERC20Downloader.
// A PreloadedTransaction represents a Transaction which contains one or more
// ERC20/ERC721 transfer events.
// To be converted into one or many Transfer objects post-indexing.
type PreloadedTransaction struct {
	NetworkID   uint64
	Type        w_common.Type  `json:"type"`
	ID          common.Hash    `json:"-"`
	Address     common.Address `json:"address"`
	BlockNumber *big.Int       `json:"blockNumber"`
	BlockHash   common.Hash    `json:"blockhash"`
	Loaded      bool
	// From is derived from tx signature in order to offload this computation from UI component.
	From common.Address `json:"from"`
	// Log that was used to generate preloaded transaction.
	Log         *types.Log `json:"log"`
	BaseGasFees string
}

// Transfer stores information about transfer.
// A Transfer represents a plain ETH transfer or some token activity inside a Transaction
type Transfer struct {
	Type        w_common.Type      `json:"type"`
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

// Only used by status-mobile
func getTransferByHash(ctx context.Context, client *chain.ClientWithFallback, signer types.Signer, address common.Address, hash common.Hash) (*Transfer, error) {
	transaction, _, err := client.TransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	receipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}

	eventType, transactionLog := w_common.GetFirstEvent(receipt.Logs)
	transactionType := w_common.EventTypeToSubtransactionType(eventType)

	from, err := types.Sender(signer, transaction)

	if err != nil {
		return nil, err
	}

	baseGasFee, err := client.GetBaseFeeFromBlock(big.NewInt(int64(transactionLog.BlockNumber)))
	if err != nil {
		return nil, err
	}

	transfer := &Transfer{
		Type:        transactionType,
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
	startTs := time.Now()

	for _, address := range accounts {
		// During block discovery, we should have populated the DB with 1 item per Transaction containing
		// erc20/erc721 transfers
		transactionsToLoad, err := d.db.GetTransactionsToLoad(d.chainClient.ChainID, address, blk.Number())
		if err != nil {
			return nil, err
		}

		areSubTxsCheckedForTxHash := make(map[common.Hash]bool)

		for _, t := range transactionsToLoad {
			subtransactions, err := d.subTransactionsFromTransactionHash(ctx, t.Log.TxHash, address)
			if err != nil {
				log.Error("can't fetch subTxs for erc20/erc721 transfer", "error", err)
				return nil, err
			}
			rst = append(rst, subtransactions...)
			areSubTxsCheckedForTxHash[t.Log.TxHash] = true
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

			isPlainTransfer := from == address || (tx.To() != nil && *tx.To() == address)
			mustCheckSubTxs := false

			if !isPlainTransfer {
				// We might miss some subTransactions of interest for some transaction types. We need to check if we
				// find the address in the transaction data.
				switch tx.Type() {
				case types.OptimismDepositTxType, types.ArbitrumDepositTxType, types.ArbitrumRetryTxType:
					mustCheckSubTxs = !areSubTxsCheckedForTxHash[tx.Hash()] && w_common.TxDataContainsAddress(tx.Type(), tx.Data(), address)
				}
			}

			if isPlainTransfer || mustCheckSubTxs {
				receipt, err := d.chainClient.TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					return nil, err
				}

				baseGasFee, err := d.chainClient.GetBaseFeeFromBlock(blk.Number())
				if err != nil {
					return nil, err
				}

				// Since we've already got the receipt, check for subTxs of
				// interest in case we haven't already.
				if !areSubTxsCheckedForTxHash[tx.Hash()] {
					subtransactions, err := d.subTransactionsFromTransactionData(tx, receipt, blk, baseGasFee, address)
					if err != nil {
						log.Error("can't fetch subTxs for eth transfer", "error", err)
						return nil, err
					}
					rst = append(rst, subtransactions...)
					areSubTxsCheckedForTxHash[tx.Hash()] = true
				}

				// If it's a plain ETH transfer, add it to the list
				if isPlainTransfer {
					rst = append(rst, Transfer{
						Type:               w_common.EthTransfer,
						NetworkID:          tx.ChainId().Uint64(),
						ID:                 tx.Hash(),
						Address:            address,
						BlockNumber:        blk.Number(),
						BlockHash:          receipt.BlockHash,
						Timestamp:          blk.Time(),
						Transaction:        tx,
						From:               from,
						Receipt:            receipt,
						Log:                nil,
						BaseGasFees:        baseGasFee,
						MultiTransactionID: NoMultiTransactionID})
				}
			}
		}
	}
	log.Debug("getTransfersInBlock found", "block", blk.Number(), "len", len(rst), "time", time.Since(startTs))
	// TODO(dshulyak) test that balance difference was covered by transactions
	return rst, nil
}

// NewERC20TransfersDownloader returns new instance.
func NewERC20TransfersDownloader(client *chain.ClientWithFallback, accounts []common.Address, signer types.Signer) *ERC20TransfersDownloader {
	signature := w_common.GetEventSignatureHash(w_common.Erc20_721TransferEventSignature)

	return &ERC20TransfersDownloader{
		client:    client,
		accounts:  accounts,
		signature: signature,
		signer:    signer,
	}
}

// ERC20TransfersDownloader is a downloader for erc20 and erc721 tokens transfers.
// Since both transaction types share the same signature, both will be assigned
// type Erc20Transfer. Until the downloader gets refactored and a migration of the
// database gets implemented, differentiation between erc20 and erc721 will handled
// in the controller.
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

func (d *ETHDownloader) subTransactionsFromTransactionHash(parent context.Context, txHash common.Hash, address common.Address) ([]Transfer, error) {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	tx, _, err := d.chainClient.TransactionByHash(ctx, txHash)
	cancel()
	if err != nil {
		return nil, err
	}

	ctx, cancel = context.WithTimeout(parent, 3*time.Second)
	receipt, err := d.chainClient.TransactionReceipt(ctx, txHash)
	cancel()
	if err != nil {
		return nil, err
	}

	ctx, cancel = context.WithTimeout(parent, 3*time.Second)
	blk, err := d.chainClient.BlockByHash(ctx, receipt.BlockHash)
	cancel()
	if err != nil {
		return nil, err
	}

	baseGasFee, err := d.chainClient.GetBaseFeeFromBlock(receipt.BlockNumber)
	if err != nil {
		return nil, err
	}

	return d.subTransactionsFromTransactionData(tx, receipt, blk, baseGasFee, address)
}

func (d *ETHDownloader) subTransactionsFromTransactionData(tx *types.Transaction, receipt *types.Receipt, blk *types.Block, baseGasFee string, address common.Address) ([]Transfer, error) {
	from, err := types.Sender(d.signer, tx)
	if err != nil {
		if err == core.ErrTxTypeNotSupported {
			return nil, nil
		}
		return nil, err
	}

	rst := make([]Transfer, 0, len(receipt.Logs))

	for _, log := range receipt.Logs {
		eventType := w_common.GetEventType(log)
		// Only add ERC20/ERC721 transfers from/to the given account
		// Other types of events get always added
		mustAppend := false
		switch eventType {
		case w_common.Erc20TransferEventType:
			trFrom, trTo, _ := w_common.ParseErc20TransferLog(log)
			if trFrom == address || trTo == address {
				mustAppend = true
			}
		case w_common.Erc721TransferEventType:
			trFrom, trTo, _ := w_common.ParseErc721TransferLog(log)
			if trFrom == address || trTo == address {
				mustAppend = true
			}
		case w_common.UniswapV2SwapEventType, w_common.UniswapV3SwapEventType:
			mustAppend = true
		case w_common.HopBridgeTransferSentToL2EventType, w_common.HopBridgeTransferFromL1CompletedEventType:
			mustAppend = true
		}

		if mustAppend {
			transfer := Transfer{
				Type:               w_common.EventTypeToSubtransactionType(eventType),
				ID:                 getLogSubTxID(*log),
				Address:            address,
				BlockNumber:        new(big.Int).SetUint64(log.BlockNumber),
				BlockHash:          log.BlockHash,
				Loaded:             true,
				NetworkID:          d.signer.ChainID().Uint64(),
				From:               from,
				Log:                log,
				BaseGasFees:        baseGasFee,
				Transaction:        tx,
				Receipt:            receipt,
				Timestamp:          blk.Time(),
				MultiTransactionID: NoMultiTransactionID,
			}

			rst = append(rst, transfer)
		}
	}

	return rst, nil
}

func (d *ERC20TransfersDownloader) blocksFromLogs(parent context.Context, logs []types.Log, address common.Address) ([]*DBHeader, error) {
	concurrent := NewConcurrentDownloader(parent, NoThreadLimit)
	for i := range logs {
		l := logs[i]

		if l.Removed {
			continue
		}

		id := getLogSubTxID(l)
		baseGasFee, err := d.client.GetBaseFeeFromBlock(new(big.Int).SetUint64(l.BlockNumber))
		if err != nil {
			return nil, err
		}

		header := &DBHeader{
			Number: big.NewInt(int64(l.BlockNumber)),
			Hash:   l.BlockHash,
			PreloadedTransactions: []*PreloadedTransaction{{
				Address:     address,
				BlockNumber: big.NewInt(int64(l.BlockNumber)),
				BlockHash:   l.BlockHash,
				ID:          id,
				From:        address,
				Loaded:      false,
				Type:        w_common.Erc20Transfer,
				Log:         &l,
				BaseGasFees: baseGasFee,
			}},
			Loaded: false,
		}

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

// GetHeadersInRange returns transfers between two blocks.
// time to get logs for 100000 blocks = 1.144686979s. with 249 events in the result set.
func (d *ERC20TransfersDownloader) GetHeadersInRange(parent context.Context, from, to *big.Int) ([]*DBHeader, error) {
	start := time.Now()
	log.Debug("get erc20 transfers in range start", "chainID", d.client.ChainID, "from", from, "to", to)
	headers := []*DBHeader{}
	ctx := context.Background()
	for _, address := range d.accounts {
		outbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: from,
			ToBlock:   to,
			Topics:    d.outboundTopics(address),
		})
		if err != nil {
			return nil, err
		}
		inbound, err := d.client.FilterLogs(ctx, ethereum.FilterQuery{
			FromBlock: from,
			ToBlock:   to,
			Topics:    d.inboundTopics(address),
		})
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
		if len(rst) == 0 {
			log.Warn("no headers found in logs for account", "chainID", d.client.ChainID, "address", address, "from", from, "to", to)
			continue
		} else {
			headers = append(headers, rst...)
			log.Debug("found erc20 transfers for account", "chainID", d.client.ChainID, "address", address,
				"from", from, "to", to, "headers", len(headers))
		}
	}
	log.Debug("get erc20 transfers in range end", "chainID", d.client.ChainID,
		"from", from, "to", to, "headers", len(headers), "took", time.Since(start))
	return headers, nil
}
