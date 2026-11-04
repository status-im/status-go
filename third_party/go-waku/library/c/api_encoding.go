package main

/*
#include <cgo_utils.h>
*/
import "C"
import (
	"unsafe"

	"github.com/waku-org/go-waku/library"
)

// Decode a waku message using a 32 bytes symmetric key. The key must be a hex encoded string with "0x" prefix
//
//export waku_decode_symmetric
func waku_decode_symmetric(messageJSON *C.char, symmetricKey *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExecNoCtx(func() (string, error) {
		return library.DecodeSymmetric(C.GoString(messageJSON), C.GoString(symmetricKey))
	}, cb, userData)
}

// Decode a waku message using a secp256k1 private key. The key must be a hex encoded string with "0x" prefix
//
//export waku_decode_asymmetric
func waku_decode_asymmetric(messageJSON *C.char, privateKey *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExecNoCtx(func() (string, error) {
		return library.DecodeAsymmetric(C.GoString(messageJSON), C.GoString(privateKey))
	}, cb, userData)
}

// Encrypt a message with a secp256k1 public key.
// publicKey must be a hex string prefixed with "0x" containing a valid secp256k1 public key.
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// The message version will be set to 1
//
//export waku_encode_asymmetric
func waku_encode_asymmetric(messageJSON *C.char, publicKey *C.char, optionalSigningKey *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExecNoCtx(func() (string, error) {
		return library.EncodeAsymmetric(C.GoString(messageJSON), C.GoString(publicKey), C.GoString(optionalSigningKey))
	}, cb, userData)
}

// Encrypt a message with a 32 bytes symmetric key
// symmetricKey must be a hex string prefixed with "0x" containing a 32 bytes symmetric key
// optionalSigningKey is an optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
// The message version will be set to 1
//
//export waku_encode_symmetric
func waku_encode_symmetric(messageJSON *C.char, symmetricKey *C.char, optionalSigningKey *C.char, cb C.WakuCallBack, userData unsafe.Pointer) C.int {
	return singleFnExecNoCtx(func() (string, error) {
		return library.EncodeSymmetric(C.GoString(messageJSON), C.GoString(symmetricKey), C.GoString(optionalSigningKey))
	}, cb, userData)
}
