package rln

/*
#include "./librln.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// RLN represents the context used for rln.
type RLN struct {
	ptr *C.RLN
}

func toCBufferPtr(input []byte) *C.Buffer {
	buf := toBuffer(input)

	size := int(unsafe.Sizeof(buf))
	in := (*C.Buffer)(C.malloc(C.size_t(size)))
	*in = buf

	return in
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

func NewWithParams(depth int, wasm []byte, zkey []byte, verifKey []byte) (*RLN, error) {
	wasmBuffer := toCBufferPtr(wasm)
	zkeyBuffer := toCBufferPtr(zkey)
	verifKeyBuffer := toCBufferPtr(verifKey)
	r := &RLN{}

	if !bool(C.new_with_params(C.uintptr_t(depth), wasmBuffer, zkeyBuffer, verifKeyBuffer, &r.ptr)) {
		return nil, errors.New("failed to initialize")
	}

	return r, nil
}

func NewWithFolder(depth int, resourcesFolderPath string) (*RLN, error) {
	r := &RLN{}
	pathBuffer := toCBufferPtr([]byte(resourcesFolderPath))
	if !bool(C.new(C.uintptr_t(depth), pathBuffer, &r.ptr)) {
		return nil, errors.New("failed to initialize")
	}
	return r, nil
}

func (r *RLN) ExtendedKeyGen() []byte {
	buffer := toBuffer([]byte{})
	if !bool(C.extended_key_gen(r.ptr, &buffer)) {
		return nil
	}
	return C.GoBytes(unsafe.Pointer(buffer.ptr), C.int(buffer.len))
}

func (r *RLN) Hash(input []byte) ([]byte, error) {
	inpBuff := toCBufferPtr(input)

	var output []byte
	out := toBuffer(output)

	if !bool(C.hash(inpBuff, &out)) {
		return nil, errors.New("failed to hash")
	}

	return C.GoBytes(unsafe.Pointer(out.ptr), C.int(out.len)), nil
}

func (r *RLN) PoseidonHash(input []byte) ([]byte, error) {
	inpBuff := toCBufferPtr(input)

	var output []byte
	out := toBuffer(output)

	if !bool(C.poseidon_hash(inpBuff, &out)) {
		return nil, errors.New("error in poseidon hash")
	}

	return C.GoBytes(unsafe.Pointer(out.ptr), C.int(out.len)), nil
}

func (r *RLN) GenerateRLNProof(input []byte) ([]byte, error) {
	inputBuffer := toCBufferPtr(input)

	var output []byte
	out := toBuffer(output)

	if !bool(C.generate_rln_proof(r.ptr, inputBuffer, &out)) {
		return nil, errors.New("could not generate the proof")
	}

	return C.GoBytes(unsafe.Pointer(out.ptr), C.int(out.len)), nil
}

func (r *RLN) VerifyWithRoots(input []byte, roots []byte) (bool, error) {
	proofBuf := toCBufferPtr(input)
	rootBuf := toCBufferPtr(roots)

	res := C.bool(false)
	if !bool(C.verify_with_roots(r.ptr, proofBuf, rootBuf, &res)) {
		return false, errors.New("could not verify with roots")
	}

	return bool(res), nil
}

func (r *RLN) SetNextLeaf(idcommitment []byte) bool {
	buff := toCBufferPtr(idcommitment[:])
	return bool(C.set_next_leaf(r.ptr, buff))
}

func (r *RLN) SetLeavesFrom(index uint, idcommitments []byte) bool {
	idCommBuffer := toCBufferPtr(idcommitments)
	return bool(C.set_leaves_from(r.ptr, C.uintptr_t(index), idCommBuffer))
}

func (r *RLN) DeleteLeaf(index uint) bool {
	return bool(C.delete_leaf(r.ptr, C.uintptr_t(index)))
}

func (r *RLN) GetRoot() ([]byte, error) {
	var output []byte
	out := toBuffer(output)

	if !bool(C.get_root(r.ptr, &out)) {
		return nil, errors.New("could not get the root")
	}

	return C.GoBytes(unsafe.Pointer(out.ptr), C.int(out.len)), nil
}
