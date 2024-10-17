package utils

import (
	"errors"
	"strings"
)

var renameMapping = map[string]string{
	"STT": "SNT",
}

func RenameSymbols(symbols []string) (renames []string) {
	for _, symbol := range symbols {
		renames = append(renames, GetRealSymbol(symbol))
	}
	return
}

func RemoveDuplicates(strings []string) []string {
	uniqueStrings := make(map[string]bool)
	var uniqueSlice []string
	for _, str := range strings {
		if !uniqueStrings[str] {
			uniqueStrings[str] = true
			uniqueSlice = append(uniqueSlice, str)
		}
	}
	return uniqueSlice
}

func GetRealSymbol(symbol string) string {
	if val, ok := renameMapping[strings.ToUpper(symbol)]; ok {
		return val
	}
	return strings.ToUpper(symbol)
}

type ChunkSymbolsParams struct {
	MaxSymbolsPerChunk  int
	MaxCharsPerChunk    int
	ExtraCharsPerSymbol int
}

func ChunkSymbols(symbols []string, params ChunkSymbolsParams) ([][]string, error) {
	var chunks [][]string
	if len(symbols) == 0 {
		return chunks, nil
	}

	chunk := make([]string, 0, 100)
	chunkChars := 0
	for _, symbol := range symbols {
		symbolChars := len(symbol) + params.ExtraCharsPerSymbol
		if params.MaxCharsPerChunk > 0 && symbolChars > params.MaxCharsPerChunk {
			return nil, errors.New("chunk cannot fit symbol: " + symbol)
		}
		if (params.MaxCharsPerChunk > 0 && chunkChars+symbolChars > params.MaxCharsPerChunk) ||
			(params.MaxSymbolsPerChunk > 0 && len(chunk) >= params.MaxSymbolsPerChunk) {
			// Max chars/symbols reached, store chunk and start a new one
			chunks = append(chunks, chunk)
			chunk = make([]string, 0, 100)
			chunkChars = 0
		}
		chunk = append(chunk, symbol)
		chunkChars += symbolChars
	}
	chunks = append(chunks, chunk)

	return chunks, nil
}
