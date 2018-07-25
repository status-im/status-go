package main

import (
	"errors"
	"strings"
)

// ErrorEmpty returned when value is empty.
var ErrorEmpty = errors.New("empty value not allowed")

// StringSlice is a type of flag that allows setting multiple string values.
type StringSlice []string

func (s *StringSlice) String() string {
	return "string slice"
}

// Set trims space from string and stores it.
func (s *StringSlice) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 {
		return ErrorEmpty
	}
	*s = append(*s, trimmed)
	return nil
}
