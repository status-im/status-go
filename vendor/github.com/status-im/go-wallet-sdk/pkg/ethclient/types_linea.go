package ethclient

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type LineaEstimateGasResult struct {
	BaseFeePerGas     *big.Int `json:"baseFeePerGas"`
	GasLimit          *big.Int `json:"gasLimit"`
	PriorityFeePerGas *big.Int `json:"priorityFeePerGas"`
}

// lineaEstimateGasResultJSON is the internal type used for JSON marshaling/unmarshaling
type lineaEstimateGasResultJSON struct {
	BaseFeePerGas     *hexutil.Big `json:"baseFeePerGas"`
	GasLimit          *hexutil.Big `json:"gasLimit"`
	PriorityFeePerGas *hexutil.Big `json:"priorityFeePerGas"`
}

// MarshalJSON implements json.Marshaler
func (t *LineaEstimateGasResult) MarshalJSON() ([]byte, error) {
	tx := lineaEstimateGasResultJSON{
		BaseFeePerGas:     (*hexutil.Big)(t.BaseFeePerGas),
		GasLimit:          (*hexutil.Big)(t.GasLimit),
		PriorityFeePerGas: (*hexutil.Big)(t.PriorityFeePerGas),
	}
	return json.Marshal(tx)
}

// UnmarshalJSON implements json.Unmarshaler
func (r *LineaEstimateGasResult) UnmarshalJSON(data []byte) error {
	var result lineaEstimateGasResultJSON
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	r.BaseFeePerGas = (*big.Int)(result.BaseFeePerGas)
	r.GasLimit = (*big.Int)(result.GasLimit)
	r.PriorityFeePerGas = (*big.Int)(result.PriorityFeePerGas)
	return nil
}
