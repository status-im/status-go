package typeddata

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func deps(target string, types Types) []string {
	unique := map[string]struct{}{}
	unique[target] = struct{}{}
	visited := []string{target}
	deps := []string{}
	for len(visited) > 0 {
		current := visited[0]
		fields := types[current]
		for i := range fields {
			f := fields[i]
			if _, defined := types[f.Type]; defined {
				if _, exist := unique[f.Type]; !exist {
					visited = append(visited, f.Type)
					unique[f.Type] = struct{}{}
				}
			}
		}
		visited = visited[1:]
		deps = append(deps, current)
	}
	sort.Slice(deps[1:], func(i, j int) bool {
		return deps[1:][i] < deps[1:][j]
	})
	return deps
}

func typeString(target string, types Types) string {
	b := new(bytes.Buffer)
	for _, dep := range deps(target, types) {
		b.WriteString(dep)
		b.WriteString("(")
		fields := types[dep]
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
		}
		b.WriteString(")")
	}
	return b.String()
}

func typeHash(target string, types Types) (rst common.Hash) {
	return crypto.Keccak256Hash([]byte(typeString(target, types)))
}

func encodeData(target string, data map[string]interface{}, types Types) (rst common.Hash, err error) {
	fields := types[target]
	typeh := typeHash(target, types)
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
			str, ok := data[f.Name].(string)
			if !ok {
				return rst, fmt.Errorf("%v is not a string", data[f.Name])
			}
			val = crypto.Keccak256Hash([]byte(str))
		} else if f.Type == "bytes" {
			typ = bytes32
			bytes, ok := data[f.Name].([]byte)
			if !ok {
				return rst, fmt.Errorf("%v is not a byte slice", data[f.Name])
			}
			val = crypto.Keccak256Hash(bytes)
		} else if _, exist := types[f.Type]; exist {
			obj, ok := data[f.Name].(map[string]interface{})
			if !ok {
				return rst, fmt.Errorf("%v is not an object", data[f.Name])
			}
			val, err = encodeData(f.Type, obj, types)
			if err != nil {
				return
			}
			typ = bytes32
		} else {
			typ, err = abi.NewType(f.Type)
			if err != nil {
				return
			}
			if typ.T == abi.SliceTy || typ.T == abi.ArrayTy || typ.T == abi.FunctionTy {
				err = errors.New("arrays, slices and functions are not supported")
				return
			}
			val = data[f.Name]
			if typ.T == abi.AddressTy {
				strval, ok := val.(string)
				if !ok {
					err = fmt.Errorf("can't cast %v to a string", val)
				}
				val = common.HexToAddress(strval)
			}
			if typ.T == abi.IntTy || typ.T == abi.UintTy {
				val, err = castInteger(typ, val)
				if err != nil {
					return
				}
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

func castInteger(typ abi.Type, val interface{}) (interface{}, error) {
	if typ.Kind == reflect.Ptr {
		return castToBig(typ, val)
	}
	if typ.T == abi.IntTy {
		return castToInt(typ, val)
	}
	if typ.T == abi.UintTy {
		return castToUint(typ, val)
	}
	return nil, fmt.Errorf("value %d of type %v is not an integer", val, typ)
}

func castToInt(typ abi.Type, val interface{}) (rst interface{}, err error) {
	intval, ok := val.(int)
	if ok {
		switch typ.Size {
		case 8:
			rst = int8(intval)
		case 16:
			rst = int16(intval)
		case 32:
			rst = int32(intval)
		case 64:
			rst = int64(intval)
		}
	}
	if !ok {
		err = fmt.Errorf("can't cast %v to int%d", val, typ.Size)
	}
	return
}

func castToUint(typ abi.Type, val interface{}) (rst interface{}, err error) {
	intval, ok := val.(uint)
	if ok {
		switch typ.Size {
		case 8:
			rst = uint8(intval)
		case 16:
			rst = uint16(intval)
		case 32:
			rst = uint32(intval)
		case 64:
			rst = uint64(intval)
		}
	}
	if !ok {
		err = fmt.Errorf("can't cast %v to uint%d", val, typ.Size)
	}
	return
}

func castToBig(typ abi.Type, val interface{}) (interface{}, error) {
	strval, ok := val.(string)
	if !ok {
		// fallback to integers
		intval, ok := val.(int)
		if !ok {
			return nil, fmt.Errorf("can't cast %v to an integer", val)
		}
		val = new(big.Int).SetInt64(int64(intval))
		return val, nil
	}
	val, ok = new(big.Int).SetString(strval, 0)
	if !ok {
		return nil, fmt.Errorf("failed to set big.Int from string value %s", strval)
	}
	return val, nil
}
