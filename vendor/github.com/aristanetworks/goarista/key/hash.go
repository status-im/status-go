// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key

import "unsafe"

//go:noescape
//go:linkname strhash runtime.strhash
func strhash(a unsafe.Pointer, h uintptr) uintptr

func _strhash(s string) uintptr {
	return strhash(unsafe.Pointer(&s), 0)
}

//go:noescape
//go:linkname nilinterhash runtime.nilinterhash
func nilinterhash(a unsafe.Pointer, h uintptr) uintptr

func _nilinterhash(v interface{}) uintptr {
	return nilinterhash(unsafe.Pointer(&v), 0)
}
