// Package permit2 builds the off-chain signatures that let a swap pull the user's
// tokens without a separate on-chain approval transaction.
//
// Two variants are supported, both routed through LI.FI's Permit2Proxy, which pulls the
// tokens and forwards the swap calldata to the LI.FI diamond:
//
//   - EIP-2612: the token's own permit() is called by the proxy. Needs no prior
//     approval of any kind, but only works for tokens implementing the standard.
//   - Permit2: Uniswap's singleton transfers on the proxy's behalf. Works for any
//     ERC-20, but the user must have approved the Permit2 singleton once beforehand.
//
// In both cases the proxy requires msg.sender to be the permit signer, so the user still
// sends and pays for the swap transaction. What goes away is the separate approval tx and
// the wait for it to be mined.
package permit2

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	signercore "github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/status-im/status-go/services/typeddata"
)

// Type identifies which permit variant a swap uses.
type Type int

const (
	// TypeNone means the swap falls back to the regular approve-then-swap flow.
	TypeNone Type = iota
	// TypeEIP2612 uses the token's own permit(); no prior approval is needed.
	TypeEIP2612
	// TypePermit2 uses Uniswap's Permit2 singleton; requires a one-time approval of it.
	TypePermit2
)

func (t Type) String() string {
	switch t {
	case TypeEIP2612:
		return "eip2612"
	case TypePermit2:
		return "permit2"
	}
	return "none"
}

var (
	ErrUnsupportedPermitType = errors.New("unsupported permit type")
	ErrMissingPermitDetails  = errors.New("missing permit details")
)

// Details carries what's needed to build the digest the user signs, render it for
// confirmation and pack the resulting signature into the swap calldata.
type Details struct {
	Type Type

	ChainID uint64
	Owner   common.Address
	Token   common.Address
	Amount  *big.Int

	// Spender is the Permit2Proxy for both variants, the contract that ends up holding the
	// tokens and forwarding the swap to the diamond.
	Spender common.Address
	// Permit2 is the singleton the PermitTransferFrom is signed against. TypePermit2 only.
	Permit2 common.Address

	Nonce    *big.Int
	Deadline *big.Int

	// TokenName and TokenVersion make up the token's EIP-712 domain. TypeEIP2612 only;
	// resolved by checking candidate domains against the token's own DOMAIN_SEPARATOR.
	TokenName    string
	TokenVersion string

	// Signature is the 65-byte r||s||v signature over Digest(), filled in once the
	// client has signed it.
	Signature []byte
}

// TypedData builds the EIP-712 payload the user is asked to sign, and that the client
// renders at confirmation time in place of the bare hash.
func (d *Details) TypedData() (signercore.TypedData, error) {
	if d == nil {
		return signercore.TypedData{}, ErrMissingPermitDetails
	}

	chainID := (*math.HexOrDecimal256)(new(big.Int).SetUint64(d.ChainID))

	switch d.Type {
	case TypePermit2:
		// Permit2's domain deliberately carries no version field, and the signed struct
		// includes the spender even though the on-chain struct omits it (Permit2
		// substitutes msg.sender there).
		return signercore.TypedData{
			Types: signercore.Types{
				"EIP712Domain": []signercore.Type{
					{Name: "name", Type: "string"},
					{Name: "chainId", Type: "uint256"},
					{Name: "verifyingContract", Type: "address"},
				},
				"TokenPermissions": []signercore.Type{
					{Name: "token", Type: "address"},
					{Name: "amount", Type: "uint256"},
				},
				"PermitTransferFrom": []signercore.Type{
					{Name: "permitted", Type: "TokenPermissions"},
					{Name: "spender", Type: "address"},
					{Name: "nonce", Type: "uint256"},
					{Name: "deadline", Type: "uint256"},
				},
			},
			PrimaryType: "PermitTransferFrom",
			Domain: signercore.TypedDataDomain{
				Name:              "Permit2",
				ChainId:           chainID,
				VerifyingContract: d.Permit2.Hex(),
			},
			Message: signercore.TypedDataMessage{
				"permitted": signercore.TypedDataMessage{
					"token":  d.Token.Hex(),
					"amount": d.Amount,
				},
				"spender":  d.Spender.Hex(),
				"nonce":    d.Nonce,
				"deadline": d.Deadline,
			},
		}, nil

	case TypeEIP2612:
		return signercore.TypedData{
			Types: signercore.Types{
				"EIP712Domain": []signercore.Type{
					{Name: "name", Type: "string"},
					{Name: "version", Type: "string"},
					{Name: "chainId", Type: "uint256"},
					{Name: "verifyingContract", Type: "address"},
				},
				"Permit": []signercore.Type{
					{Name: "owner", Type: "address"},
					{Name: "spender", Type: "address"},
					{Name: "value", Type: "uint256"},
					{Name: "nonce", Type: "uint256"},
					{Name: "deadline", Type: "uint256"},
				},
			},
			PrimaryType: "Permit",
			Domain: signercore.TypedDataDomain{
				Name:              d.TokenName,
				Version:           d.TokenVersion,
				ChainId:           chainID,
				VerifyingContract: d.Token.Hex(),
			},
			Message: signercore.TypedDataMessage{
				"owner":    d.Owner.Hex(),
				"spender":  d.Spender.Hex(),
				"value":    d.Amount,
				"nonce":    d.Nonce,
				"deadline": d.Deadline,
			},
		}, nil
	}

	return signercore.TypedData{}, ErrUnsupportedPermitType
}

// Digest is the 32-byte EIP-712 hash the client signs. Same shape as a transaction hash,
// which is why the existing r/s/v signing pipeline (Keycard included) carries it unchanged.
func (d *Details) Digest() (common.Hash, error) {
	td, err := d.TypedData()
	if err != nil {
		return common.Hash{}, err
	}
	return typeddata.HashTypedDataV4(td, new(big.Int).SetUint64(d.ChainID))
}

// MaxAllowance is the amount approved to the Permit2 singleton when a token has no
// allowance yet. Approving the maximum makes it a one-time cost: every later swap of that
// token then needs only a signature. Permit2 still bounds what each spender may take, per
// signed permit.
func MaxAllowance() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

// domainSeparatorFor computes the EIP-712 domain separator the given details would sign
// against, so it can be compared with the one the token reports on-chain.
func domainSeparatorFor(d *Details) (common.Hash, error) {
	td, err := d.TypedData()
	if err != nil {
		return common.Hash{}, err
	}
	separator, err := td.HashStruct("EIP712Domain", td.Domain.Map())
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(separator), nil
}

// Copy deep-copies the details so a Path can be duplicated without sharing state.
func (d *Details) Copy() *Details {
	if d == nil {
		return nil
	}

	newDetails := *d
	if d.Amount != nil {
		newDetails.Amount = new(big.Int).Set(d.Amount)
	}
	if d.Nonce != nil {
		newDetails.Nonce = new(big.Int).Set(d.Nonce)
	}
	if d.Deadline != nil {
		newDetails.Deadline = new(big.Int).Set(d.Deadline)
	}
	if d.Signature != nil {
		newDetails.Signature = make([]byte, len(d.Signature))
		copy(newDetails.Signature, d.Signature)
	}
	return &newDetails
}
