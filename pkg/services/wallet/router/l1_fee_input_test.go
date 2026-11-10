package router

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/status-im/status-go/params"
	walletCommon "github.com/status-im/status-go/pkg/services/wallet/common"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
	types "github.com/status-im/status-go/pkg/services/wallet/token/tokenlist"
	tokentypes "github.com/status-im/status-go/pkg/services/wallet/token/types"
)

var (
	l1FeeTestRecipient = common.HexToAddress("0x1111111111111111111111111111111111111111")
	l1FeeTestToken     = common.HexToAddress("0x2222222222222222222222222222222222222222")
	l1FeeTestSpender   = common.HexToAddress("0x3333333333333333333333333333333333333333")
)

// oracleRecorder keeps what the L1 fee oracle was asked to price.
type oracleRecorder struct {
	inputs [][]byte
	fee    *big.Int
	err    error
}

func (o *oracleRecorder) l1Fee(_ uint64, tx []byte) (*big.Int, error) {
	o.inputs = append(o.inputs, tx)
	if o.err != nil {
		return nil, o.err
	}
	return o.fee, nil
}

func decodeOracleInput(t *testing.T, input []byte) *ethTypes.Transaction {
	t.Helper()
	tx := new(ethTypes.Transaction)
	require.NoError(t, tx.UnmarshalBinary(input), "the oracle input must be a serialized transaction")
	return tx
}

func scrollPath(token *tokentypes.Token) *routes.Path {
	nonce := hexutil.Uint64(7)
	return &routes.Path{
		ProcessorName:   pathProcessorCommon.ProcessorTransferName,
		FromChain:       &params.Network{ChainID: walletCommon.ScrollMainnet, EIP1559Enabled: true},
		FromToken:       token,
		AmountIn:        bigHex(1_000_000),
		TxNonce:         &nonce,
		TxGasAmount:     21_000,
		TxMaxFeesPerGas: bigHex(2_000_000),
		TxPriorityFee:   bigHex(1_000),
	}
}

func nativeToken() *tokentypes.Token {
	return &tokentypes.Token{Token: &types.Token{Symbol: "ETH", Address: common.Address{}}}
}

func erc20Token() *tokentypes.Token {
	return &tokentypes.Token{Token: &types.Token{Symbol: "USDC", Address: l1FeeTestToken}}
}

func TestPathL1Fees_NativeTransferIsPricedAsAWholeTransaction(t *testing.T) {
	path := scrollPath(nativeToken())
	path.TxPackedData = []byte{}
	oracle := &oracleRecorder{fee: big.NewInt(500)}

	txFee, approvalFee, err := pathL1Fees(path, l1FeeTestRecipient, oracle.l1Fee)

	require.NoError(t, err)
	require.Equal(t, big.NewInt(500), txFee)
	require.Zero(t, approvalFee.Sign())
	require.Len(t, oracle.inputs, 1)
	require.NotEmpty(t, oracle.inputs[0], "an empty input makes Scroll's oracle answer 0")

	tx := decodeOracleInput(t, oracle.inputs[0])
	require.Equal(t, uint8(ethTypes.DynamicFeeTxType), tx.Type())
	require.Equal(t, new(big.Int).SetUint64(walletCommon.ScrollMainnet), tx.ChainId())
	require.Equal(t, l1FeeTestRecipient, *tx.To())
	require.Equal(t, big.NewInt(1_000_000), tx.Value())
	require.Equal(t, uint64(7), tx.Nonce())
	require.Equal(t, uint64(21_000), tx.Gas())
	require.Equal(t, big.NewInt(2_000_000), tx.GasFeeCap())
	require.Equal(t, big.NewInt(1_000), tx.GasTipCap())
	require.Empty(t, tx.Data())
}

func TestPathL1Fees_TokenTransferGoesToTheTokenContract(t *testing.T) {
	path := scrollPath(erc20Token())
	calldata := append([]byte{0xa9, 0x05, 0x9c, 0xbb}, make([]byte, 64)...) // transfer(address,uint256)
	path.TxPackedData = calldata
	oracle := &oracleRecorder{fee: big.NewInt(500)}

	_, _, err := pathL1Fees(path, l1FeeTestRecipient, oracle.l1Fee)

	require.NoError(t, err)
	tx := decodeOracleInput(t, oracle.inputs[0])
	require.Equal(t, l1FeeTestToken, *tx.To())
	require.Zero(t, tx.Value().Sign(), "a token transfer carries no native value")
	require.Equal(t, calldata, tx.Data())
	require.Greater(t, len(oracle.inputs[0]), len(calldata))
}

func TestPathL1Fees_ContractCallGoesToTheUsedContract(t *testing.T) {
	path := scrollPath(nativeToken())
	path.ProcessorName = pathProcessorCommon.ProcessorLiFiName
	path.UsedContractAddress = &l1FeeTestSpender
	path.TxPackedData = []byte{0xde, 0xad, 0xbe, 0xef}
	oracle := &oracleRecorder{fee: big.NewInt(500)}

	_, _, err := pathL1Fees(path, l1FeeTestRecipient, oracle.l1Fee)

	require.NoError(t, err)
	tx := decodeOracleInput(t, oracle.inputs[0])
	require.Equal(t, l1FeeTestSpender, *tx.To())
	require.Equal(t, big.NewInt(1_000_000), tx.Value())
}

func TestPathL1Fees_ContractDeploymentHasNoRecipient(t *testing.T) {
	path := scrollPath(nativeToken())
	path.ProcessorName = pathProcessorCommon.ProcessorCommunityDeployAssetsName
	path.TxPackedData = []byte{0x60, 0x80}
	oracle := &oracleRecorder{fee: big.NewInt(500)}

	_, _, err := pathL1Fees(path, l1FeeTestRecipient, oracle.l1Fee)

	require.NoError(t, err)
	require.Nil(t, decodeOracleInput(t, oracle.inputs[0]).To())
}

func TestPathL1Fees_ApprovalIsPricedAsItsOwnTransaction(t *testing.T) {
	path := scrollPath(erc20Token())
	path.ProcessorName = pathProcessorCommon.ProcessorLiFiName
	path.UsedContractAddress = &l1FeeTestSpender
	path.TxPackedData = []byte{0xde, 0xad, 0xbe, 0xef}
	approvalData, err := walletCommon.PackApprovalInputData(big.NewInt(1_000_000), &l1FeeTestSpender)
	require.NoError(t, err)
	approvalNonce := hexutil.Uint64(6)
	path.ApprovalRequired = true
	path.ApprovalPackedData = approvalData
	path.ApprovalTxNonce = &approvalNonce
	path.ApprovalGasAmount = 48_300
	path.ApprovalMaxFeesPerGas = bigHex(3_000_000)
	path.ApprovalPriorityFee = bigHex(2_000)
	oracle := &oracleRecorder{fee: big.NewInt(500)}

	txFee, approvalFee, err := pathL1Fees(path, l1FeeTestRecipient, oracle.l1Fee)

	require.NoError(t, err)
	require.Equal(t, big.NewInt(500), txFee)
	require.Equal(t, big.NewInt(500), approvalFee)
	require.Len(t, oracle.inputs, 2)

	var approval *ethTypes.Transaction
	for _, input := range oracle.inputs {
		if tx := decodeOracleInput(t, input); tx.Nonce() == 6 {
			approval = tx
		}
	}
	require.NotNil(t, approval, "the approval must be priced with its own nonce")
	require.Equal(t, l1FeeTestToken, *approval.To())
	require.Zero(t, approval.Value().Sign())
	require.Equal(t, approvalData, approval.Data())
	require.Equal(t, uint64(48_300), approval.Gas())
	require.Equal(t, big.NewInt(3_000_000), approval.GasFeeCap())
	require.Equal(t, big.NewInt(2_000), approval.GasTipCap())
}

func TestPathL1Fees_LegacyChainIsPricedAsALegacyTransaction(t *testing.T) {
	path := scrollPath(nativeToken())
	path.FromChain.EIP1559Enabled = false
	path.TxMaxFeesPerGas = nil
	path.TxPriorityFee = nil
	path.TxGasPrice = bigHex(5_000)
	oracle := &oracleRecorder{fee: big.NewInt(500)}

	_, _, err := pathL1Fees(path, l1FeeTestRecipient, oracle.l1Fee)

	require.NoError(t, err)
	tx := decodeOracleInput(t, oracle.inputs[0])
	require.Equal(t, uint8(ethTypes.LegacyTxType), tx.Type())
	require.Equal(t, big.NewInt(5_000), tx.GasPrice())
}

func TestPathL1Fees_ChainWithoutL1FeeDoesNotAskTheOracle(t *testing.T) {
	path := scrollPath(nativeToken())
	path.FromChain.ChainID = walletCommon.EthereumMainnet
	oracle := &oracleRecorder{fee: big.NewInt(500)}

	txFee, approvalFee, err := pathL1Fees(path, l1FeeTestRecipient, oracle.l1Fee)

	require.NoError(t, err)
	require.Zero(t, txFee.Sign())
	require.Zero(t, approvalFee.Sign())
	require.Empty(t, oracle.inputs)
}

func TestPathL1Fees_OracleErrorIsReturned(t *testing.T) {
	oracle := &oracleRecorder{err: errors.New("rpc down")}

	_, _, err := pathL1Fees(scrollPath(nativeToken()), l1FeeTestRecipient, oracle.l1Fee)

	require.ErrorIs(t, err, oracle.err)
}
