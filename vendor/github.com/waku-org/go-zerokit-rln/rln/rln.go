package rln

/*
#include "./librln.h"
*/
import "C"
import (
	"encoding/binary"
	"errors"
	"unsafe"

	"github.com/waku-org/go-zerokit-rln/rln/resources"
)

// RLN represents the context used for rln.
type RLN struct {
	ptr *C.RLN
}

// NewRLN generates an instance of RLN. An instance supports both zkSNARKs logics
// and Merkle tree data structure and operations. It uses a depth of 20 by default
func NewRLN() (*RLN, error) {

	wasm, err := resources.Asset("tree_height_20/rln.wasm")
	if err != nil {
		return nil, err
	}

	zkey, err := resources.Asset("tree_height_20/rln_final.zkey")
	if err != nil {
		return nil, err
	}

	verifKey, err := resources.Asset("tree_height_20/verification_key.json")
	if err != nil {
		return nil, err
	}

	r := &RLN{}

	depth := 20

	wasmBuffer := toCBufferPtr(wasm)
	zkeyBuffer := toCBufferPtr(zkey)
	verifKeyBuffer := toCBufferPtr(verifKey)

	if !bool(C.new_with_params(C.uintptr_t(depth), wasmBuffer, zkeyBuffer, verifKeyBuffer, &r.ptr)) {
		return nil, errors.New("failed to initialize")
	}

	return r, nil
}

// NewRLNWithParams generates an instance of RLN. An instance supports both zkSNARKs logics
// and Merkle tree data structure and operations. The parameter `depth“ indicates the depth of Merkle tree
func NewRLNWithParams(depth int, wasm []byte, zkey []byte, verifKey []byte) (*RLN, error) {
	r := &RLN{}

	wasmBuffer := toCBufferPtr(wasm)
	zkeyBuffer := toCBufferPtr(zkey)
	verifKeyBuffer := toCBufferPtr(verifKey)

	if !bool(C.new_with_params(C.uintptr_t(depth), wasmBuffer, zkeyBuffer, verifKeyBuffer, &r.ptr)) {
		return nil, errors.New("failed to initialize")
	}

	return r, nil
}

// NewRLNWithFolder generates an instance of RLN. An instance supports both zkSNARKs logics
// and Merkle tree data structure and operations. The parameter `deptk` indicates the depth of Merkle tree
// The parameter “
func NewRLNWithFolder(depth int, resourcesFolderPath string) (*RLN, error) {
	r := &RLN{}

	pathBuffer := toCBufferPtr([]byte(resourcesFolderPath))

	if !bool(C.new(C.uintptr_t(depth), pathBuffer, &r.ptr)) {
		return nil, errors.New("failed to initialize")
	}

	return r, nil
}

func toCBufferPtr(input []byte) *C.Buffer {
	buf := toBuffer(input)

	size := int(unsafe.Sizeof(buf))
	in := (*C.Buffer)(C.malloc(C.size_t(size)))
	*in = buf

	return in
}

// MembershipKeyGen generates a MembershipKeyPair that can be used for the registration into the rln membership contract
func (r *RLN) MembershipKeyGen() (*MembershipKeyPair, error) {
	buffer := toBuffer([]byte{})
	if !bool(C.key_gen(r.ptr, &buffer)) {
		return nil, errors.New("error in key generation")
	}

	key := &MembershipKeyPair{
		IDKey:        [32]byte{},
		IDCommitment: [32]byte{},
	}

	// the public and secret keys together are 64 bytes
	generatedKeys := C.GoBytes(unsafe.Pointer(buffer.ptr), C.int(buffer.len))
	if len(generatedKeys) != 64 {
		return nil, errors.New("the generated keys are invalid")
	}

	copy(key.IDKey[:], generatedKeys[:32])
	copy(key.IDCommitment[:], generatedKeys[32:64])

	return key, nil
}

// appendLength returns length prefixed version of the input with the following format
// [len<8>|input<var>], the len is a 8 byte value serialized in little endian
func appendLength(input []byte) []byte {
	inputLen := make([]byte, 8)
	binary.LittleEndian.PutUint64(inputLen, uint64(len(input)))
	return append(inputLen, input...)
}

// toBuffer converts the input to a buffer object that is used to communicate data with the rln lib
func toBuffer(data []byte) C.Buffer {
	dataPtr, dataLen := sliceToPtr(data)
	return C.Buffer{
		ptr: dataPtr,
		len: C.uintptr_t(dataLen),
	}
}

func sliceToPtr(slice []byte) (*C.uchar, C.int) {
	if len(slice) == 0 {
		return nil, 0
	} else {
		return (*C.uchar)(unsafe.Pointer(&slice[0])), C.int(len(slice))
	}
}

// Hash hashes the plain text supplied in inputs_buffer and then maps it to a field element
// this proc is used to map arbitrary signals to field element for the sake of proof generation
// inputs holds the hash input as a byte slice, the output slice will contain a 32 byte slice
func (r *RLN) Hash(data []byte) (MerkleNode, error) {
	//  a thin layer on top of the Nim wrapper of the Poseidon hasher
	lenPrefData := appendLength(data)

	hashInputBuffer := toCBufferPtr(lenPrefData)

	var output []byte
	out := toBuffer(output)

	if !bool(C.hash(r.ptr, hashInputBuffer, &out)) {
		return MerkleNode{}, errors.New("failed to hash")
	}

	b := C.GoBytes(unsafe.Pointer(out.ptr), C.int(out.len))

	var result MerkleNode
	copy(result[:], b)

	return result, nil
}

// GenerateProof generates a proof for the RLN given a KeyPair and the index in a merkle tree.
// The output will containt the proof data and should be parsed as |proof<128>|root<32>|epoch<32>|share_x<32>|share_y<32>|nullifier<32>|
// integers wrapped in <> indicate value sizes in bytes
func (r *RLN) GenerateProof(data []byte, key MembershipKeyPair, index MembershipIndex, epoch Epoch) (*RateLimitProof, error) {
	input := serialize(key.IDKey, index, epoch, data)
	inputBuffer := toCBufferPtr(input)

	var output []byte
	out := toBuffer(output)

	if !bool(C.generate_rln_proof(r.ptr, inputBuffer, &out)) {
		return nil, errors.New("could not generate the proof")
	}

	proofBytes := C.GoBytes(unsafe.Pointer(out.ptr), C.int(out.len))

	if len(proofBytes) != 320 {
		return nil, errors.New("invalid proof generated")
	}

	// parse the proof as [ proof<128> | root<32> | epoch<32> | share_x<32> | share_y<32> | nullifier<32> | rln_identifier<32> ]
	proofOffset := 128
	rootOffset := proofOffset + 32
	epochOffset := rootOffset + 32
	shareXOffset := epochOffset + 32
	shareYOffset := shareXOffset + 32
	nullifierOffset := shareYOffset + 32
	rlnIdentifierOffset := nullifierOffset + 32

	var zkproof ZKSNARK
	var proofRoot, shareX, shareY MerkleNode
	var epochR Epoch
	var nullifier Nullifier
	var rlnIdentifier RLNIdentifier

	copy(zkproof[:], proofBytes[0:proofOffset])
	copy(proofRoot[:], proofBytes[proofOffset:rootOffset])
	copy(epochR[:], proofBytes[rootOffset:epochOffset])
	copy(shareX[:], proofBytes[epochOffset:shareXOffset])
	copy(shareY[:], proofBytes[shareXOffset:shareYOffset])
	copy(nullifier[:], proofBytes[shareYOffset:nullifierOffset])
	copy(rlnIdentifier[:], proofBytes[nullifierOffset:rlnIdentifierOffset])

	return &RateLimitProof{
		Proof:         zkproof,
		MerkleRoot:    proofRoot,
		Epoch:         epochR,
		ShareX:        shareX,
		ShareY:        shareY,
		Nullifier:     nullifier,
		RLNIdentifier: rlnIdentifier,
	}, nil
}

// Verify verifies a proof generated for the RLN.
// proof [ proof<128>| root<32>| epoch<32>| share_x<32>| share_y<32>| nullifier<32> | signal_len<8> | signal<var> ]
func (r *RLN) Verify(data []byte, proof RateLimitProof) (bool, error) {
	proofBytes := proof.serialize(data)
	proofBuf := toCBufferPtr(proofBytes)
	res := C.bool(false)
	if !bool(C.verify_rln_proof(r.ptr, proofBuf, &res)) {
		return false, errors.New("could not verify rln proof")
	}

	return bool(res), nil
}

func serializeRoots(roots [][32]byte) []byte {
	var result []byte
	for _, r := range roots {
		result = append(result, r[:]...)
	}
	return result
}

func (r *RLN) VerifyWithRoots(data []byte, proof RateLimitProof, roots [][32]byte) (bool, error) {
	proofBytes := proof.serialize(data)
	proofBuf := toCBufferPtr(proofBytes)

	rootBytes := serializeRoots(roots)
	rootBuf := toCBufferPtr(rootBytes)

	res := C.bool(false)
	if !bool(C.verify_with_roots(r.ptr, proofBuf, rootBuf, &res)) {
		return false, errors.New("could not verify with roots")
	}

	return bool(res), nil
}

// InsertMember adds the member to the tree
func (r *RLN) InsertMember(idComm IDCommitment) error {
	idCommBuffer := toCBufferPtr(idComm[:])
	insertionSuccess := bool(C.set_next_leaf(r.ptr, idCommBuffer))
	if !insertionSuccess {
		return errors.New("could not insert member")
	}
	return nil
}

// DeleteMember removes an IDCommitment key from the tree. The index
// parameter is the position of the id commitment key to be deleted from the tree.
// The deleted id commitment key is replaced with a zero leaf
func (r *RLN) DeleteMember(index MembershipIndex) error {
	deletionSuccess := bool(C.delete_leaf(r.ptr, C.uintptr_t(index)))
	if !deletionSuccess {
		return errors.New("could not delete member")
	}
	return nil
}

// GetMerkleRoot reads the Merkle Tree root after insertion
func (r *RLN) GetMerkleRoot() (MerkleNode, error) {
	var output []byte
	out := toBuffer(output)

	if !bool(C.get_root(r.ptr, &out)) {
		return MerkleNode{}, errors.New("could not get the root")
	}

	b := C.GoBytes(unsafe.Pointer(out.ptr), C.int(out.len))

	if len(b) != 32 {
		return MerkleNode{}, errors.New("wrong output size")
	}

	var result MerkleNode
	copy(result[:], b)

	return result, nil
}

// AddAll adds members to the Merkle tree
func (r *RLN) AddAll(list []IDCommitment) error {
	for _, member := range list {
		if err := r.InsertMember(member); err != nil {
			return err
		}
	}
	return nil
}

// CalcMerkleRoot returns the root of the Merkle tree that is computed from the supplied list
func CalcMerkleRoot(list []IDCommitment) (MerkleNode, error) {
	rln, err := NewRLN()
	if err != nil {
		return MerkleNode{}, err
	}

	// create a Merkle tree
	for _, c := range list {
		if err := rln.InsertMember(c); err != nil {
			return MerkleNode{}, err
		}
	}

	return rln.GetMerkleRoot()
}

// CreateMembershipList produces a list of membership key pairs and also returns the root of a Merkle tree constructed
// out of the identity commitment keys of the generated list. The output of this function is used to initialize a static
// group keys (to test waku-rln-relay in the off-chain mode)
func CreateMembershipList(n int) ([]MembershipKeyPair, MerkleNode, error) {
	// initialize a Merkle tree
	rln, err := NewRLN()
	if err != nil {
		return nil, MerkleNode{}, err
	}

	var output []MembershipKeyPair
	for i := 0; i < n; i++ {
		// generate a keypair
		keypair, err := rln.MembershipKeyGen()
		if err != nil {
			return nil, MerkleNode{}, err
		}

		output = append(output, *keypair)

		// insert the key to the Merkle tree
		if err := rln.InsertMember(keypair.IDCommitment); err != nil {
			return nil, MerkleNode{}, err
		}
	}

	root, err := rln.GetMerkleRoot()
	if err != nil {
		return nil, MerkleNode{}, err
	}

	return output, root, nil
}
