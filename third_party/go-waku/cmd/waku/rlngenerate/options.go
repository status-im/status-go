package rlngenerate

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
)

// Options are settings used to create RLN credentials.
type Options struct {
	CredentialsPath           string
	CredentialsPassword       string
	ETHPrivateKey             *ecdsa.PrivateKey
	ETHClientAddress          string
	MembershipContractAddress common.Address
	ETHGasLimit               uint64
	ETHNonce                  string
	ETHGasPrice               string
	ETHGasFeeCap              string
	ETHGasTipCap              string
}
