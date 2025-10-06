package ssz

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/emicklei/dot"
)

// Proof represents a merkle proof against a general index.
type Proof struct {
	Index  int
	Leaf   []byte
	Hashes [][]byte
}

// Multiproof represents a merkle proof of several leaves.
type Multiproof struct {
	Indices []int
	Leaves  [][]byte
	Hashes  [][]byte
}

// Compress returns a new proof with zero hashes omitted.
// See `CompressedMultiproof` for more info.
func (p *Multiproof) Compress() *CompressedMultiproof {
	compressed := &CompressedMultiproof{
		Indices:    p.Indices,
		Leaves:     p.Leaves,
		Hashes:     make([][]byte, 0, len(p.Hashes)),
		ZeroLevels: make([]int, 0, len(p.Hashes)),
	}

	for _, h := range p.Hashes {
		if l, ok := zeroHashLevels[string(h)]; ok {
			compressed.ZeroLevels = append(compressed.ZeroLevels, l)
			compressed.Hashes = append(compressed.Hashes, nil)
		} else {
			compressed.Hashes = append(compressed.Hashes, h)
		}
	}

	return compressed
}

// CompressedMultiproof represents a compressed merkle proof of several leaves.
// Compression is achieved by omitting zero hashes (and their hashes). `ZeroLevels`
// contains information which helps the verifier fill in those hashes.
type CompressedMultiproof struct {
	Indices    []int
	Leaves     [][]byte
	Hashes     [][]byte
	ZeroLevels []int // Stores the level for every omitted zero hash in the proof
}

// Decompress returns a new multiproof, filling in the omitted
// zero hashes. See `CompressedMultiProof` for more info.
func (c *CompressedMultiproof) Decompress() *Multiproof {
	p := &Multiproof{
		Indices: c.Indices,
		Leaves:  c.Leaves,
		Hashes:  make([][]byte, len(c.Hashes)),
	}

	zc := 0
	for i, h := range c.Hashes {
		if h == nil {
			p.Hashes[i] = zeroHashes[c.ZeroLevels[zc]][:]
			zc++
		} else {
			p.Hashes[i] = c.Hashes[i]
		}
	}

	return p
}

// Node represents a node in the tree
// backing of a SSZ object.
type Node struct {
	left    *Node
	right   *Node
	isEmpty bool

	value []byte
}

func (n *Node) Draw(w io.Writer) {
	g := dot.NewGraph(dot.Directed)
	n.draw(1, g)
	g.Write(w)
}

func (n *Node) draw(levelOrder int, g *dot.Graph) dot.Node {
	var h string
	if n.left != nil || n.right != nil {
		h = hex.EncodeToString(n.Hash())
	}
	if n.value != nil {
		h = hex.EncodeToString(n.value)
	}
	dn := g.Node(fmt.Sprintf("n%d", levelOrder)).
		Label(fmt.Sprintf("%d\n%s..%s", levelOrder, h[:3], h[len(h)-3:]))

	if n.left != nil {
		ln := n.left.draw(2*levelOrder, g)
		g.Edge(dn, ln).Label("0")
	}
	if n.right != nil {
		rn := n.right.draw(2*levelOrder+1, g)
		g.Edge(dn, rn).Label("1")
	}
	return dn
}

func (n *Node) Show(maxDepth int) {
	fmt.Printf("--- Show node ---\n")
	n.show(0, maxDepth)
}

func (n *Node) show(depth int, maxDepth int) {
	space := ""
	for i := 0; i < depth; i++ {
		space += "\t"
	}
	print := func(msgs ...string) {
		for _, msg := range msgs {
			fmt.Printf("%s%s", space, msg)
		}
	}

	if n.left != nil || n.right != nil {
		// leaf hash is the same as value
		print("HASH: " + hex.EncodeToString(n.Hash()) + "\n")
	}
	if n.value != nil {
		print("VALUE: " + hex.EncodeToString(n.value) + "\n")
	}

	if maxDepth > 0 {
		if depth == maxDepth {
			// only print hash if we are too deep
			return
		}
	}

	if n.left != nil {
		print("LEFT: \n")
		n.left.show(depth+1, maxDepth)
	}
	if n.right != nil {
		print("RIGHT: \n")
		n.right.show(depth+1, maxDepth)
	}
}

// NewNodeWithValue initializes a leaf node.
func NewNodeWithValue(value []byte) *Node {
	return &Node{left: nil, right: nil, value: value}
}

func NewEmptyNode(zeroOrderHash []byte) *Node {
	return &Node{left: nil, right: nil, value: zeroOrderHash, isEmpty: true}
}

// NewNodeWithLR initializes a branch node.
func NewNodeWithLR(left, right *Node) *Node {
	return &Node{left: left, right: right, value: nil}
}

// TreeFromChunks constructs a tree from leaf values.
// The number of leaves should be a power of 2.
func TreeFromChunks(chunks [][]byte) (*Node, error) {
	numLeaves := len(chunks)
	if !isPowerOfTwo(numLeaves) {
		return nil, errors.New("Number of leaves should be a power of 2")
	}

	leaves := make([]*Node, numLeaves)
	for i, c := range chunks {
		leaves[i] = NewNodeWithValue(c)
	}
	return TreeFromNodes(leaves, numLeaves)
}

// TreeFromNodes constructs a tree from leaf nodes.
// This is useful for merging subtrees.
// The limit should be a power of 2.
// Adjacent sibling nodes will be filled with zero order hashes that have been precomputed based on the tree depth.
func TreeFromNodes(leaves []*Node, limit int) (*Node, error) {
	numLeaves := len(leaves)

	depth := floorLog2(limit)
	zeroOrderHashes := getZeroOrderHashes(depth)

	// there are no leaves, return a zero order hash node
	if numLeaves == 0 {
		return NewEmptyNode(zeroOrderHashes[0]), nil
	}

	// now we know numLeaves are at least 1.

	// if the max leaf limit is 1, return the one leaf we have
	if limit == 1 {
		return leaves[0], nil
	}
	// if the max leaf limit is 2
	if limit == 2 {
		// but we only have 1 leaf, add a zero order hash as the right node
		if numLeaves == 1 {
			return NewNodeWithLR(leaves[0], NewEmptyNode(zeroOrderHashes[1])), nil
		}
		// otherwise return the two nodes we have
		return NewNodeWithLR(leaves[0], leaves[1]), nil
	}

	if !isPowerOfTwo(limit) {
		return nil, errors.New("number of leaves should be a power of 2")
	}

	leavesStart := powerTwo(depth)
	leafIndex := numLeaves - 1

	nodes := make(map[int]*Node)

	nodesStartIndex := leavesStart
	nodesEndIndex := nodesStartIndex + numLeaves - 1

	// for each tree level
	for k := depth; k >= 0; k-- {
		for i := nodesEndIndex; i >= nodesStartIndex; i-- {
			// leaf node, add to map
			if k == depth {
				nodes[i] = leaves[leafIndex]
				leafIndex--
			} else { // branch node, compute
				leftIndex := i * 2
				rightIndex := i*2 + 1
				// both nodes are empty, unexpected condition
				if nodes[leftIndex] == nil && nodes[rightIndex] == nil {
					return nil, errors.New("unexpected empty right and left nodes")
				}
				// node with empty right node, add zero order hash as right node and mark right node as empty
				if nodes[leftIndex] != nil && nodes[rightIndex] == nil {
					nodes[i] = NewNodeWithLR(nodes[leftIndex], NewEmptyNode(zeroOrderHashes[k+1]))
				}
				// node with left and right child
				if nodes[leftIndex] != nil && nodes[rightIndex] != nil {
					nodes[i] = NewNodeWithLR(nodes[leftIndex], nodes[rightIndex])
				}
			}
		}
		nodesStartIndex = nodesStartIndex / 2
		nodesEndIndex = int(math.Floor(float64(nodesEndIndex)) / 2)
	}

	rootNode := nodes[1]

	if rootNode == nil {
		return nil, errors.New("tree root node could not be computed")
	}

	return nodes[1], nil
}

func TreeFromNodesWithMixin(leaves []*Node, num, limit int) (*Node, error) {
	if !isPowerOfTwo(limit) {
		return nil, errors.New("size of tree should be a power of 2")
	}

	mainTree, err := TreeFromNodes(leaves, limit)
	if err != nil {
		return nil, err
	}

	// Mixin len
	countLeaf := LeafFromUint64(uint64(num))
	node := NewNodeWithLR(mainTree, countLeaf)
	return node, nil
}

// Get fetches a node with the given general index.
func (n *Node) Get(index int) (*Node, error) {
	pathLen := getPathLength(index)
	cur := n
	for i := pathLen - 1; i >= 0; i-- {
		if isRight := getPosAtLevel(index, i); isRight {
			cur = cur.right
		} else {
			cur = cur.left
		}
		if cur == nil {
			return nil, errors.New("Node not found in tree")
		}
	}

	return cur, nil
}

// Hash returns the hash of the subtree with the given Node as its root.
// If root has no children, it returns root's value (not its hash).
func (n *Node) Hash() []byte {
	// TODO: handle special cases: empty root, one non-empty node
	return hashNode(n)
}

func hashNode(n *Node) []byte {
	if n.left == nil && n.right == nil {
		return n.value
	}

	if n.left == nil {
		panic("Tree incomplete")
	}

	if n.value != nil {
		// This value has already been hashed, don't do the work again.
		return n.value
	}

	if n.right.isEmpty {
		result := hashFn(append(hashNode(n.left), n.right.value...))
		n.value = result // Set the hash result on each node so that proofs can be generated for any level
		return result
	}

	result := hashFn(append(hashNode(n.left), hashNode(n.right)...))
	n.value = result
	return result
}

// getZeroOrderHashes precomputes zero order hashes to create an easy map lookup
// for zero leafs and their parent nodes.
func getZeroOrderHashes(depth int) map[int][]byte {
	zeroOrderHashes := make(map[int][]byte)

	emptyValue := make([]byte, 32)
	zeroOrderHashes[depth] = emptyValue

	for i := depth - 1; i >= 0; i-- {
		zeroOrderHashes[i] = hashFn(append(zeroOrderHashes[i+1], zeroOrderHashes[i+1]...))
	}

	return zeroOrderHashes
}

// Prove returns a list of sibling values and hashes needed
// to compute the root hash for a given general index.
func (n *Node) Prove(index int) (*Proof, error) {
	pathLen := getPathLength(index)
	proof := &Proof{Index: index}
	hashes := make([][]byte, 0, pathLen)

	cur := n
	for i := pathLen - 1; i >= 0; i-- {
		var siblingHash []byte
		if isRight := getPosAtLevel(index, i); isRight {
			siblingHash = hashNode(cur.left)
			cur = cur.right
		} else {
			siblingHash = hashNode(cur.right)
			cur = cur.left
		}
		hashes = append([][]byte{siblingHash}, hashes...)
		if cur == nil {
			return nil, errors.New("Node not found in tree")
		}
	}

	proof.Hashes = hashes
	if cur.value == nil {
		// This is an intermediate node without a value; add the hash to it so that we're providing a suitable leaf value.
		cur.value = hashNode(cur)
	}
	proof.Leaf = cur.value

	return proof, nil
}

func (n *Node) ProveMulti(indices []int) (*Multiproof, error) {
	reqIndices := getRequiredIndices(indices)
	proof := &Multiproof{Indices: indices, Leaves: make([][]byte, len(indices)), Hashes: make([][]byte, len(reqIndices))}

	for i, gi := range indices {
		node, err := n.Get(gi)
		if err != nil {
			return nil, err
		}
		proof.Leaves[i] = node.value
	}

	for i, gi := range reqIndices {
		cur, err := n.Get(gi)
		if err != nil {
			return nil, err
		}
		proof.Hashes[i] = hashNode(cur)
	}

	return proof, nil
}

func LeafFromUint64(i uint64) *Node {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf[:8], i)
	return NewNodeWithValue(buf)
}

func LeafFromUint32(i uint32) *Node {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint32(buf[:4], i)
	return NewNodeWithValue(buf)
}

func LeafFromUint16(i uint16) *Node {
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint16(buf[:2], i)
	return NewNodeWithValue(buf)
}

func LeafFromUint8(i uint8) *Node {
	buf := make([]byte, 32)
	buf[0] = byte(i)
	return NewNodeWithValue(buf)
}

func LeafFromBool(b bool) *Node {
	buf := make([]byte, 32)
	if b {
		buf[0] = 1
	}
	return NewNodeWithValue(buf)
}

func LeafFromBytes(b []byte) *Node {
	l := len(b)
	if l > 32 {
		panic("Unimplemented")
	}

	if l == 32 {
		return NewNodeWithValue(b[:])
	}

	// < 32
	return NewNodeWithValue(append(b, zeroBytes[:32-l]...))
}

func EmptyLeaf() *Node {
	return NewNodeWithValue(zeroBytes[:32])
}

func LeavesFromUint64(items []uint64) []*Node {
	if len(items) == 0 {
		return []*Node{}
	}

	numLeaves := (len(items)*8 + 31) / 32
	buf := make([]byte, numLeaves*32)
	for i, v := range items {
		binary.LittleEndian.PutUint64(buf[i*8:(i+1)*8], v)
	}

	leaves := make([]*Node, numLeaves)
	for i := 0; i < numLeaves; i++ {
		v := buf[i*32 : (i+1)*32]
		leaves[i] = NewNodeWithValue(v)
	}

	return leaves
}

func isPowerOfTwo(n int) bool {
	return (n & (n - 1)) == 0
}

func floorLog2(n int) int {
	return int(math.Floor(math.Log2(float64(n))))
}

func powerTwo(n int) int {
	return int(math.Pow(2, float64(n)))
}
