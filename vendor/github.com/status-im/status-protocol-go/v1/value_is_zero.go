// +build go1.13

package statusproto

import "reflect"

// isZeroValue reports whether v is the zero value for its type.
// It panics if the argument is invalid.
func isZeroValue(v reflect.Value) bool {
	return v.IsZero()
}
