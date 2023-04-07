//go:build (linux || windows) && amd64 && !android
// +build linux windows
// +build amd64
// +build !android

package link

import r "github.com/waku-org/go-zerokit-rln-x86_64/rln"

type RLNWrapper struct {
	ffi *r.RLN
}

func NewWithParams(depth int, wasm []byte, zkey []byte, verifKey []byte) (*RLNWrapper, error) {
	rln, err := r.NewWithParams(depth, wasm, zkey, verifKey)
	if err != nil {
		return nil, err
	}
	return &RLNWrapper{ffi: rln}, nil
}

func NewWithFolder(depth int, resourcesFolderPath string) (*RLNWrapper, error) {
	rln, err := r.NewWithFolder(depth, resourcesFolderPath)
	if err != nil {
		return nil, err
	}
	return &RLNWrapper{ffi: rln}, nil
}

func (i RLNWrapper) ExtendedKeyGen() []byte {
	return i.ffi.ExtendedKeyGen()
}

func (i RLNWrapper) Hash(input []byte) ([]byte, error) {
	return i.ffi.Hash(input)
}

func (i RLNWrapper) PoseidonHash(input []byte) ([]byte, error) {
	return i.ffi.PoseidonHash(input)
}

func (i RLNWrapper) SetNextLeaf(idcommitment []byte) bool {
	return i.ffi.SetNextLeaf(idcommitment)
}

func (i RLNWrapper) SetLeavesFrom(index uint, idcommitments []byte) bool {
	return i.ffi.SetLeavesFrom(index, idcommitments)
}

func (i RLNWrapper) DeleteLeaf(index uint) bool {
	return i.ffi.DeleteLeaf(index)
}

func (i RLNWrapper) GetRoot() ([]byte, error) {
	return i.ffi.GetRoot()
}

func (i RLNWrapper) GenerateRLNProof(input []byte) ([]byte, error) {
	return i.ffi.GenerateRLNProof(input)
}

func (i RLNWrapper) VerifyWithRoots(input []byte, roots []byte) (bool, error) {
	return i.ffi.VerifyWithRoots(input, roots)
}
