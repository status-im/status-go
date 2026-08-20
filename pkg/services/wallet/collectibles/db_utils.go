package collectibles

import (
	"slices"
	"strings"
)

func insertStatement(allowUpdate bool) string {
	if allowUpdate {
		return `INSERT OR REPLACE`
	}
	return `INSERT OR IGNORE`
}

// valuePlaceholders returns the "?, ?, ..." list matching a comma-separated column
// list, so that adding a column cannot silently desync the two.
func valuePlaceholders(columns string) string {
	return strings.Join(slices.Repeat([]string{"?"}, strings.Count(columns, ",")+1), ", ")
}
