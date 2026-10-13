package defaulttokenlists

import (
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func allLists() []*DownloadedTokenList {
	return []*DownloadedTokenList{
		&StatusTokenList, &UniswapTokenList, &CoingeckoEthereumTokenList, &CoingeckoOptimismTokenList,
		&CoingeckoArbitrumTokenList, &CoingeckoBaseTokenList, &CoingeckoBscTokenList, &CoingeckoLineaTokenList,
	}
}

// The embedded lists are megabytes of JSON; they must stay in the binary's data section, not be copied to the heap.
// The default heap profile samples about every 512KiB, so heap copies made by this package's init would show up.
func TestEmbeddedListsAreNotOnHeap(t *testing.T) {
	total := 0
	for _, l := range allLists() {
		require.NotEmpty(t, l.JsonData, l.ID)
		total += len(l.JsonData)
	}
	require.Greater(t, total, 2<<20)

	runtime.GC()
	records := make([]runtime.MemProfileRecord, 4096)
	n, ok := runtime.MemProfile(records, false)
	require.True(t, ok)
	for _, r := range records[:n] {
		frames := runtime.CallersFrames(r.Stack())
		for {
			f, more := frames.Next()
			require.False(t, strings.Contains(f.Function, "default-lists.init"),
				"%d bytes allocated by %s are still on the heap", r.InUseBytes(), f.Function)
			if !more {
				break
			}
		}
	}
}
