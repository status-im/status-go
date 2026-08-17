package permit2

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	testPermit2      = common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3")
	testPermit2Proxy = common.HexToAddress("0x89c6340B1a1f4b25D36cd8B063D49045caF3f818")
	testToken        = common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	testOwner        = common.HexToAddress("0x1111111111111111111111111111111111111111")
)

// word left-pads a value to a 32-byte ABI word.
func word(b []byte) []byte { return common.LeftPadBytes(b, 32) }

func concat(chunks ...[]byte) []byte {
	var out []byte
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}

// eip712Digest recreates the \x19\x01 envelope by hand.
func eip712Digest(domainSeparator, structHash common.Hash) common.Hash {
	return crypto.Keccak256Hash([]byte{0x19, 0x01}, domainSeparator.Bytes(), structHash.Bytes())
}

func permit2Details() *Details {
	return &Details{
		Type:     TypePermit2,
		ChainID:  1,
		Owner:    testOwner,
		Token:    testToken,
		Amount:   big.NewInt(1_500_000),
		Spender:  testPermit2Proxy,
		Permit2:  testPermit2,
		Nonce:    big.NewInt(7),
		Deadline: big.NewInt(1_800_000_000),
	}
}

func eip2612Details() *Details {
	return &Details{
		Type:         TypeEIP2612,
		ChainID:      1,
		Owner:        testOwner,
		Token:        testToken,
		Amount:       big.NewInt(1_500_000),
		Spender:      testPermit2Proxy,
		Nonce:        big.NewInt(3),
		Deadline:     big.NewInt(1_800_000_000),
		TokenName:    "USD Coin",
		TokenVersion: "2",
	}
}

// TestPermit2DigestMatchesManualEncoding derives the digest straight from the EIP-712
// and Permit2 specs and checks the typed-data path produces the same thing. A mismatch
// here means we would hand the user a signature Permit2 cannot verify.
func TestPermit2DigestMatchesManualEncoding(t *testing.T) {
	details := permit2Details()

	// Permit2's domain carries no version field.
	domainTypehash := crypto.Keccak256Hash([]byte(
		"EIP712Domain(string name,uint256 chainId,address verifyingContract)"))
	domainSeparator := crypto.Keccak256Hash(concat(
		domainTypehash.Bytes(),
		crypto.Keccak256([]byte("Permit2")),
		word(big.NewInt(1).Bytes()),
		word(testPermit2.Bytes()),
	))

	tokenPermissionsTypehash := crypto.Keccak256Hash([]byte(
		"TokenPermissions(address token,uint256 amount)"))
	tokenPermissionsHash := crypto.Keccak256Hash(concat(
		tokenPermissionsTypehash.Bytes(),
		word(testToken.Bytes()),
		word(details.Amount.Bytes()),
	))

	// The referenced struct is appended to the primary type, per EIP-712 encodeType.
	permitTypehash := crypto.Keccak256Hash([]byte(
		"PermitTransferFrom(TokenPermissions permitted,address spender,uint256 nonce,uint256 deadline)" +
			"TokenPermissions(address token,uint256 amount)"))
	structHash := crypto.Keccak256Hash(concat(
		permitTypehash.Bytes(),
		tokenPermissionsHash.Bytes(),
		word(testPermit2Proxy.Bytes()),
		word(details.Nonce.Bytes()),
		word(details.Deadline.Bytes()),
	))

	got, err := details.Digest()
	require.NoError(t, err)
	require.Equal(t, eip712Digest(domainSeparator, structHash), got)
}

// TestEIP2612DigestMatchesManualEncoding does the same for the token-native variant.
func TestEIP2612DigestMatchesManualEncoding(t *testing.T) {
	details := eip2612Details()

	domainTypehash := crypto.Keccak256Hash([]byte(
		"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	domainSeparator := crypto.Keccak256Hash(concat(
		domainTypehash.Bytes(),
		crypto.Keccak256([]byte("USD Coin")),
		crypto.Keccak256([]byte("2")),
		word(big.NewInt(1).Bytes()),
		word(testToken.Bytes()),
	))

	permitTypehash := crypto.Keccak256Hash([]byte(
		"Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"))
	structHash := crypto.Keccak256Hash(concat(
		permitTypehash.Bytes(),
		word(testOwner.Bytes()),
		word(testPermit2Proxy.Bytes()),
		word(details.Amount.Bytes()),
		word(details.Nonce.Bytes()),
		word(details.Deadline.Bytes()),
	))

	got, err := details.Digest()
	require.NoError(t, err)
	require.Equal(t, eip712Digest(domainSeparator, structHash), got)

	// The same domain separator is what resolveEIP2612 compares against the token's own,
	// so the two derivations have to agree.
	resolved, err := domainSeparatorFor(details)
	require.NoError(t, err)
	require.Equal(t, domainSeparator, resolved)
}

// TestEIP2612DomainSeparatorIsVersionSensitive guards the check that makes version
// guessing safe: a wrong version must produce a different separator, so it gets rejected
// rather than silently signed.
func TestEIP2612DomainSeparatorIsVersionSensitive(t *testing.T) {
	v2 := eip2612Details()
	v1 := eip2612Details()
	v1.TokenVersion = "1"

	sep2, err := domainSeparatorFor(v2)
	require.NoError(t, err)
	sep1, err := domainSeparatorFor(v1)
	require.NoError(t, err)
	require.NotEqual(t, sep1, sep2)
}

func TestDigestRejectsUnsupportedType(t *testing.T) {
	details := permit2Details()
	details.Type = TypeNone
	_, err := details.Digest()
	require.ErrorIs(t, err, ErrUnsupportedPermitType)

	var nilDetails *Details
	_, err = nilDetails.TypedData()
	require.ErrorIs(t, err, ErrMissingPermitDetails)
}

func TestCopyIsDeep(t *testing.T) {
	original := permit2Details()
	original.Signature = make([]byte, signatureLength)
	original.Signature[0] = 0xAA

	clone := original.Copy()
	clone.Amount.SetInt64(1)
	clone.Nonce.SetInt64(99)
	clone.Deadline.SetInt64(5)
	clone.Signature[0] = 0xBB

	require.Equal(t, int64(1_500_000), original.Amount.Int64())
	require.Equal(t, int64(7), original.Nonce.Int64())
	require.Equal(t, int64(1_800_000_000), original.Deadline.Int64())
	require.Equal(t, byte(0xAA), original.Signature[0])

	require.Nil(t, (*Details)(nil).Copy())
}
