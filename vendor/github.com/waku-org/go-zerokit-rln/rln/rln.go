package rln

import "C"
import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/waku-org/go-zerokit-rln/rln/link"
	"github.com/waku-org/go-zerokit-rln/rln/resources"
)

// RLN represents the context used for rln.
type RLN struct {
	w *link.RLNWrapper
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

	r.w, err = link.NewWithParams(depth, wasm, zkey, verifKey)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// NewRLNWithParams generates an instance of RLN. An instance supports both zkSNARKs logics
// and Merkle tree data structure and operations. The parameter `depth“ indicates the depth of Merkle tree
func NewRLNWithParams(depth int, wasm []byte, zkey []byte, verifKey []byte) (*RLN, error) {
	r := &RLN{}
	var err error

	r.w, err = link.NewWithParams(depth, wasm, zkey, verifKey)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// NewRLNWithFolder generates an instance of RLN. An instance supports both zkSNARKs logics
// and Merkle tree data structure and operations. The parameter `deptk` indicates the depth of Merkle tree
// The parameter “
func NewRLNWithFolder(depth int, resourcesFolderPath string) (*RLN, error) {
	r := &RLN{}

	var err error

	r.w, err = link.NewWithFolder(depth, resourcesFolderPath)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// MembershipKeyGen generates a IdentityCredential that can be used for the
// registration into the rln membership contract. Returns an error if the key generation fails
func (r *RLN) MembershipKeyGen() (*IdentityCredential, error) {
	generatedKeys := r.w.ExtendedKeyGen()
	if generatedKeys == nil {
		return nil, errors.New("error in key generation")
	}

	key := &IdentityCredential{
		IDTrapdoor:   [32]byte{},
		IDNullifier:  [32]byte{},
		IDSecretHash: [32]byte{},
		IDCommitment: [32]byte{},
	}

	if len(generatedKeys) != 32*4 {
		return nil, errors.New("generated keys are of invalid length")
	}

	copy(key.IDTrapdoor[:], generatedKeys[:32])
	copy(key.IDNullifier[:], generatedKeys[32:64])
	copy(key.IDSecretHash[:], generatedKeys[64:96])
	copy(key.IDCommitment[:], generatedKeys[96:128])

	return key, nil
}

// appendLength returns length prefixed version of the input with the following format
// [len<8>|input<var>], the len is a 8 byte value serialized in little endian

func appendLength(input []byte) []byte {
	inputLen := make([]byte, 8)
	binary.LittleEndian.PutUint64(inputLen, uint64(len(input)))
	return append(inputLen, input...)
}

func (r *RLN) Sha256(data []byte) (MerkleNode, error) {
	lenPrefData := appendLength(data)

	b, err := r.w.Hash(lenPrefData)
	if err != nil {
		return MerkleNode{}, err
	}

	var result MerkleNode
	copy(result[:], b)

	return result, nil
}

func (r *RLN) Poseidon(input ...[]byte) (MerkleNode, error) {
	data := serializeSlice(input)

	inputLen := make([]byte, 8)
	binary.LittleEndian.PutUint64(inputLen, uint64(len(input)))

	lenPrefData := append(inputLen, data...)

	b, err := r.w.PoseidonHash(lenPrefData)
	if err != nil {
		return MerkleNode{}, err
	}

	var result MerkleNode
	copy(result[:], b)

	return result, nil
}

func (r *RLN) ExtractMetadata(proof RateLimitProof) (ProofMetadata, error) {
	externalNullifierRes, err := r.Poseidon(proof.Epoch[:], proof.RLNIdentifier[:])
	if err != nil {
		return ProofMetadata{}, fmt.Errorf("could not construct the external nullifier: %w", err)
	}

	return ProofMetadata{
		Nullifier:         proof.Nullifier,
		ShareX:            proof.ShareX,
		ShareY:            proof.ShareY,
		ExternalNullifier: externalNullifierRes,
	}, nil
}

// GenerateProof generates a proof for the RLN given a KeyPair and the index in a merkle tree.
// The output will containt the proof data and should be parsed as |proof<128>|root<32>|epoch<32>|share_x<32>|share_y<32>|nullifier<32>|
// integers wrapped in <> indicate value sizes in bytes
func (r *RLN) GenerateProof(data []byte, key IdentityCredential, index MembershipIndex, epoch Epoch) (*RateLimitProof, error) {
	input := serialize(key.IDSecretHash, index, epoch, data)
	proofBytes, err := r.w.GenerateRLNProof(input)
	if err != nil {
		return nil, err
	}

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

func serialize32(roots [][32]byte) []byte {
	var result []byte
	for _, r := range roots {
		result = append(result, r[:]...)
	}
	return result
}

func serializeSlice(roots [][]byte) []byte {
	var result []byte
	for _, r := range roots {
		result = append(result, r[:]...)
	}
	return result
}

func serializeCommitments(commitments []IDCommitment) []byte {
	// serializes a seq of IDCommitments to a byte seq
	// the serialization is based on https://github.com/status-im/nwaku/blob/37bd29fbc37ce5cf636734e7dd410b1ed27b88c8/waku/v2/protocol/waku_rln_relay/rln.nim#L142
	// the order of serialization is |id_commitment_len<8>|id_commitment<var>|
	var result []byte

	inputLen := make([]byte, 8)
	binary.LittleEndian.PutUint64(inputLen, uint64(len(commitments)))
	result = append(result, inputLen...)

	for _, idComm := range commitments {
		result = append(result, idComm[:]...)
	}

	return result
}

// proof [ proof<128>| root<32>| epoch<32>| share_x<32>| share_y<32>| nullifier<32> | signal_len<8> | signal<var> ]
// validRoots should contain a sequence of roots in the acceptable windows.
// As default, it is set to an empty sequence of roots. This implies that the validity check for the proof's root is skipped
func (r *RLN) Verify(data []byte, proof RateLimitProof, roots ...[32]byte) (bool, error) {
	proofBytes := proof.serialize(data)
	rootBytes := serialize32(roots)

	res, err := r.w.VerifyWithRoots(proofBytes, rootBytes)
	if err != nil {
		return false, err
	}

	return bool(res), nil
}

// InsertMember adds the member to the tree
func (r *RLN) InsertMember(idComm IDCommitment) error {
	insertionSuccess := r.w.SetNextLeaf(idComm[:])
	if !insertionSuccess {
		return errors.New("could not insert member")
	}
	return nil
}

// Insert multiple members i.e., identity commitments starting from index
// This proc is atomic, i.e., if any of the insertions fails, all the previous insertions are rolled back
func (r *RLN) InsertMembers(index MembershipIndex, idComms []IDCommitment) error {
	idCommBytes := serializeCommitments(idComms)
	insertionSuccess := r.w.SetLeavesFrom(index, idCommBytes)
	if !insertionSuccess {
		return errors.New("could not insert members")
	}
	return nil
}

// DeleteMember removes an IDCommitment key from the tree. The index
// parameter is the position of the id commitment key to be deleted from the tree.
// The deleted id commitment key is replaced with a zero leaf
func (r *RLN) DeleteMember(index MembershipIndex) error {
	deletionSuccess := r.w.DeleteLeaf(index)
	if !deletionSuccess {
		return errors.New("could not delete member")
	}
	return nil
}

// GetMerkleRoot reads the Merkle Tree root after insertion
func (r *RLN) GetMerkleRoot() (MerkleNode, error) {
	b, err := r.w.GetRoot()
	if err != nil {
		return MerkleNode{}, err
	}

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
func CreateMembershipList(n int) ([]IdentityCredential, MerkleNode, error) {
	// initialize a Merkle tree
	rln, err := NewRLN()
	if err != nil {
		return nil, MerkleNode{}, err
	}

	var output []IdentityCredential
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
