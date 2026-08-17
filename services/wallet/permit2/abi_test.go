package permit2

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// selector returns the 4-byte selector for a Solidity signature.
func selector(signature string) []byte {
	return crypto.Keccak256([]byte(signature))[:4]
}

func signedDetails(permitType Type) *Details {
	var details *Details
	if permitType == TypePermit2 {
		details = permit2Details()
	} else {
		details = eip2612Details()
	}

	signature := make([]byte, signatureLength)
	for i := range signature {
		signature[i] = byte(i + 1)
	}
	signature[64] = 27
	details.Signature = signature
	return details
}

// TestPackSwapCalldataPermit2 checks the wrapped calldata targets the right function and
// round-trips back to the values the user signed. The Permit2 tuple deliberately omits
// the spender -- Permit2 substitutes msg.sender for it on-chain.
func TestPackSwapCalldataPermit2(t *testing.T) {
	details := signedDetails(TypePermit2)
	diamondCalldata := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	packed, err := PackSwapCalldata(details, diamondCalldata)
	require.NoError(t, err)

	want := selector("callDiamondWithPermit2(bytes,((address,uint256),uint256,uint256),bytes)")
	require.True(t, bytes.HasPrefix(packed, want), "unexpected selector")

	args, err := parsedPermit2ProxyABI.Methods["callDiamondWithPermit2"].Inputs.Unpack(packed[4:])
	require.NoError(t, err)
	require.Equal(t, diamondCalldata, args[0])
	require.Equal(t, details.Signature, args[2])

	permit, ok := args[1].(struct {
		Permitted struct {
			Token  common.Address `json:"token"`
			Amount *big.Int       `json:"amount"`
		} `json:"permitted"`
		Nonce    *big.Int `json:"nonce"`
		Deadline *big.Int `json:"deadline"`
	})
	require.True(t, ok, "unexpected permit tuple shape: %T", args[1])
	require.Equal(t, details.Token, permit.Permitted.Token)
	require.Equal(t, details.Amount, permit.Permitted.Amount)
	require.Equal(t, details.Nonce, permit.Nonce)
	require.Equal(t, details.Deadline, permit.Deadline)
}

// TestPackSwapCalldataEIP2612 checks the token-native variant, where r, s and v are
// passed as separate arguments rather than a packed signature.
func TestPackSwapCalldataEIP2612(t *testing.T) {
	details := signedDetails(TypeEIP2612)
	diamondCalldata := []byte{0x01, 0x02, 0x03}

	packed, err := PackSwapCalldata(details, diamondCalldata)
	require.NoError(t, err)

	want := selector("callDiamondWithEIP2612Signature(address,uint256,uint256,uint8,bytes32,bytes32,bytes)")
	require.True(t, bytes.HasPrefix(packed, want), "unexpected selector")

	args, err := parsedPermit2ProxyABI.Methods["callDiamondWithEIP2612Signature"].Inputs.Unpack(packed[4:])
	require.NoError(t, err)
	require.Equal(t, details.Token, args[0])
	require.Equal(t, details.Amount, args[1])
	require.Equal(t, details.Deadline, args[2])
	require.Equal(t, uint8(27), args[3])

	r := args[4].([32]byte)
	s := args[5].([32]byte)
	require.Equal(t, details.Signature[:32], r[:])
	require.Equal(t, details.Signature[32:64], s[:])
	require.Equal(t, diamondCalldata, args[6])
}

// TestPackSwapCalldataRejectsBadInput makes sure an unsigned or malformed permit can
// never be turned into calldata -- it would produce a transaction that reverts on-chain.
func TestPackSwapCalldataRejectsBadInput(t *testing.T) {
	_, err := PackSwapCalldata(nil, nil)
	require.ErrorIs(t, err, ErrMissingPermitDetails)

	unsigned := permit2Details()
	_, err = PackSwapCalldata(unsigned, nil)
	require.ErrorIs(t, err, ErrInvalidSignatureLength)

	shortSig := permit2Details()
	shortSig.Signature = make([]byte, 64)
	_, err = PackSwapCalldata(shortSig, nil)
	require.ErrorIs(t, err, ErrInvalidSignatureLength)

	unknownType := signedDetails(TypePermit2)
	unknownType.Type = TypeNone
	_, err = PackSwapCalldata(unknownType, nil)
	require.ErrorIs(t, err, ErrUnsupportedPermitType)
}

func TestMaxAllowanceIsFullUint256(t *testing.T) {
	max := MaxAllowance()
	require.Equal(t, 256, max.BitLen())
	require.Equal(t, 0, new(big.Int).Add(max, big.NewInt(1)).Cmp(new(big.Int).Lsh(big.NewInt(1), 256)))
}

func TestPlanNeedsApproval(t *testing.T) {
	require.False(t, (*Plan)(nil).NeedsApproval())
	require.False(t, (&Plan{Details: permit2Details()}).NeedsApproval())
	require.True(t, (&Plan{Details: permit2Details(), ApprovalSpender: testPermit2}).NeedsApproval())
}
