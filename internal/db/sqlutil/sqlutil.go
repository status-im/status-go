package sqlutil

import (
	"fmt"
	"strings"
)

func Placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?, ", count), ", ")
}

func In(query string, counts ...int) string {
	lists := make([]interface{}, len(counts))
	for i, count := range counts {
		lists[i] = Placeholders(count)
	}
	return fmt.Sprintf(query, lists...)
}
