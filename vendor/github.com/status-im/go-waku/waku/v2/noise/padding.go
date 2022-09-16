package noise

import (
	"errors"
)

// PKCS7_Pad pads a payload according to PKCS#7 as per
// RFC 5652 https://datatracker.ietf.org/doc/html/rfc5652#section-6.3
func PKCS7_Pad(payload []byte, paddingSize int) ([]byte, error) {
	if paddingSize >= 256 {
		return nil, errors.New("invalid padding size")
	}

	k := paddingSize - (len(payload) % paddingSize)

	var padVal int
	if k != 0 {
		padVal = k
	} else {
		padVal = paddingSize
	}

	padding := make([]byte, padVal)
	for i := range padding {
		padding[i] = byte(padVal)
	}

	return append(payload, padding...), nil
}

// PKCS7_Unpad unpads a payload according to PKCS#7 as per
// RFC 5652 https://datatracker.ietf.org/doc/html/rfc5652#section-6.3
func PKCS7_Unpad(payload []byte, paddingSize int) ([]byte, error) {
	if paddingSize >= 256 {
		return nil, errors.New("invalid padding size")
	}

	if len(payload) == 0 {
		return nil, nil // empty payload
	}

	high := len(payload) - 1
	k := payload[high]

	unpadded := payload[0:(high + 1 - int(k))]

	return unpadded, nil
}
