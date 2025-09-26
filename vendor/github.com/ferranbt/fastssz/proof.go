package ssz

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/minio/sha256-simd"
)

// VerifyProof verifies a single merkle branch. It's more
// efficient than VerifyMultiproof for proving one leaf.
func VerifyProof(root []byte, proof *Proof) (bool, error) {
	if len(proof.Hashes) != getPathLength(proof.Index) {
		return false, errors.New("invalid proof length")
	}

	node := proof.Leaf[:]
	tmp := make([]byte, 64)
	for i, h := range proof.Hashes {
		if getPosAtLevel(proof.Index, i) {
			copy(tmp[:32], h[:])
			copy(tmp[32:], node[:])
			node = hashFn(tmp)
		} else {
			copy(tmp[:32], node[:])
			copy(tmp[32:], h[:])
			node = hashFn(tmp)
		}
	}

	return bytes.Equal(root, node), nil
}

// VerifyMultiproof verifies a proof for multiple leaves against the given root.
func VerifyMultiproof(root []byte, proof [][]byte, leaves [][]byte, indices []int) (bool, error) {
	if len(indices) == 0 {
		return false, errors.New("indices length is zero")
	}

	if len(leaves) != len(indices) {
		return false, errors.New("number of leaves and indices mismatch")
	}

	reqIndices := getRequiredIndices(indices)
	if len(reqIndices) != len(proof) {
		return false, fmt.Errorf("number of proof hashes %d and required indices %d mismatch", len(proof), len(reqIndices))
	}

	// userGenIndices contains all generalised indices between leaves and proof hashes
	// i.e., the indices retrieved from the user of this function
	userGenIndices := make([]int, len(indices)+len(reqIndices))
	pos := 0
	// Create database of index -> value (hash) from inputs
	db := make(map[int][]byte)
	for i, leaf := range leaves {
		db[indices[i]] = leaf
		userGenIndices[pos] = indices[i]
		pos++
	}
	for i, h := range proof {
		db[reqIndices[i]] = h
		userGenIndices[pos] = reqIndices[i]
		pos++
	}

	// Make sure keys are sorted in reverse order since we start from the leaves
	sort.Sort(sort.Reverse(sort.IntSlice(userGenIndices)))

	// The depth of the tree up to the greatest index
	cap := int(math.Log2(float64(userGenIndices[0])))

	// Allocate space for auxiliary keys created when computing intermediate hashes
	// Auxiliary indices are useful to avoid using store all indices to traverse
	// in a single array and sort upon an insertion, which would be inefficient.
	auxGenIndices := make([]int, 0, cap)

	// To keep track the current position to inspect in both arrays
	pos = 0
	posAux := 0

	tmp := make([]byte, 64)
	var index int

	// Iter over the tree, computing hashes and storing them
	// in the in-memory database, until the root is reached.
	//
	// EXIT CONDITION: no more indices to use in both arrays
	for posAux < len(auxGenIndices) || pos < len(userGenIndices) {
		// We need to establish from which array we're going to take the next index
		//
		// 1. If we've no auxiliary indices yet, we're going to use the generalised ones
		// 2. If we have no more client indices, we're going to use the auxiliary ones
		// 3. If we both, then we're going to compare them and take the biggest one
		if len(auxGenIndices) == 0 || (pos < len(userGenIndices) && auxGenIndices[posAux] < userGenIndices[pos]) {
			index = userGenIndices[pos]
			pos++
		} else {
			index = auxGenIndices[posAux]
			posAux++
		}

		// Root has been reached
		if index == 1 {
			break
		}

		// If the parent is already computed, we don't need to calculate the intermediate hash
		_, hasParent := db[getParent(index)]
		if hasParent {
			continue
		}

		left, hasLeft := db[(index|1)^1]
		right, hasRight := db[index|1]
		if !hasRight || !hasLeft {
			return false, fmt.Errorf("proof is missing required nodes, either %d or %d", (index|1)^1, index|1)
		}

		copy(tmp[:32], left[:])
		copy(tmp[32:], right[:])
		parentIndex := getParent(index)
		db[parentIndex] = hashFn(tmp)

		// An intermediate hash has been computed, as such we need to store its index
		// to remember to examine it later
		auxGenIndices = append(auxGenIndices, parentIndex)

	}

	res, ok := db[1]
	if !ok {
		return false, fmt.Errorf("root was not computed during proof verification")
	}

	return bytes.Equal(res, root), nil
}

// Returns the position (i.e. false for left, true for right)
// of an index at a given level.
// Level 0 is the actual index's level, Level 1 is the position
// of the parent, etc.
func getPosAtLevel(index int, level int) bool {
	return (index & (1 << level)) > 0
}

// Returns the length of the path to a node represented by its generalized index.
func getPathLength(index int) int {
	return int(math.Log2(float64(index)))
}

// Returns the generalized index for a node's sibling.
func getSibling(index int) int {
	return index ^ 1
}

// Returns the generalized index for a node's parent.
func getParent(index int) int {
	return index >> 1
}

// Returns generalized indices for all nodes in the tree that are
// required to prove the given leaf indices. The returned indices
// are in a decreasing order.
func getRequiredIndices(leafIndices []int) []int {
	exists := struct{}{}
	// Sibling hashes needed for verification
	required := make(map[int]struct{})
	// Set of hashes that will be computed
	// on the path from leaf to root.
	computed := make(map[int]struct{})
	leaves := make(map[int]struct{})

	for _, leaf := range leafIndices {
		leaves[leaf] = exists
		cur := leaf
		for cur > 1 {
			sibling := getSibling(cur)
			parent := getParent(cur)
			required[sibling] = exists
			computed[parent] = exists
			cur = parent
		}
	}

	requiredList := make([]int, 0, len(required))
	// Remove computed indices from required ones
	for r := range required {
		_, isComputed := computed[r]
		_, isLeaf := leaves[r]
		if !isComputed && !isLeaf {
			requiredList = append(requiredList, r)
		}
	}

	sort.Sort(sort.Reverse(sort.IntSlice(requiredList)))
	return requiredList
}

func hashFn(data []byte) []byte {
	res := sha256.Sum256(data)
	return res[:]
}
