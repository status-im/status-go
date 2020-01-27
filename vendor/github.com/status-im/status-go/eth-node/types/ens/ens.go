package enstypes

import "crypto/ecdsa"

type ENSVerifier interface {
	// CheckBatch verifies that a registered ENS name matches the expected public key
	CheckBatch(ensDetails []ENSDetails, rpcEndpoint, contractAddress string) (map[string]*ENSResponse, error)
}

type ENSDetails struct {
	Name            string `json:"name"`
	PublicKeyString string `json:"publicKey"`
	Clock           uint64 `json:"clock"`
}

type ENSResponse struct {
	Name            string           `json:"name"`
	Verified        bool             `json:"verified"`
	VerifiedAt      uint64           `json:"verifiedAt"`
	Error           error            `json:"error"`
	PublicKey       *ecdsa.PublicKey `json:"-"`
	PublicKeyString string           `json:"publicKey"`
}
