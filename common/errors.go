package common

import "fmt"

var (
	ErrBigIntSetFromString = func(val string) error { return fmt.Errorf("failed to set big.Int balance from string '%s'", val) }
)
