package main

import "strconv"

// atoi is a helper to safely convert a string to an int
func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}
