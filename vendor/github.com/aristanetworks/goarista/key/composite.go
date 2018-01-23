// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/aristanetworks/goarista/areflect"
)

// composite allows storing a map[string]interface{} as a key in a Go map.
// This is useful when the key isn't a fixed data structure known at compile
// time but rather something generic, like a bag of key-value pairs.
// Go does not allow storing a map inside the key of a map, because maps are
// not comparable or hashable, and keys in maps must be both.  This file is
// a hack specific to the 'gc' implementation of Go (which is the one most
// people use when they use Go), to bypass this check, by abusing reflection
// to override how Go compares composite for equality or how it's hashed.
// The values allowed in this map are only the types whitelisted in New() as
// well as map[Key]interface{}.
//
// See also https://github.com/golang/go/issues/283
type composite struct {
	// This value must always be set to the sentinel constant above.
	sentinel uintptr
	m        map[string]interface{}
}

func (k composite) Key() interface{} {
	return k.m
}

func (k composite) String() string {
	return stringify(k.Key())
}

func (k composite) GoString() string {
	return fmt.Sprintf("key.New(%#v)", k.Key())
}

func (k composite) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.Key())
}

func (k composite) Equal(other interface{}) bool {
	o, ok := other.(Key)
	if !ok {
		return false
	}
	return keyEqual(k.Key(), o.Key())
}

func hashInterface(v interface{}) uintptr {
	switch v := v.(type) {
	case map[string]interface{}:
		return hashMapString(v)
	case map[Key]interface{}:
		return hashMapKey(v)
	default:
		return _nilinterhash(v)
	}
}

func hashMapString(m map[string]interface{}) uintptr {
	h := uintptr(31 * (len(m) + 1))
	for k, v := range m {
		// Use addition so that the order of iteration doesn't matter.
		h += _strhash(k)
		h += hashInterface(v)
	}
	return h
}

func hashMapKey(m map[Key]interface{}) uintptr {
	h := uintptr(31 * (len(m) + 1))
	for k, v := range m {
		// Use addition so that the order of iteration doesn't matter.
		switch k := k.(type) {
		case keyImpl:
			h += _nilinterhash(k.key)
		case composite:
			h += hashMapString(k.m)
		}
		h += hashInterface(v)
	}
	return h
}

func hash(p unsafe.Pointer, seed uintptr) uintptr {
	ck := *(*composite)(p)
	if ck.sentinel != sentinel {
		panic("use of unhashable type in a map")
	}
	return seed ^ hashMapString(ck.m)
}

func equal(a unsafe.Pointer, b unsafe.Pointer) bool {
	ca := (*composite)(a)
	cb := (*composite)(b)
	if ca.sentinel != sentinel {
		panic("use of uncomparable type on the lhs of ==")
	}
	if cb.sentinel != sentinel {
		panic("use of uncomparable type on the rhs of ==")
	}
	return ca.Equal(*cb)
}

func init() {
	typ := reflect.TypeOf(composite{})
	alg := reflect.ValueOf(typ).Elem().FieldByName("alg").Elem()
	// Pretty certain that doing this voids your warranty.
	// This overwrites the typeAlg of either alg_NOEQ64 (on 32-bit platforms)
	// or alg_NOEQ128 (on 64-bit platforms), which means that all unhashable
	// types that were using this typeAlg are now suddenly hashable and will
	// attempt to use our equal/hash functions, which will lead to undefined
	// behaviors.  But then these types shouldn't have been hashable in the
	// first place, so no one should have attempted to use them as keys in a
	// map.  The compiler will emit an error if it catches someone trying to
	// do this, but if they do it through a map that uses an interface type as
	// the key, then the compiler can't catch it.
	// To prevent this we could instead override the alg pointer in the type,
	// but it's in a read-only data section in the binary (it's put there by
	// dcommontype() in gc/reflect.go), so changing it is also not without
	// perils.  Basically: Here Be Dragons.
	areflect.ForceExport(alg.FieldByName("hash")).Set(reflect.ValueOf(hash))
	areflect.ForceExport(alg.FieldByName("equal")).Set(reflect.ValueOf(equal))
}
