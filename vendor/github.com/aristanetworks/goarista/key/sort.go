// Copyright (C) 2015  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package key

import (
	"sort"
)

// SortedKeys returns the keys of the given map, in a sorted order.
func SortedKeys(m map[string]interface{}) []string {
	res := make([]string, len(m))
	var i int
	for k := range m {
		res[i] = k
		i++
	}
	sort.Strings(res)
	return res
}
