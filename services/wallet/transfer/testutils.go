package transfer

import (
	"database/sql"
	"fmt"
	"math/big"
	"testing"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/services/wallet/common"
	w_common "github.com/status-im/status-go/services/wallet/common"
	"github.com/status-im/status-go/services/wallet/testutils"
	"github.com/status-im/status-go/services/wallet/token"

	"github.com/stretchr/testify/require"
)

type TestTransaction struct {
	Hash               eth_common.Hash
	ChainID            common.ChainID
	From               eth_common.Address // [sender]
	Timestamp          int64
	BlkNumber          int64
	Success            bool
	MultiTransactionID MultiTransactionIDType
}

type TestTransfer struct {
	TestTransaction
	To    eth_common.Address // [address]
	Value int64
	Token *token.Token
}

type TestMultiTransaction struct {
	MultiTransactionID   MultiTransactionIDType
	MultiTransactionType MultiTransactionType
	FromAddress          eth_common.Address
	ToAddress            eth_common.Address
	FromToken            string
	ToToken              string
	FromAmount           int64
	ToAmount             int64
	Timestamp            int64
}

func SeedToToken(seed int) *token.Token {
	tokenIndex := seed % len(TestTokens)
	return TestTokens[tokenIndex]
}

func TestTrToToken(t *testing.T, tt *TestTransaction) (token *token.Token, isNative bool) {
	// Sanity check that none of the markers changed and they should be equal to seed
	require.Equal(t, tt.Timestamp, tt.BlkNumber)

	tokenIndex := int(tt.Timestamp) % len(TestTokens)
	isNative = testutils.SliceContains(NativeTokenIndices, tokenIndex)

	return TestTokens[tokenIndex], isNative
}

func generateTestTransaction(seed int) TestTransaction {
	token := SeedToToken(seed)
	return TestTransaction{
		Hash:               eth_common.HexToHash(fmt.Sprintf("0x1%d", seed)),
		ChainID:            common.ChainID(token.ChainID),
		From:               eth_common.HexToAddress(fmt.Sprintf("0x2%d", seed)),
		Timestamp:          int64(seed),
		BlkNumber:          int64(seed),
		Success:            true,
		MultiTransactionID: NoMultiTransactionID,
	}
}

func generateTestTransfer(seed int) TestTransfer {
	tokenIndex := seed % len(TestTokens)
	token := TestTokens[tokenIndex]
	return TestTransfer{
		TestTransaction: generateTestTransaction(seed),
		To:              eth_common.HexToAddress(fmt.Sprintf("0x3%d", seed)),
		Value:           int64(seed),
		Token:           token,
	}
}

func GenerateTestSendMultiTransaction(tr TestTransfer) TestMultiTransaction {
	return TestMultiTransaction{
		MultiTransactionType: MultiTransactionSend,
		FromAddress:          tr.From,
		ToAddress:            tr.To,
		FromToken:            tr.Token.Symbol,
		ToToken:              tr.Token.Symbol,
		FromAmount:           tr.Value,
		ToAmount:             0,
		Timestamp:            tr.Timestamp,
	}
}

func GenerateTestSwapMultiTransaction(tr TestTransfer, toToken string, toAmount int64) TestMultiTransaction {
	return TestMultiTransaction{
		MultiTransactionType: MultiTransactionSwap,
		FromAddress:          tr.From,
		ToAddress:            tr.To,
		FromToken:            tr.Token.Symbol,
		ToToken:              toToken,
		FromAmount:           tr.Value,
		ToAmount:             toAmount,
		Timestamp:            tr.Timestamp,
	}
}

func GenerateTestBridgeMultiTransaction(fromTr, toTr TestTransfer) TestMultiTransaction {
	return TestMultiTransaction{
		MultiTransactionType: MultiTransactionBridge,
		FromAddress:          fromTr.From,
		ToAddress:            toTr.To,
		FromToken:            fromTr.Token.Symbol,
		ToToken:              toTr.Token.Symbol,
		FromAmount:           fromTr.Value,
		ToAmount:             toTr.Value,
		Timestamp:            fromTr.Timestamp,
	}
}

// GenerateTestTransfers will generate transaction based on the TestTokens index and roll over if there are more than
// len(TestTokens) transactions
func GenerateTestTransfers(t *testing.T, db *sql.DB, firstStartIndex int, count int) (result []TestTransfer, fromAddresses, toAddresses []eth_common.Address) {
	for i := firstStartIndex; i < (firstStartIndex + count); i++ {
		tr := generateTestTransfer(i)
		fromAddresses = append(fromAddresses, tr.From)
		toAddresses = append(toAddresses, tr.To)
		result = append(result, tr)
	}
	return
}

var EthMainnet = token.Token{
	Address: eth_common.HexToAddress("0x"),
	Name:    "Ether",
	Symbol:  "ETH",
	ChainID: 1,
}

var EthGoerli = token.Token{
	Address: eth_common.HexToAddress("0x"),
	Name:    "Ether",
	Symbol:  "ETH",
	ChainID: 5,
}

var EthOptimism = token.Token{
	Address: eth_common.HexToAddress("0x"),
	Name:    "Ether",
	Symbol:  "ETH",
	ChainID: 10,
}

var UsdcMainnet = token.Token{
	Address: eth_common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
	Name:    "USD Coin",
	Symbol:  "USDC",
	ChainID: 1,
}

var UsdcGoerli = token.Token{
	Address: eth_common.HexToAddress("0x98339d8c260052b7ad81c28c16c0b98420f2b46a"),
	Name:    "USD Coin",
	Symbol:  "USDC",
	ChainID: 5,
}

var UsdcOptimism = token.Token{
	Address: eth_common.HexToAddress("0x7f5c764cbc14f9669b88837ca1490cca17c31607"),
	Name:    "USD Coin",
	Symbol:  "USDC",
	ChainID: 10,
}

var SntMainnet = token.Token{
	Address: eth_common.HexToAddress("0x744d70fdbe2ba4cf95131626614a1763df805b9e"),
	Name:    "Status Network Token",
	Symbol:  "SNT",
	ChainID: 1,
}

var DaiMainnet = token.Token{
	Address: eth_common.HexToAddress("0xf2edF1c091f683E3fb452497d9a98A49cBA84666"),
	Name:    "DAI Stablecoin",
	Symbol:  "DAI",
	ChainID: 5,
}

var DaiGoerli = token.Token{
	Address: eth_common.HexToAddress("0xf2edF1c091f683E3fb452497d9a98A49cBA84666"),
	Name:    "DAI Stablecoin",
	Symbol:  "DAI",
	ChainID: 5,
}

// TestTokens contains ETH/Mainnet, ETH/Goerli, ETH/Optimism, USDC/Mainnet, USDC/Goerli, USDC/Optimism, SNT/Mainnet, DAI/Mainnet, DAI/Goerli
var TestTokens = []*token.Token{
	&EthMainnet, &EthGoerli, &EthOptimism, &UsdcMainnet, &UsdcGoerli, &UsdcOptimism, &SntMainnet, &DaiMainnet, &DaiGoerli,
}

var NativeTokenIndices = []int{0, 1, 2}

func InsertTestTransfer(t *testing.T, db *sql.DB, address eth_common.Address, tr *TestTransfer) {
	token := TestTokens[int(tr.Timestamp)%len(TestTokens)]
	InsertTestTransferWithOptions(t, db, address, tr, &TestTransferOptions{
		TokenAddress: token.Address,
	})
}

type TestTransferOptions struct {
	TokenAddress     eth_common.Address
	NullifyAddresses []eth_common.Address
}

func InsertTestTransferWithOptions(t *testing.T, db *sql.DB, address eth_common.Address, tr *TestTransfer, opt *TestTransferOptions) {
	var (
		tx *sql.Tx
	)
	tx, err := db.Begin()
	require.NoError(t, err)
	defer func() {
		if err == nil {
			err = tx.Commit()
			return
		}
		_ = tx.Rollback()
	}()

	blkHash := eth_common.HexToHash("4")

	block := blockDBFields{
		chainID:     uint64(tr.ChainID),
		account:     tr.To,
		blockNumber: big.NewInt(tr.BlkNumber),
		blockHash:   blkHash,
	}

	// Respect `FOREIGN KEY(network_id,address,blk_hash)` of `transfers` table
	err = insertBlockDBFields(tx, block)
	require.NoError(t, err)

	receiptStatus := uint64(0)
	if tr.Success {
		receiptStatus = 1
	}

	tokenType := "eth"
	if (opt.TokenAddress != eth_common.Address{}) {
		tokenType = "erc20"
	}

	// Workaround to simulate writing of NULL values for addresses
	txTo := &tr.To
	txFrom := &tr.From
	for i := 0; i < len(opt.NullifyAddresses); i++ {
		if opt.NullifyAddresses[i] == tr.To {
			txTo = nil
		}
		if opt.NullifyAddresses[i] == tr.From {
			txFrom = nil
		}
	}

	transfer := transferDBFields{
		chainID:            uint64(tr.ChainID),
		id:                 tr.Hash,
		address:            address,
		blockHash:          blkHash,
		blockNumber:        big.NewInt(tr.BlkNumber),
		sender:             tr.From,
		transferType:       w_common.Type(tokenType),
		timestamp:          uint64(tr.Timestamp),
		multiTransactionID: tr.MultiTransactionID,
		baseGasFees:        "0x0",
		receiptStatus:      &receiptStatus,
		txValue:            big.NewInt(tr.Value),
		txFrom:             txFrom,
		txTo:               txTo,
		tokenAddress:       &opt.TokenAddress,
	}
	err = updateOrInsertTransfersDBFields(tx, []transferDBFields{transfer})
	require.NoError(t, err)
}

func InsertTestPendingTransaction(t *testing.T, db *sql.DB, tr *TestTransfer) {
	_, err := db.Exec(`
		INSERT INTO pending_transactions (network_id, hash, timestamp, from_address, to_address,
			symbol, gas_price, gas_limit, value, data, type, additional_data, multi_transaction_id
		) VALUES (?, ?, ?, ?, ?, 'ETH', 0, 0, ?, '', 'test', '', ?)`,
		tr.ChainID, tr.Hash, tr.Timestamp, tr.From, tr.To, tr.Value, tr.MultiTransactionID)
	require.NoError(t, err)
}

func InsertTestMultiTransaction(t *testing.T, db *sql.DB, tr *TestMultiTransaction) MultiTransactionIDType {
	fromTokenType := tr.FromToken
	if tr.FromToken == "" {
		fromTokenType = testutils.EthSymbol
	}
	toTokenType := tr.ToToken
	if tr.ToToken == "" {
		toTokenType = testutils.EthSymbol
	}
	fromAmount := (*hexutil.Big)(big.NewInt(tr.FromAmount))
	toAmount := (*hexutil.Big)(big.NewInt(tr.ToAmount))

	result, err := db.Exec(`
		INSERT INTO multi_transactions (from_address, from_asset, from_amount, to_address, to_asset, to_amount, type, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tr.FromAddress, fromTokenType, fromAmount.String(), tr.ToAddress, toTokenType, toAmount.String(), tr.MultiTransactionType, tr.Timestamp)
	require.NoError(t, err)
	rowID, err := result.LastInsertId()
	require.NoError(t, err)
	tr.MultiTransactionID = MultiTransactionIDType(rowID)
	return tr.MultiTransactionID
}
