package ens

import (
	"crypto/ecdsa"
)

type Response struct {
	Name            string           `json:"name"`
	Verified        bool             `json:"verified"`
	VerifiedAt      int64            `json:"verifiedAt"`
	Error           error            `json:"error"`
	PublicKey       *ecdsa.PublicKey `json:"-"`
	PublicKeyString string           `json:"publicKey"`
}

type Details struct {
	Name            string `json:"name"`
	PublicKeyString string `json:"publicKey"`
}
