package common

import (
	"fmt"
	"math/big"
	"strings"
)

func MakeKey(fromChain, toChain uint64, fromTokenSymbol, toTokenSymbol string, amount *big.Int) string {
	key := fmt.Sprintf("%d-%d", fromChain, toChain)
	if fromTokenSymbol != "" || toTokenSymbol != "" {
		key = fmt.Sprintf("%s-%s-%s", key, fromTokenSymbol, toTokenSymbol)
	}
	if amount != nil {
		key = fmt.Sprintf("%s-%s", key, amount.String())
	}
	return key
}

func GetNameFromEnsUsername(ensUsername string) string {
	suffix := ".stateofus.eth"
	if strings.HasSuffix(ensUsername, suffix) {
		return ensUsername[:len(ensUsername)-len(suffix)]
	}
	return ensUsername
}
