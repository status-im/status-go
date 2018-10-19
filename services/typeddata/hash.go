package typeddata

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func typeString(target string, types Types) (string, error) {
	// FIX composite types after primary must be sorted alphabetically
	unique := map[string]struct{}{}
	unique[target] = struct{}{}
	deps := []string{target}
	b := new(bytes.Buffer)
	for len(deps) > 0 {
		current := deps[0]
		b.WriteString(current)
		b.WriteString("(")
		fields, defined := types[current]
		if !defined {
			return "", fmt.Errorf("type `%s` is not defined in `types`: %v", current, types)
		}
		first := true
		for i := range fields {
			if !first {
				b.WriteString(",")
			} else {
				first = false
			}
			f := fields[i]
			b.WriteString(f.Type)
			b.WriteString(" ")
			b.WriteString(f.Name)
			if _, defined := types[f.Type]; defined {
				if _, exist := unique[f.Type]; !exist {
					deps = append(deps, f.Type)
					unique[f.Type] = struct{}{}
				}
			}
		}
		b.WriteString(")")
		deps = deps[1:]
	}
	return b.String(), nil
}

func typeHash(target string, types Types) (rst common.Hash, err error) {
	data, err := typeString(target, types)
	if err != nil {
		return rst, err
	}
	return crypto.Keccak256Hash([]byte(data)), nil
}

func encodeData(target string, data map[string]interface{}, types Types) (rst common.Hash, err error) {
	fields := types[target]
	typeh, err := typeHash(target, types)
	if err != nil {
		return
	}
	bytes32, err := abi.NewType("bytes32")
	if err != nil {
		return
	}
	args := abi.Arguments{{Type: bytes32}}
	vals := []interface{}{typeh}
	for i := range fields {
		f := fields[i]
		var (
			val interface{}
			typ abi.Type
		)
		if f.Type == "string" {
			typ = bytes32
			val = crypto.Keccak256Hash([]byte(data[f.Name].(string)))
		} else if f.Type == "bytes" {
			typ = bytes32
			val = crypto.Keccak256Hash(data[f.Name].([]byte))
		} else if _, exist := types[f.Type]; exist {
			val, err = encodeData(f.Name, data[f.Name].(map[string]interface{}), types)
			if err != nil {
				return
			}
			typ = bytes32
		} else {
			typ, err = abi.NewType(f.Type)
			if err != nil {
				return
			}
			if typ.T == abi.SliceTy || typ.T == abi.ArrayTy {
				err = errors.New("arrays or slices are not supported")
				return
			}
			val = data[f.Name]
			if typ.T == abi.AddressTy {
				val = common.HexToAddress(val.(string))
			}
			// if size of the integer > 64 - abi expects pointer to a big.Int
			if (typ.T == abi.IntTy || typ.T == abi.UintTy) && typ.Kind == reflect.Ptr {
				val = new(big.Int).SetUint64(uint64(val.(int)))
			}
		}
		vals = append(vals, val)
		args = append(args, abi.Argument{Name: f.Name, Type: typ})
	}
	packed, err := args.Pack(vals...)
	if err != nil {
		return
	}
	return crypto.Keccak256Hash(packed), nil
}
