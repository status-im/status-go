package main

import (
	"fmt"
	"sort"
	"unicode"
	"unicode/utf8"
)

type metrics map[string]interface{}

type flatMetrics map[string]string

func (fm flatMetrics) String() string {
	var s string
	keys := make([]string, 0)

	for k, _ := range fm {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		s += fmt.Sprintf("%s %s\n", k, fm[k])
	}

	return s
}

func transformMetrics(data map[string]interface{}) flatMetrics {
	return flattenMetrics(data, make(flatMetrics), "")
}

func flattenMetrics(data metrics, memo flatMetrics, prefix string) flatMetrics {
	for k, v := range data {
		key := prefix + normalizeKey(k)

		switch value := v.(type) {
		case map[string]interface{}:
			memo = flattenMetrics(value, memo, key+"_")
		case string:
			memo[key] = value
		default:
			stringValue := fmt.Sprintf("%+v", value)
			memo[key] = stringValue
		}
	}

	return memo
}

func normalizeKey(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return ""
	}

	return string(unicode.ToLower(r)) + s[n:]
}
