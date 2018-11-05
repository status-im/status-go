package typeddata

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	bytes32Type, _ = abi.NewType("bytes32")
	int256Type, _  = abi.NewType("int256")
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

func hashStruct(target string, data map[string]json.RawMessage, types Types) (rst common.Hash, err error) {
	fields := types[target]
	typeh := typeHash(target, types)
	args := abi.Arguments{{Type: bytes32Type}}
	vals := []interface{}{typeh}
	for i := range fields {
		f := fields[i]
		val, typ, err := toABITypeAndValue(f, data, types)
		if err != nil {
			return rst, err
		}
		vals = append(vals, val)
		args = append(args, abi.Argument{Name: f.Name, Type: typ})
	}
	packed, err := args.Pack(vals...)
	if err != nil {
		return rst, err
	}
	return crypto.Keccak256Hash(packed), nil
}

func toABITypeAndValue(f Field, data map[string]json.RawMessage, types Types) (val interface{}, typ abi.Type, err error) {
	if f.Type == "string" {
		var str string
		if err = json.Unmarshal(data[f.Name], &str); err != nil {
			return
		}
		typ = bytes32Type
		val = crypto.Keccak256Hash([]byte(str))
	} else if f.Type == "bytes" {
		typ = bytes32Type
		var bytes hexutil.Bytes
		if err = json.Unmarshal(data[f.Name], &bytes); err != nil {
			return
		}
		val = crypto.Keccak256Hash(bytes)
	} else if _, exist := types[f.Type]; exist {
		var obj map[string]json.RawMessage
		if err = json.Unmarshal(data[f.Name], &obj); err != nil {
			return
		}
		val, err = hashStruct(f.Type, obj, types)
		if err != nil {
			return
		}
		typ = bytes32Type
	} else {
		typ, err = abi.NewType(f.Type)
		if err != nil {
			return
		}
		if typ.T == abi.SliceTy || typ.T == abi.ArrayTy || typ.T == abi.FunctionTy {
			return val, typ, errors.New("arrays, slices and functions are not supported")
		} else if typ.T == abi.FixedBytesTy {
			var bytes hexutil.Bytes
			if err = json.Unmarshal(data[f.Name], &bytes); err != nil {
				return
			}
			typ = bytes32Type
			rst := [32]byte{}
			// reduce the length to the advertised type
			if len(bytes) > typ.Size {
				bytes = bytes[:typ.Size]
			}
			copy(rst[:], bytes)
			val = rst
		} else if typ.T == abi.AddressTy {
			var addr common.Address
			if err = json.Unmarshal(data[f.Name], &addr); err != nil {
				return
			}
			val = addr
		} else if typ.T == abi.IntTy || typ.T == abi.UintTy {
			var big big.Int
			if err = json.Unmarshal(data[f.Name], &big); err != nil {
				return
			}
			typ = int256Type
			val = &big
		} else if typ.T == abi.BoolTy {
			var rst bool
			if err = json.Unmarshal(data[f.Name], &rst); err != nil {
				return
			}
			val = rst
		}
	}
	return
}
