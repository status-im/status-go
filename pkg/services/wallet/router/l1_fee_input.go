package router

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"

	gaspriceproxy "github.com/status-im/status-go/internal/contracts/gas-price-proxy"
	pathProcessorCommon "github.com/status-im/status-go/pkg/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/pkg/services/wallet/router/routes"
)

// l1FeeFunc asks the L1 fee oracle of the chain to price a serialized transaction.
type l1FeeFunc func(chainID uint64, tx []byte) (*big.Int, error)

// pathL1Fees returns the L1 data fees of the path's transaction and of its approval.
// Both are zero on a chain that charges no L1 data fee.
func pathL1Fees(path *routes.Path, recipient common.Address, l1Fee l1FeeFunc) (txFee *big.Int, approvalFee *big.Int, err error) {
	txFee = big.NewInt(0)
	approvalFee = big.NewInt(0)

	chainID := path.FromChain.ChainID
	if !gaspriceproxy.ChargesL1Fee(chainID) {
		return txFee, approvalFee, nil
	}

	if path.ApprovalRequired {
		input, err := unsignedApprovalTx(path)
		if err != nil {
			return nil, nil, err
		}
		approvalFee, err = l1Fee(chainID, input)
		if err != nil {
			return nil, nil, err
		}
	}

	input, err := unsignedMainTx(path, recipient)
	if err != nil {
		return nil, nil, err
	}
	txFee, err = l1Fee(chainID, input)
	if err != nil {
		return nil, nil, err
	}

	return txFee, approvalFee, nil
}

// unsignedMainTx serializes the path's transaction without a signature. It has the size and the
// calldata of the one that will be sent; for a swap the address is the spender's, not the router's.
func unsignedMainTx(path *routes.Path, recipient common.Address) ([]byte, error) {
	value := big.NewInt(0)
	if path.FromToken.IsNative() && path.AmountIn != nil {
		value = path.AmountIn.ToInt()
	}

	return serializeUnsignedTx(path.FromChain.ChainID, path.FromChain.EIP1559Enabled, unsignedTxFields{
		to:          mainTxTo(path, recipient),
		value:       value,
		data:        path.TxPackedData,
		nonce:       path.TxNonce,
		gas:         path.TxGasAmount,
		gasPrice:    path.TxGasPrice,
		maxFee:      path.TxMaxFeesPerGas,
		priorityFee: path.TxPriorityFee,
	})
}

// unsignedApprovalTx serializes the path's approval, a call to the token contract.
func unsignedApprovalTx(path *routes.Path) ([]byte, error) {
	token := path.FromToken.Address

	return serializeUnsignedTx(path.FromChain.ChainID, path.FromChain.EIP1559Enabled, unsignedTxFields{
		to:          &token,
		value:       big.NewInt(0),
		data:        path.ApprovalPackedData,
		nonce:       path.ApprovalTxNonce,
		gas:         path.ApprovalGasAmount,
		gasPrice:    path.ApprovalGasPrice,
		maxFee:      path.ApprovalMaxFeesPerGas,
		priorityFee: path.ApprovalPriorityFee,
	})
}

// mainTxTo returns the address the path's transaction is sent to, nil for a contract deployment.
func mainTxTo(path *routes.Path, recipient common.Address) *common.Address {
	switch path.ProcessorName {
	case pathProcessorCommon.ProcessorCommunityDeployCollectiblesName,
		pathProcessorCommon.ProcessorCommunityDeployAssetsName:
		return nil
	}

	if path.UsedContractAddress != nil && *path.UsedContractAddress != (common.Address{}) {
		return path.UsedContractAddress
	}
	if !path.FromToken.IsNative() {
		token := path.FromToken.Address
		return &token
	}
	return &recipient
}

type unsignedTxFields struct {
	to          *common.Address
	value       *big.Int
	data        []byte
	nonce       *hexutil.Uint64
	gas         uint64
	gasPrice    *hexutil.Big
	maxFee      *hexutil.Big
	priorityFee *hexutil.Big
}

func serializeUnsignedTx(chainID uint64, eip1559 bool, fields unsignedTxFields) ([]byte, error) {
	nonce := uint64(0)
	if fields.nonce != nil {
		nonce = uint64(*fields.nonce)
	}

	var txData ethTypes.TxData
	if eip1559 {
		txData = &ethTypes.DynamicFeeTx{
			ChainID:   new(big.Int).SetUint64(chainID),
			Nonce:     nonce,
			GasTipCap: bigOrZero(fields.priorityFee),
			GasFeeCap: bigOrZero(fields.maxFee),
			Gas:       fields.gas,
			To:        fields.to,
			Value:     fields.value,
			Data:      fields.data,
		}
	} else {
		txData = &ethTypes.LegacyTx{
			Nonce:    nonce,
			GasPrice: bigOrZero(fields.gasPrice),
			Gas:      fields.gas,
			To:       fields.to,
			Value:    fields.value,
			Data:     fields.data,
		}
	}

	return ethTypes.NewTx(txData).MarshalBinary()
}

func bigOrZero(v *hexutil.Big) *big.Int {
	if v == nil {
		return big.NewInt(0)
	}
	return v.ToInt()
}
