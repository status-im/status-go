package gowaku

import "github.com/waku-org/go-waku/library"

// DecodeSymmetric decodes a waku message using a 32 bytes symmetric key. The key must be a hex encoded string with "0x" prefix
func DecodeSymmetric(messageJSON string, symmetricKey string) string {
	response, err := library.DecodeSymmetric(messageJSON, symmetricKey)
	return prepareJSONResponse(response, err)
}

// DecodeAsymmetric decodes a waku message using a secp256k1 private key. The key must be a hex encoded string with "0x" prefix
func DecodeAsymmetric(messageJSON string, privateKey string) string {
	response, err := library.DecodeAsymmetric(messageJSON, privateKey)
	return prepareJSONResponse(response, err)
}

// EncodeSymmetric encodes a waku message using a 32 bytes symmetric key. A secp256k1 private key can be used to optionally sign the message.
// The keys must be a hex encoded string with "0x" prefix
func EncodeSymmetric(messageJSON string, symmetricKey string, optionalSigningKey string) string {
	response, err := library.EncodeSymmetric(messageJSON, symmetricKey, optionalSigningKey)
	return prepareJSONResponse(response, err)
}

// EncodeAsymmetric encodes a waku message using a secp256k1 public key. A secp256k1 private key can be used to optionally sign the message.
// The keys must be a hex encoded string with "0x" prefix
func EncodeAsymmetric(messageJSON string, publicKey string, optionalSigningKey string) string {
	response, err := library.EncodeAsymmetric(messageJSON, publicKey, optionalSigningKey)
	return prepareJSONResponse(response, err)
}
