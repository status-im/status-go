package abispec

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var methodPattern = regexp.MustCompile(`(^[a-zA-Z].*)\((.*)\)`)

var maxSafeInteger = big.NewInt(int64(9007199254740991))

const transferInputs = `[{"type":"address"},{"type":"uint256"}]`
const transferFunctionName = "transfer"

var transferInDef = fmt.Sprintf(`[{ "name" : "%s", "type": "function", "inputs": %s}]`, transferFunctionName, transferInputs)
var transferAbi, _ = abi.JSON(strings.NewReader(transferInDef))

func EncodeTransfer(to string, value string) (string, error) {
	amount, success := big.NewInt(0).SetString(value, 0)
	if !success {
		return "", fmt.Errorf("failed to convert %s to big.Int", value)
	}
	address := common.HexToAddress(to)
	result, err := transferAbi.Pack(transferFunctionName, address, amount)
	if err != nil {
		return "", fmt.Errorf("pack failed: %v", err)
	}
	return "0x" + hex.EncodeToString(result), nil
}

func Encode(method string, paramsJSON string) (string, error) {
	matches := methodPattern.FindStringSubmatch(method)
	if len(matches) != 3 {
		return "", fmt.Errorf("unrecognized method: %s", method)
	}
	methodName := matches[1]
	paramTypesString := strings.TrimSpace(matches[2])

	// value of inputs looks like: `[{ "type": "uint32" },{ "type": "bool" }]`
	inputs := "["
	var params []interface{}
	if len(paramTypesString) > 0 {
		var paramsRaw []json.RawMessage
		if err := json.Unmarshal([]byte(paramsJSON), &paramsRaw); err != nil {
			return "", fmt.Errorf("unable to unmarshal params: %v", err)
		}
		types := strings.Split(paramTypesString, ",")
		if len(paramsRaw) != len(types) {
			return "", fmt.Errorf("num of param type should equal to num of param value, actual value: %d, %d", len(types), len(paramsRaw))
		}

		for i, typeName := range types {
			if i != 0 {
				inputs += ","
			}
			inputs += fmt.Sprintf(`{"type":"%s"}`, typeName)

			param, err := toGoTypeValue(typeName, paramsRaw[i])
			if err != nil {
				return "", err
			}
			params = append(params, param.Interface())
		}
	}
	inputs += "]"

	inDef := fmt.Sprintf(`[{ "name" : "%s", "type": "function", "inputs": %s}]`, methodName, inputs)
	inAbi, err := abi.JSON(strings.NewReader(inDef))
	if err != nil {
		return "", fmt.Errorf("invalid ABI definition %s, %v", inDef, err)
	}
	var result []byte
	result, err = inAbi.Pack(methodName, params...)

	if err != nil {
		return "", fmt.Errorf("Pack failed: %v", err)
	}

	return "0x" + hex.EncodeToString(result), nil
}

// override result to make it looks like what status-mobile need
func overrideResult(out []interface{}) []interface{} {
	for i, v := range out {
		outType := reflect.TypeOf(v)
		switch outType.String() {
		case "[]uint8":
			out[i] = "0x" + common.Bytes2Hex(v.([]uint8))
		case bigIntType:
			vv := v.(*big.Int)
			if vv.Cmp(maxSafeInteger) == 1 {
				out[i] = vv.String()
			}
		}

		if outType.Kind() == reflect.Array && outType.Elem().Kind() == reflect.Array && outType.Elem().Elem().Kind() == reflect.Uint8 { //case e.g. [2][3]uint8
			val := reflect.ValueOf(v)
			rowNum := val.Len()
			colNum := val.Index(0).Len()
			var ss = make([]string, rowNum)
			for i := 0; i < rowNum; i++ {
				bytes := make([]uint8, colNum)
				for j := 0; j < colNum; j++ {
					bytes[j] = uint8(val.Index(i).Index(j).Uint())
				}
				ss[i] = common.Bytes2Hex(bytes)
			}
			out[i] = ss
		} else if outType.String() != "common.Address" && outType.Kind() == reflect.Array && outType.Elem().Kind() == reflect.Uint8 {
			val := reflect.ValueOf(v)
			len := val.Len()
			bytes := make([]uint8, len)
			for i := 0; i < len; i++ {
				bytes[i] = uint8(val.Index(i).Uint())
			}
			out[i] = common.Bytes2Hex(bytes)
		}

	}
	return out
}

// bytesString e.g. 0x000000000000000000000000000000000000000000000000000000005bc741cd00000000000000000000000000000000000000000000000000000000000000a000000000000000000000000013b86dbf1a83c9e6a492914a0ee39e8a5b7eb60700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002e516d533152484e4a57414b356e426f6f57454d34654d644268707a35666e325764557473457357754a4b79356147000000000000000000000000000000000000
// types e.g. []string{"uint256","bytes","address","uint256","uint256"}
func Decode(bytesString string, types []string) ([]interface{}, error) {
	outputs := "["
	for i, typeName := range types {
		if i != 0 {
			outputs += ","
		}
		outputs += fmt.Sprintf(`{"type":"%s"}`, typeName)
	}
	outputs += "]"
	def := fmt.Sprintf(`[{ "name" : "method", "type": "function", "outputs": %s}]`, outputs)
	abi, err := abi.JSON(strings.NewReader(def))
	if err != nil {
		return nil, fmt.Errorf("invalid ABI definition %s: %v", def, err)
	}

	bytesString = strings.TrimPrefix(bytesString, "0x")

	bytes, err := hex.DecodeString(bytesString)
	if err != nil {
		return nil, fmt.Errorf("invalid hex %s: %v", bytesString, err)
	}
	out, err := abi.Unpack("method", bytes)
	if err != nil {
		return nil, fmt.Errorf("unpack failed: %v", err)
	}

	return overrideResult(out), nil
}
