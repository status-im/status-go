package abispec

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const bigIntType = "*big.Int"

var zero = big.NewInt(0)

var arrayTypePattern = regexp.MustCompile(`(\[([\d]*)\])`)

var bytesType = reflect.TypeOf([]byte{})

var typeMap = map[string]reflect.Type{
	"uint8":   reflect.TypeOf(uint8(0)),
	"int8":    reflect.TypeOf(int8(0)),
	"uint16":  reflect.TypeOf(uint16(0)),
	"int16":   reflect.TypeOf(int16(0)),
	"uint32":  reflect.TypeOf(uint32(0)),
	"int32":   reflect.TypeOf(int32(0)),
	"uint64":  reflect.TypeOf(uint64(0)),
	"int64":   reflect.TypeOf(int64(0)),
	"bytes":   bytesType,
	"bytes1":  reflect.TypeOf([1]byte{}),
	"bytes2":  reflect.TypeOf([2]byte{}),
	"bytes3":  reflect.TypeOf([3]byte{}),
	"bytes4":  reflect.TypeOf([4]byte{}),
	"bytes5":  reflect.TypeOf([5]byte{}),
	"bytes6":  reflect.TypeOf([6]byte{}),
	"bytes7":  reflect.TypeOf([7]byte{}),
	"bytes8":  reflect.TypeOf([8]byte{}),
	"bytes9":  reflect.TypeOf([9]byte{}),
	"bytes10": reflect.TypeOf([10]byte{}),
	"bytes11": reflect.TypeOf([11]byte{}),
	"bytes12": reflect.TypeOf([12]byte{}),
	"bytes13": reflect.TypeOf([13]byte{}),
	"bytes14": reflect.TypeOf([14]byte{}),
	"bytes15": reflect.TypeOf([15]byte{}),
	"bytes16": reflect.TypeOf([16]byte{}),
	"bytes17": reflect.TypeOf([17]byte{}),
	"bytes18": reflect.TypeOf([18]byte{}),
	"bytes19": reflect.TypeOf([19]byte{}),
	"bytes20": reflect.TypeOf([20]byte{}),
	"bytes21": reflect.TypeOf([21]byte{}),
	"bytes22": reflect.TypeOf([22]byte{}),
	"bytes23": reflect.TypeOf([23]byte{}),
	"bytes24": reflect.TypeOf([24]byte{}),
	"bytes25": reflect.TypeOf([25]byte{}),
	"bytes26": reflect.TypeOf([26]byte{}),
	"bytes27": reflect.TypeOf([27]byte{}),
	"bytes28": reflect.TypeOf([28]byte{}),
	"bytes29": reflect.TypeOf([29]byte{}),
	"bytes30": reflect.TypeOf([30]byte{}),
	"bytes31": reflect.TypeOf([31]byte{}),
	"bytes32": reflect.TypeOf([32]byte{}),
	"address": reflect.TypeOf(common.Address{}),
	"bool":    reflect.TypeOf(false),
	"string":  reflect.TypeOf(""),
}

func toGoType(solidityType string) (reflect.Type, error) {
	if t, ok := typeMap[solidityType]; ok {
		return t, nil
	}

	if arrayTypePattern.MatchString(solidityType) { // type of array
		index := arrayTypePattern.FindStringIndex(solidityType)[0]
		arrayType, err := toGoType(solidityType[0:index])
		if err != nil {
			return nil, err
		}
		matches := arrayTypePattern.FindAllStringSubmatch(solidityType, -1)
		for i := 0; i <= len(matches)-1; i++ {
			sizeStr := matches[i][2]
			if sizeStr == "" {
				arrayType = reflect.SliceOf(arrayType)
			} else {
				length, err := strconv.Atoi(sizeStr)
				if err != nil {
					return nil, err
				}
				arrayType = reflect.ArrayOf(length, arrayType)
			}
		}
		return arrayType, nil
	}

	// uint and int are aliases for uint256 and int256, respectively.
	// source: https://docs.soliditylang.org/en/v0.8.11/types.html
	//TODO should we support type: uint ?? currently, go-ethereum doesn't support type uint
	if strings.HasPrefix(solidityType, "uint") || strings.HasPrefix(solidityType, "int") {
		return reflect.TypeOf(zero), nil
	}

	return nil, fmt.Errorf("unsupported type: %s", solidityType)
}

func toGoTypeValue(solidityType string, raw json.RawMessage) (*reflect.Value, error) {
	goType, err := toGoType(solidityType)
	if err != nil {
		return nil, err
	}

	value := reflect.New(goType)

	if goType == bytesType { // to support case like: Encode("sam(bytes)", `["dave"]`)
		var s string
		err = json.Unmarshal(raw, &s)
		if err != nil {
			return nil, err
		}
		bytes := []byte(s)
		value.Elem().SetBytes(bytes)
		return &value, nil
	}

	err = json.Unmarshal(raw, value.Interface())
	if err != nil {
		if goType.String() == bigIntType {
			var s string
			err = json.Unmarshal(raw, &s)
			if err != nil {
				return nil, err
			}
			v, success := big.NewInt(0).SetString(s, 0)
			if !success {
				return nil, fmt.Errorf("convert to go type value failed, value: %s", s)
			}
			value = reflect.ValueOf(v)

		} else if goType.Kind() == reflect.Array { // to support case like: Encode("f(bytes10)", `["1234567890"]`)
			elemKind := goType.Elem().Kind()
			if elemKind == reflect.Uint8 {
				var s string
				err = json.Unmarshal(raw, &s)
				if err != nil {
					return nil, err
				}
				bytes := []byte(s)
				for i, b := range bytes {
					value.Elem().Index(i).Set(reflect.ValueOf(b))
				}
				return &value, nil
			}

			if elemKind == reflect.Array { // to support case like: Encode("bar(bytes3[2])", `[["abc","def"]]`)
				var ss []string
				err = json.Unmarshal(raw, &ss)
				if err != nil {
					return nil, err
				}

				var bytes [][]byte
				for _, s := range ss {
					bytes = append(bytes, []byte(s))
				}

				// convert []byte to []int
				// note: Array and slice values encode as JSON arrays, except that []byte encodes as a base64-encoded string, and a nil slice encodes as the null JSON object.
				var ints = make([][]int, len(bytes))
				for i, r := range bytes {
					ints[i] = make([]int, len(r))
					for j, b := range r {
						ints[i][j] = int(b)
					}
				}

				jsonString, err := json.Marshal(ints)
				if err != nil {
					return nil, err
				}
				if err = json.Unmarshal(jsonString, value.Interface()); err != nil {
					return nil, err
				}
			}

		}
	}

	return &value, err
}
