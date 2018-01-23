// Copyright (C) 2016  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key

// This value needs to look very much not like a pointer.
const sentinel = uintptr(0xFF123456FFABCDEF)
