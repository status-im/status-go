// Copyright (C) 2015  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/aristanetworks/goarista/value"
)

// StringifyInterface transforms an arbitrary interface into its string
// representation.  We need to do this because some entities use the string
// representation of their keys as their names.
// Note: this API is deprecated and will be removed.
func StringifyInterface(key interface{}) (string, error) {
	if key == nil {
		return "", errors.New("Unable to stringify nil")
	}
	var str string
	switch key := key.(type) {
	case bool:
		str = strconv.FormatBool(key)
	case uint8:
		str = strconv.FormatUint(uint64(key), 10)
	case uint16:
		str = strconv.FormatUint(uint64(key), 10)
	case uint32:
		str = strconv.FormatUint(uint64(key), 10)
	case uint64:
		str = strconv.FormatUint(key, 10)
	case int8:
		str = strconv.FormatInt(int64(key), 10)
	case int16:
		str = strconv.FormatInt(int64(key), 10)
	case int32:
		str = strconv.FormatInt(int64(key), 10)
	case int64:
		str = strconv.FormatInt(key, 10)
	case float32:
		str = "f" + strconv.FormatInt(int64(math.Float32bits(key)), 10)
	case float64:
		str = "f" + strconv.FormatInt(int64(math.Float64bits(key)), 10)
	case string:
		str = key
		for i := 0; i < len(str); i++ {
			if chr := str[i]; chr < 0x20 || chr > 0x7E {
				str = strconv.QuoteToASCII(str)
				str = str[1 : len(str)-1] // Drop the leading and trailing quotes.
				break
			}
		}
	case map[string]interface{}:
		keys := SortedKeys(key)
		for i, k := range keys {
			v := key[k]
			keys[i] = stringify(v)
		}
		str = strings.Join(keys, "_")
	case *map[string]interface{}:
		return StringifyInterface(*key)
	case map[Key]interface{}:
		m := make(map[string]interface{}, len(key))
		for k, v := range key {
			m[k.String()] = v
		}
		keys := SortedKeys(m)
		for i, k := range keys {
			keys[i] = stringify(k) + "=" + stringify(m[k])
		}
		str = strings.Join(keys, "_")

	case value.Value:
		return key.String(), nil

	default:
		panic(fmt.Errorf("Unable to stringify type %T: %#v", key, key))
	}

	return str, nil
}

func stringify(key interface{}) string {
	s, err := StringifyInterface(key)
	if err != nil {
		panic(err)
	}
	return s
}
