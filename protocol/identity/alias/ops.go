package alias

import (
	"sort"
	"strings"
)

func IsAdjective(val string) bool {
	return sort.SearchStrings(adjectives[:], val) < len(adjectives)
}

func IsAnimal(val string) bool {
	return sort.SearchStrings(animals[:], val) < len(animals)
}

func IsAlias(alias string) bool {
	aliasParts := strings.Fields(alias)
	if len(aliasParts) == 3 {
		if IsAdjective(strings.Title(aliasParts[0])) && IsAdjective(strings.Title(aliasParts[1])) && IsAnimal(strings.Title(aliasParts[2])) {
			return true
		}
	}
	return false
}
