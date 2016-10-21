// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package abi

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// The ABI holds information about a contract's context and available
// invokable methods. It will allow you to type check function calls and
// packs data accordingly.
type ABI struct {
	Constructor Method
	Methods     map[string]Method
	Events      map[string]Event
}

// JSON returns a parsed ABI interface and error if it failed.
func JSON(reader io.Reader) (ABI, error) {
	dec := json.NewDecoder(reader)

	var abi ABI
	if err := dec.Decode(&abi); err != nil {
		return ABI{}, err
	}

	return abi, nil
}

// Pack the given method name to conform the ABI. Method call's data
// will consist of method_id, args0, arg1, ... argN. Method id consists
// of 4 bytes and arguments are all 32 bytes.
// Method ids are created from the first 4 bytes of the hash of the
// methods string signature. (signature = baz(uint32,string32))
func (abi ABI) Pack(name string, args ...interface{}) ([]byte, error) {
	// Fetch the ABI of the requested method
	var method Method

	if name == "" {
		method = abi.Constructor
	} else {
		m, exist := abi.Methods[name]
		if !exist {
			return nil, fmt.Errorf("method '%s' not found", name)
		}
		method = m
	}
	arguments, err := method.pack(method, args...)
	if err != nil {
		return nil, err
	}
	// Pack up the method ID too if not a constructor and return
	if name == "" {
		return arguments, nil
	}
	return append(method.Id(), arguments...), nil
}

// toGoSliceType parses the input and casts it to the proper slice defined by the ABI
// argument in T.
func toGoSlice(i int, t Argument, output []byte) (interface{}, error) {
	index := i * 32
	// The slice must, at very least be large enough for the index+32 which is exactly the size required
	// for the [offset in output, size of offset].
	if index+32 > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go slice: insufficient size output %d require %d", len(output), index+32)
	}
	elem := t.Type.Elem

	// first we need to create a slice of the type
	var refSlice reflect.Value
	switch elem.T {
	case IntTy, UintTy, BoolTy: // int, uint, bool can all be of type big int.
		refSlice = reflect.ValueOf([]*big.Int(nil))
	case AddressTy: // address must be of slice Address
		refSlice = reflect.ValueOf([]common.Address(nil))
	case HashTy: // hash must be of slice hash
		refSlice = reflect.ValueOf([]common.Hash(nil))
	case FixedBytesTy:
		refSlice = reflect.ValueOf([]byte(nil))
	default: // no other types are supported
		return nil, fmt.Errorf("abi: unsupported slice type %v", elem.T)
	}
	// get the offset which determines the start of this array ...
	offset := int(common.BytesToBig(output[index : index+32]).Uint64())
	if offset+32 > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go slice: offset %d would go over slice boundary (len=%d)", len(output), offset+32)
	}

	slice := output[offset:]
	// ... starting with the size of the array in elements ...
	size := int(common.BytesToBig(slice[:32]).Uint64())
	slice = slice[32:]
	// ... and make sure that we've at the very least the amount of bytes
	// available in the buffer.
	if size*32 > len(slice) {
		return nil, fmt.Errorf("abi: cannot marshal in to go slice: insufficient size output %d require %d", len(output), offset+32+size*32)
	}

	// reslice to match the required size
	slice = slice[:(size * 32)]
	for i := 0; i < size; i++ {
		var (
			inter        interface{}             // interface type
			returnOutput = slice[i*32 : i*32+32] // the return output
		)

		// set inter to the correct type (cast)
		switch elem.T {
		case IntTy, UintTy:
			inter = common.BytesToBig(returnOutput)
		case BoolTy:
			inter = common.BytesToBig(returnOutput).Uint64() > 0
		case AddressTy:
			inter = common.BytesToAddress(returnOutput)
		case HashTy:
			inter = common.BytesToHash(returnOutput)
		}
		// append the item to our reflect slice
		refSlice = reflect.Append(refSlice, reflect.ValueOf(inter))
	}

	// return the interface
	return refSlice.Interface(), nil
}

// toGoType parses the input and casts it to the proper type defined by the ABI
// argument in T.
func toGoType(i int, t Argument, output []byte) (interface{}, error) {
	// we need to treat slices differently
	if (t.Type.IsSlice || t.Type.IsArray) && t.Type.T != BytesTy && t.Type.T != StringTy && t.Type.T != FixedBytesTy {
		return toGoSlice(i, t, output)
	}

	index := i * 32
	if index+32 > len(output) {
		return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), index+32)
	}

	// Parse the given index output and check whether we need to read
	// a different offset and length based on the type (i.e. string, bytes)
	var returnOutput []byte
	switch t.Type.T {
	case StringTy, BytesTy: // variable arrays are written at the end of the return bytes
		// parse offset from which we should start reading
		offset := int(common.BytesToBig(output[index : index+32]).Uint64())
		if offset+32 > len(output) {
			return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), offset+32)
		}
		// parse the size up until we should be reading
		size := int(common.BytesToBig(output[offset : offset+32]).Uint64())
		if offset+32+size > len(output) {
			return nil, fmt.Errorf("abi: cannot marshal in to go type: length insufficient %d require %d", len(output), offset+32+size)
		}

		// get the bytes for this return value
		returnOutput = output[offset+32 : offset+32+size]
	default:
		returnOutput = output[index : index+32]
	}

	// convert the bytes to whatever is specified by the ABI.
	switch t.Type.T {
	case IntTy, UintTy:
		bigNum := common.BytesToBig(returnOutput)

		// If the type is a integer convert to the integer type
		// specified by the ABI.
		switch t.Type.Kind {
		case reflect.Uint8:
			return uint8(bigNum.Uint64()), nil
		case reflect.Uint16:
			return uint16(bigNum.Uint64()), nil
		case reflect.Uint32:
			return uint32(bigNum.Uint64()), nil
		case reflect.Uint64:
			return uint64(bigNum.Uint64()), nil
		case reflect.Int8:
			return int8(bigNum.Int64()), nil
		case reflect.Int16:
			return int16(bigNum.Int64()), nil
		case reflect.Int32:
			return int32(bigNum.Int64()), nil
		case reflect.Int64:
			return int64(bigNum.Int64()), nil
		case reflect.Ptr:
			return bigNum, nil
		}
	case BoolTy:
		return common.BytesToBig(returnOutput).Uint64() > 0, nil
	case AddressTy:
		return common.BytesToAddress(returnOutput), nil
	case HashTy:
		return common.BytesToHash(returnOutput), nil
	case BytesTy, FixedBytesTy:
		return returnOutput, nil
	case StringTy:
		return string(returnOutput), nil
	}
	return nil, fmt.Errorf("abi: unknown type %v", t.Type.T)
}

// these variable are used to determine certain types during type assertion for
// assignment.
var (
	r_interSlice = reflect.TypeOf([]interface{}{})
	r_hash       = reflect.TypeOf(common.Hash{})
	r_bytes      = reflect.TypeOf([]byte{})
	r_byte       = reflect.TypeOf(byte(0))
)

// Unpack output in v according to the abi specification
func (abi ABI) Unpack(v interface{}, name string, output []byte) error {
	var method = abi.Methods[name]

	if len(output) == 0 {
		return fmt.Errorf("abi: unmarshalling empty output")
	}

	// make sure the passed value is a pointer
	valueOf := reflect.ValueOf(v)
	if reflect.Ptr != valueOf.Kind() {
		return fmt.Errorf("abi: Unpack(non-pointer %T)", v)
	}

	var (
		value = valueOf.Elem()
		typ   = value.Type()
	)

	if len(method.Outputs) > 1 {
		switch value.Kind() {
		// struct will match named return values to the struct's field
		// names
		case reflect.Struct:
			for i := 0; i < len(method.Outputs); i++ {
				marshalledValue, err := toGoType(i, method.Outputs[i], output)
				if err != nil {
					return err
				}
				reflectValue := reflect.ValueOf(marshalledValue)

				for j := 0; j < typ.NumField(); j++ {
					field := typ.Field(j)
					// TODO read tags: `abi:"fieldName"`
					if field.Name == strings.ToUpper(method.Outputs[i].Name[:1])+method.Outputs[i].Name[1:] {
						if err := set(value.Field(j), reflectValue, method.Outputs[i]); err != nil {
							return err
						}
					}
				}
			}
		case reflect.Slice:
			if !value.Type().AssignableTo(r_interSlice) {
				return fmt.Errorf("abi: cannot marshal tuple in to slice %T (only []interface{} is supported)", v)
			}

			// if the slice already contains values, set those instead of the interface slice itself.
			if value.Len() > 0 {
				if len(method.Outputs) > value.Len() {
					return fmt.Errorf("abi: cannot marshal in to slices of unequal size (require: %v, got: %v)", len(method.Outputs), value.Len())
				}

				for i := 0; i < len(method.Outputs); i++ {
					marshalledValue, err := toGoType(i, method.Outputs[i], output)
					if err != nil {
						return err
					}
					reflectValue := reflect.ValueOf(marshalledValue)
					if err := set(value.Index(i).Elem(), reflectValue, method.Outputs[i]); err != nil {
						return err
					}
				}
				return nil
			}

			// create a new slice and start appending the unmarshalled
			// values to the new interface slice.
			z := reflect.MakeSlice(typ, 0, len(method.Outputs))
			for i := 0; i < len(method.Outputs); i++ {
				marshalledValue, err := toGoType(i, method.Outputs[i], output)
				if err != nil {
					return err
				}
				z = reflect.Append(z, reflect.ValueOf(marshalledValue))
			}
			value.Set(z)
		default:
			return fmt.Errorf("abi: cannot unmarshal tuple in to %v", typ)
		}

	} else {
		marshalledValue, err := toGoType(0, method.Outputs[0], output)
		if err != nil {
			return err
		}
		if err := set(value, reflect.ValueOf(marshalledValue), method.Outputs[0]); err != nil {
			return err
		}
	}

	return nil
}

func (abi *ABI) UnmarshalJSON(data []byte) error {
	var fields []struct {
		Type     string
		Name     string
		Constant bool
		Indexed  bool
		Inputs   []Argument
		Outputs  []Argument
	}

	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	abi.Methods = make(map[string]Method)
	abi.Events = make(map[string]Event)
	for _, field := range fields {
		switch field.Type {
		case "constructor":
			abi.Constructor = Method{
				Inputs: field.Inputs,
			}
		// empty defaults to function according to the abi spec
		case "function", "":
			abi.Methods[field.Name] = Method{
				Name:    field.Name,
				Const:   field.Constant,
				Inputs:  field.Inputs,
				Outputs: field.Outputs,
			}
		case "event":
			abi.Events[field.Name] = Event{
				Name:   field.Name,
				Inputs: field.Inputs,
			}
		}
	}

	return nil
}
