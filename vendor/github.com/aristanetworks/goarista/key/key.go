// Copyright (C) 2015  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key

import (
	"encoding/json"
	"fmt"

	"github.com/aristanetworks/goarista/value"
)

// Key represents the Key in the updates and deletes of the Notification
// objects.  The only reason this exists is that Go won't let us define
// our own hash function for non-hashable types, and unfortunately we
// need to be able to index maps by map[string]interface{} objects.
type Key interface {
	Key() interface{}
	String() string
	Equal(other interface{}) bool
}

type keyImpl struct {
	key interface{}
}

// New wraps the given value in a Key.
// This function panics if the value passed in isn't allowed in a Key or
// doesn't implement value.Value.
func New(intf interface{}) Key {
	switch t := intf.(type) {
	case map[string]interface{}:
		return composite{sentinel, t}
	case int8, int16, int32, int64,
		uint8, uint16, uint32, uint64,
		float32, float64, string, bool,
		value.Value:
		return keyImpl{key: intf}
	default:
		panic(fmt.Sprintf("Invalid type for key: %T", intf))
	}
}

func (k keyImpl) Key() interface{} {
	return k.key
}

func (k keyImpl) String() string {
	return stringify(k.key)
}

func (k keyImpl) GoString() string {
	return fmt.Sprintf("key.New(%#v)", k.Key())
}

func (k keyImpl) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.Key())
}

func (k keyImpl) Equal(other interface{}) bool {
	o, ok := other.(Key)
	if !ok {
		return false
	}
	return keyEqual(k.key, o.Key())
}

// Comparable types have an equality-testing method.
type Comparable interface {
	// Equal returns true if this object is equal to the other one.
	Equal(other interface{}) bool
}

func keyEqual(a, b interface{}) bool {
	switch a := a.(type) {
	case map[string]interface{}:
		b, ok := b.(map[string]interface{})
		if !ok || len(a) != len(b) {
			return false
		}
		for k, av := range a {
			if bv, ok := b[k]; !ok || !keyEqual(av, bv) {
				return false
			}
		}
		return true
	case map[Key]interface{}:
		b, ok := b.(map[Key]interface{})
		if !ok || len(a) != len(b) {
			return false
		}
		for k, av := range a {
			if bv, ok := b[k]; !ok || !keyEqual(av, bv) {
				return false
			}
		}
		return true
	case Comparable:
		return a.Equal(b)
	}

	return a == b
}
