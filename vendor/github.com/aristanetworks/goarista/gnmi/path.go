// Copyright (C) 2017  Arista Networks, Inc.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the COPYING file.

package gnmi

import "strings"

// nextTokenIndex returns the end index of the first token.
func nextTokenIndex(path string) int {
	var inBrackets bool
	var escape bool
	for i, c := range path {
		switch c {
		case '[':
			inBrackets = true
			escape = false
		case ']':
			if !escape {
				inBrackets = false
			}
			escape = false
		case '\\':
			escape = !escape
		case '/':
			if !inBrackets {
				return i
			}
			escape = false
		default:
			escape = false
		}
	}
	return len(path)
}

// SplitPath splits a gnmi path according to the spec. See
// https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-path-conventions.md
// No validation is done. Behavior is undefined if path is an invalid
// gnmi path. TODO: Do validation?
func SplitPath(path string) []string {
	var result []string
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	for len(path) > 0 {
		i := nextTokenIndex(path)
		result = append(result, path[:i])
		path = path[i:]
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
	}
	return result
}

// SplitPaths splits multiple gnmi paths
func SplitPaths(paths []string) [][]string {
	out := make([][]string, len(paths))
	for i, path := range paths {
		out[i] = SplitPath(path)
	}
	return out
}

// JoinPath joins a gnmi path
func JoinPath(path []string) string {
	return "/" + strings.Join(path, "/")
}
