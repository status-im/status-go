package group_manager

import (
	"bytes"
	"sync"

	"github.com/waku-org/go-zerokit-rln/rln"
)

type RootsPerBlock struct {
	root        rln.MerkleNode
	blockNumber uint64
}

type MerkleRootTracker struct {
	sync.RWMutex

	rln                      *rln.RLN
	acceptableRootWindowSize int
	validMerkleRoots         []RootsPerBlock
	merkleRootBuffer         []RootsPerBlock
}

const maxBufferSize = 20

func NewMerkleRootTracker(acceptableRootWindowSize int, rlnInstance *rln.RLN) (*MerkleRootTracker, error) {
	result := &MerkleRootTracker{
		acceptableRootWindowSize: acceptableRootWindowSize,
		rln:                      rlnInstance,
	}

	_, err := result.UpdateLatestRoot(0)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *MerkleRootTracker) Backfill(fromBlockNumber uint64) {
	m.Lock()
	defer m.Unlock()

	numBlocks := 0
	for i := len(m.validMerkleRoots) - 1; i >= 0; i-- {
		if m.validMerkleRoots[i].blockNumber >= fromBlockNumber {
			numBlocks++
		}
	}

	if numBlocks == 0 {
		return
	}

	// Remove last roots
	rootsToPop := numBlocks
	if len(m.validMerkleRoots) < rootsToPop {
		rootsToPop = len(m.validMerkleRoots)
	}
	m.validMerkleRoots = m.validMerkleRoots[0 : len(m.validMerkleRoots)-rootsToPop]

	if len(m.merkleRootBuffer) == 0 {
		return
	}

	// Backfill the tree's acceptable roots
	rootsToRestore := numBlocks
	bufferLen := len(m.merkleRootBuffer)
	if bufferLen < rootsToRestore {
		rootsToRestore = bufferLen
	}
	for i := 0; i < rootsToRestore; i++ {
		x, newRootBuffer := m.merkleRootBuffer[len(m.merkleRootBuffer)-1], m.merkleRootBuffer[:len(m.merkleRootBuffer)-1] // Pop
		m.validMerkleRoots = append([]RootsPerBlock{x}, m.validMerkleRoots...)
		m.merkleRootBuffer = newRootBuffer
	}
}

// ContainsRoot is used to check whether a merkle tree root is contained in the list of valid merkle roots or not
func (m *MerkleRootTracker) ContainsRoot(root [32]byte) bool {
	return m.IndexOf(root) > -1
}

// IndexOf returns the index of a root if present in the list of valid merkle roots
func (m *MerkleRootTracker) IndexOf(root [32]byte) int {
	m.RLock()
	defer m.RUnlock()

	for i := range m.validMerkleRoots {
		if bytes.Equal(m.validMerkleRoots[i].root[:], root[:]) {
			return i
		}
	}

	return -1
}

func (m *MerkleRootTracker) UpdateLatestRoot(blockNumber uint64) (rln.MerkleNode, error) {
	m.Lock()
	defer m.Unlock()

	root, err := m.rln.GetMerkleRoot()
	if err != nil {
		return [32]byte{}, err
	}

	m.pushRoot(blockNumber, root)

	return root, nil
}

func (m *MerkleRootTracker) pushRoot(blockNumber uint64, root [32]byte) {
	m.validMerkleRoots = append(m.validMerkleRoots, RootsPerBlock{
		root:        root,
		blockNumber: blockNumber,
	})

	// Maintain valid merkle root window
	if len(m.validMerkleRoots) > m.acceptableRootWindowSize {
		m.merkleRootBuffer = append(m.merkleRootBuffer, m.validMerkleRoots[0])
		m.validMerkleRoots = m.validMerkleRoots[1:]
	}

	// Maintain merkle root buffer
	if len(m.merkleRootBuffer) > maxBufferSize {
		m.merkleRootBuffer = m.merkleRootBuffer[1:]
	}

}

func (m *MerkleRootTracker) Roots() []rln.MerkleNode {
	m.RLock()
	defer m.RUnlock()

	result := make([]rln.MerkleNode, len(m.validMerkleRoots))
	for i := range m.validMerkleRoots {
		result[i] = m.validMerkleRoots[i].root
	}

	return result
}

func (m *MerkleRootTracker) Buffer() []rln.MerkleNode {
	m.RLock()
	defer m.RUnlock()

	result := make([]rln.MerkleNode, len(m.merkleRootBuffer))
	for i := range m.merkleRootBuffer {
		result[i] = m.merkleRootBuffer[i].root
	}

	return result
}
