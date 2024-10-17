package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_RenameSymbols(t *testing.T) {
	symbols := []string{"STT", "ETH", "BTC"}
	renames := RenameSymbols(symbols)
	require.Equal(t, []string{"SNT", "ETH", "BTC"}, renames)
}

func Test_RemoveDuplicates(t *testing.T) {
	strings := []string{"STT", "ETH", "BTC", "ETH", "BTC"}
	uniqueStrings := RemoveDuplicates(strings)
	require.Equal(t, []string{"STT", "ETH", "BTC"}, uniqueStrings)
}

func Test_GetRealSymbol(t *testing.T) {
	require.Equal(t, "SNT", GetRealSymbol("STT"))
	require.Equal(t, "ETH", GetRealSymbol("ETH"))
}

func Test_ChunkSymbols(t *testing.T) {
	symbols := []string{"STT", "ETH", "BTC"}
	params := ChunkSymbolsParams{MaxCharsPerChunk: 10, ExtraCharsPerSymbol: 1}
	chunks, err := ChunkSymbols(symbols, params)
	require.NoError(t, err)
	require.Equal(t, [][]string{{"STT", "ETH"}, {"BTC"}}, chunks)

	params = ChunkSymbolsParams{MaxCharsPerChunk: 10, ExtraCharsPerSymbol: 2}
	chunks, err = ChunkSymbols(symbols, params)
	require.NoError(t, err)
	require.Equal(t, [][]string{{"STT", "ETH"}, {"BTC"}}, chunks)

	params = ChunkSymbolsParams{MaxCharsPerChunk: 4, ExtraCharsPerSymbol: 1}
	chunks, err = ChunkSymbols(symbols, params)
	require.NoError(t, err)
	require.Equal(t, [][]string{{"STT"}, {"ETH"}, {"BTC"}}, chunks)

	params = ChunkSymbolsParams{MaxSymbolsPerChunk: 1, MaxCharsPerChunk: 10, ExtraCharsPerSymbol: 2}
	chunks, err = ChunkSymbols(symbols, params)
	require.NoError(t, err)
	require.Equal(t, [][]string{{"STT"}, {"ETH"}, {"BTC"}}, chunks)

	params = ChunkSymbolsParams{MaxCharsPerChunk: 9, ExtraCharsPerSymbol: 2}
	chunks, err = ChunkSymbols(symbols, params)
	require.NoError(t, err)
	require.Equal(t, [][]string{{"STT"}, {"ETH"}, {"BTC"}}, chunks)

	params = ChunkSymbolsParams{MaxCharsPerChunk: 2, ExtraCharsPerSymbol: 1}
	chunks, err = ChunkSymbols([]string{}, params)
	require.NoError(t, err)
	require.Len(t, chunks, 0)

	params = ChunkSymbolsParams{MaxCharsPerChunk: 2, ExtraCharsPerSymbol: 1}
	_, err = ChunkSymbols(symbols, params)
	require.Error(t, err)
}
