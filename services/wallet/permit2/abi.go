package permit2

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// permit2ProxyABI covers only the LI.FI Permit2Proxy methods this package uses. Note the
// PermitTransferFrom tuple has no `spender` member: Permit2 substitutes msg.sender for it
// on-chain, even though the signed EIP-712 struct does include it.
const permit2ProxyABI = `[
  {
    "name": "callDiamondWithPermit2",
    "type": "function",
    "stateMutability": "payable",
    "inputs": [
      {"name": "_diamondCalldata", "type": "bytes"},
      {"name": "_permit", "type": "tuple", "components": [
        {"name": "permitted", "type": "tuple", "components": [
          {"name": "token", "type": "address"},
          {"name": "amount", "type": "uint256"}
        ]},
        {"name": "nonce", "type": "uint256"},
        {"name": "deadline", "type": "uint256"}
      ]},
      {"name": "_signature", "type": "bytes"}
    ],
    "outputs": [{"name": "", "type": "bytes"}]
  },
  {
    "name": "callDiamondWithEIP2612Signature",
    "type": "function",
    "stateMutability": "payable",
    "inputs": [
      {"name": "tokenAddress", "type": "address"},
      {"name": "amount", "type": "uint256"},
      {"name": "deadline", "type": "uint256"},
      {"name": "v", "type": "uint8"},
      {"name": "r", "type": "bytes32"},
      {"name": "s", "type": "bytes32"},
      {"name": "diamondCalldata", "type": "bytes"}
    ],
    "outputs": [{"name": "", "type": "bytes"}]
  },
  {
    "name": "nextNonce",
    "type": "function",
    "stateMutability": "view",
    "inputs": [{"name": "owner", "type": "address"}],
    "outputs": [{"name": "nonce", "type": "uint256"}]
  }
]`

// erc20PermitABI covers the EIP-2612 surface needed to detect support and build the
// token's EIP-712 domain. `version` is optional in the standard and absent on many
// tokens, so a failed call is expected rather than exceptional.
const erc20PermitABI = `[
  {"name": "DOMAIN_SEPARATOR", "type": "function", "stateMutability": "view", "inputs": [], "outputs": [{"name": "", "type": "bytes32"}]},
  {"name": "PERMIT_TYPEHASH", "type": "function", "stateMutability": "view", "inputs": [], "outputs": [{"name": "", "type": "bytes32"}]},
  {"name": "nonces", "type": "function", "stateMutability": "view", "inputs": [{"name": "owner", "type": "address"}], "outputs": [{"name": "", "type": "uint256"}]},
  {"name": "name", "type": "function", "stateMutability": "view", "inputs": [], "outputs": [{"name": "", "type": "string"}]},
  {"name": "version", "type": "function", "stateMutability": "view", "inputs": [], "outputs": [{"name": "", "type": "string"}]}
]`

// erc20AllowanceABI is the single ERC-20 method needed to tell whether the user has
// already approved the Permit2 singleton for a token.
const erc20AllowanceABI = `[
  {"name": "allowance", "type": "function", "stateMutability": "view", "inputs": [{"name": "owner", "type": "address"}, {"name": "spender", "type": "address"}], "outputs": [{"name": "", "type": "uint256"}]}
]`

// parsedPermit2ProxyABI and parsedERC20PermitABI are parsed once at init; the JSON above
// is a compile-time constant so a parse failure is a programming error.
var (
	parsedPermit2ProxyABI   = mustParseABI(permit2ProxyABI)
	parsedERC20PermitABI    = mustParseABI(erc20PermitABI)
	parsedERC20AllowanceABI = mustParseABI(erc20AllowanceABI)
)

func mustParseABI(definition string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		panic("permit2: invalid embedded ABI: " + err.Error())
	}
	return parsed
}

// tokenPermissions and permitTransferFrom mirror ISignatureTransfer's structs for ABI
// encoding. Field order matters, go-ethereum packs tuples positionally.
type tokenPermissions struct {
	Token  common.Address `abi:"token"`
	Amount *big.Int       `abi:"amount"`
}

type permitTransferFrom struct {
	Permitted tokenPermissions `abi:"permitted"`
	Nonce     *big.Int         `abi:"nonce"`
	Deadline  *big.Int         `abi:"deadline"`
}

// PackSwapCalldata wraps the LI.FI diamond calldata in the Permit2Proxy call carrying the
// user's permit signature, producing the calldata for the single transaction that both
// pulls the tokens and performs the swap.
func PackSwapCalldata(details *Details, diamondCalldata []byte) ([]byte, error) {
	if details == nil {
		return nil, ErrMissingPermitDetails
	}
	if len(details.Signature) != signatureLength {
		return nil, ErrInvalidSignatureLength
	}

	switch details.Type {
	case TypePermit2:
		return parsedPermit2ProxyABI.Pack(
			"callDiamondWithPermit2",
			diamondCalldata,
			permitTransferFrom{
				Permitted: tokenPermissions{Token: details.Token, Amount: details.Amount},
				Nonce:     details.Nonce,
				Deadline:  details.Deadline,
			},
			details.Signature,
		)

	case TypeEIP2612:
		var r, s [32]byte
		copy(r[:], details.Signature[:32])
		copy(s[:], details.Signature[32:64])
		return parsedPermit2ProxyABI.Pack(
			"callDiamondWithEIP2612Signature",
			details.Token,
			details.Amount,
			details.Deadline,
			details.Signature[64],
			r,
			s,
			diamondCalldata,
		)
	}

	return nil, ErrUnsupportedPermitType
}
