package defaulttokenlists

import (
	"runtime"
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
func TestEmbeddedListsAreNotOnHeap(t *testing.T) {
	total := 0
	for _, l := range allLists() {
		require.NotEmpty(t, l.JsonData, l.ID)
		total += len(l.JsonData)
	}

	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	require.Less(t, m.HeapAlloc, uint64(total/2), "heap %d KiB holds the %d KiB of embedded lists", m.HeapAlloc>>10, total>>10)
}
