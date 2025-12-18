package utils

import (
	"bytes"
	"sort"
)

// MergeByteSlices deterministically concatenates multiple byte slices into one.
func MergeByteSlices(slices [][]byte) []byte {
	if len(slices) == 0 {
		return nil
	}

	idx := make([]int, len(slices))
	for i := range slices {
		idx[i] = i
	}
	sort.Slice(idx, func(i, j int) bool {
		return bytes.Compare(slices[idx[i]], slices[idx[j]]) < 0
	})

	var buf bytes.Buffer
	for _, id := range idx {
		buf.Write(slices[id])
	}

	return buf.Bytes()
}
