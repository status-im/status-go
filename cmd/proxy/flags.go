package main

import (
	"errors"
	"strconv"
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

// IntSlice is a type of flag that allows setting multiple int values.
type IntSlice []int

func (s *IntSlice) String() string {
	return "int slice"
}

// Set trims space from string and stores it.
func (s *IntSlice) Set(value string) error {
	val, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*s = append(*s, val)
	return nil
}
