package whispertypes

import "fmt"

// EnodeID is a unique identifier for each node.
type EnodeID [32]byte

// ID prints as a long hexadecimal number.
func (n EnodeID) String() string {
	return fmt.Sprintf("%x", n[:])
}
