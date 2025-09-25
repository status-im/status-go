package common

import (
	"fmt"
	"math/big"
	"strings"
)

func MakeKey(fromTokenKey, toTokenKey string, amount *big.Int) string {
	key := fmt.Sprintf("%s-%s", fromTokenKey, toTokenKey)
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
