package group_manager

import "github.com/waku-org/go-zerokit-rln/rln"

type MerkleRootTracker struct {
	rln                      *rln.RLN
	acceptableRootWindowSize int
	validMerkleRoots         []rln.MerkleNode
}

func NewMerkleRootTracker(acceptableRootWindowSize int, rlnInstance *rln.RLN) *MerkleRootTracker {
	return &MerkleRootTracker{
		acceptableRootWindowSize: acceptableRootWindowSize,
		rln:                      rlnInstance,
	}
}

func (m *MerkleRootTracker) Sync() error {
	root, err := m.rln.GetMerkleRoot()
	if err != nil {
		return err
	}

	m.validMerkleRoots = append(m.validMerkleRoots, root)
	if len(m.validMerkleRoots) > m.acceptableRootWindowSize {
		m.validMerkleRoots = m.validMerkleRoots[1:]
	}

	return nil
}

func (m *MerkleRootTracker) Roots() []rln.MerkleNode {
	return m.validMerkleRoots
}
