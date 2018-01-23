// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

// Package value defines an interface for user-defined types with value
// semantics to implement in order to be compatible with the rest of the
// Arista Go infrastructure.
package value

import (
	"encoding/json"
	"fmt"
)

// Value is the interface that all types with value semantics must implement
// in order to be compatible with the Entity infrastructure and streaming
// protocols we support.  By default all value types are just represented as
// a map[string]interface{}, but when a TypeMapper is used to remap the value
// to a user-defined type (as opposed to a built-in type), then these user
// defined types must fulfill the contract defined by this interface.
//
// Types that implement this interface must have value semantics, meaning they
// are not allowed to contain anything with pointer semantics such as slices,
// maps, channels, etc.  They must be directly usable as keys in maps and
// behave properly when compared with the built-in == operator.
type Value interface {
	fmt.Stringer
	json.Marshaler

	// ToBuiltin returns the best possible representation of this type as one
	// of the built-in types we support.  Most often this means returning the
	// string representation of this type (for example for an IP address or an
	// IP prefix), but sometimes not (e.g. a VLAN ID is better represented as
	// uint16).
	ToBuiltin() interface{}
}
