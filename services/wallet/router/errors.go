package router

import "errors"

var (
	ErrorENSRegisterRequires              = errors.New("username and public key are required for ENSRegister")
	ErrorENSRegisterTestNetSTTOnly        = errors.New("only STT is supported for ENSRegister on testnet")
	ErrorENSRegisterSNTOnly               = errors.New("only SNT is supported for ENSRegister")
	ErrorENSReleaseRequires               = errors.New("username is required for ENSRelease")
	ErrorENSSetPubKeyRequires             = errors.New("username and public key are required for ENSSetPubKey")
	ErrorLockedAmountNotSupportedNetwork  = errors.New("locked amount is not supported for the selected network")
	ErrorLockedAmountNotNegative          = errors.New("locked amount must not be negative")
	ErrorLockedAmountExcludesAllSupported = errors.New("all supported chains are excluded, routing impossible")
)
