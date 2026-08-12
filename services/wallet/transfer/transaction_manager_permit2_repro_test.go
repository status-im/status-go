package transfer

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	crypto2 "github.com/status-im/status-go/internal/crypto"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/services/wallet/permit2"
	"github.com/status-im/status-go/services/wallet/requests"
	pathProcessorCommon "github.com/status-im/status-go/services/wallet/router/pathprocessor/common"
	"github.com/status-im/status-go/services/wallet/router/routes"
)

// ---------------------------------------------------------------------------
// Finding 1: signature assembly in getSignatureForTxHash
//
//	copy(signature[64-len(rBytes):64], sBytes)
//
// The S offset is derived from len(rBytes) instead of len(sBytes), so as soon as R and S
// hex-decode to different lengths the assembled 65-byte signature is silently corrupted.
// The corrected sibling permitSignatureFor (same file, ~line 357) uses len(sBytes).
//
// Reachability: requests.SignatureDetails.Validate() only checks *string* lengths
// (len(R) == 64 && len(S) == 64 && len(V) == 2), so R/S that are simply "too short" never
// reach the assembly code - that variant of the bug is unreachable and the last table case
// below documents it as such (it passes today).
//
// The bug IS reachable, however, because getSignatureForTxHash ignores the hex decode
// errors ("rBytes, _ := hex.DecodeString(...)"). A 64-character R or S that contains
// invalid hex passes Validate(), decodes to a *partial* byte slice, and the error is
// dropped - producing exactly the mismatched-length situation the assembly mishandles.
// Both inputs come straight from the client over the RouterSendTransactionsParams RPC.
//
// Expected behaviour asserted here: R right-aligned in bytes [0,32), S right-aligned in
// [32,64), V in [64] - and malformed hex surfaced as an error rather than swallowed.
// ---------------------------------------------------------------------------

// rsBytes builds recognizable, non-uniform R and S byte values so a one-byte shift in the
// assembled signature is visible in the diff.
func rsBytes() (rBytes, sBytes []byte) {
	rBytes = make([]byte, 32)
	sBytes = make([]byte, 32)
	for i := 0; i < 32; i++ {
		rBytes[i] = byte(i + 1)    // 0x01 .. 0x20
		sBytes[i] = byte(0x81 + i) // 0x81 .. 0xa0
	}
	return rBytes, sBytes
}

// wantSignature is the layout the callers (and the corrected sibling) expect.
func wantSignature(rBytes, sBytes []byte, v byte) []byte {
	sig := make([]byte, crypto2.SignatureLength)
	copy(sig[32-len(rBytes):32], rBytes)
	copy(sig[64-len(sBytes):64], sBytes)
	sig[64] = v
	return sig
}

func TestGetSignatureForTxHash_RSAssembly(t *testing.T) {
	rFull, sFull := rsBytes()

	const txHash = "0xc8e7a34af766c4ba9dc9b3d49939806fbf41fa01250c5a26afa5659e87b2020b"

	tests := []struct {
		name    string
		r       string
		s       string
		v       string
		wantR   []byte // bytes the input actually carries for R
		wantS   []byte // bytes the input actually carries for S
		wantV   byte
		wantErr bool
	}{
		{
			// Control: 32/32. Passes on current code.
			name:  "R 32 bytes / S 32 bytes",
			r:     hex.EncodeToString(rFull),
			s:     hex.EncodeToString(sFull),
			v:     "01",
			wantR: rFull,
			wantS: sFull,
			wantV: 1,
		},
		{
			// R decodes to 31 bytes (64 chars, so Validate passes; last pair is not hex, so
			// hex.DecodeString returns 31 bytes + an error that the code drops).
			// Current code then writes S at signature[33:64] instead of [32:64].
			name:    "R 31 bytes / S 32 bytes",
			r:       hex.EncodeToString(rFull[:31]) + "zz",
			s:       hex.EncodeToString(sFull),
			v:       "01",
			wantR:   rFull[:31],
			wantS:   sFull,
			wantV:   1,
			wantErr: true, // malformed hex must not be silently accepted
		},
		{
			// S decodes to 31 bytes. Current code writes it at signature[32:63] (left
			// aligned) instead of right-aligning it at [33:64).
			name:    "R 32 bytes / S 31 bytes",
			r:       hex.EncodeToString(rFull),
			s:       hex.EncodeToString(sFull[:31]) + "zz",
			v:       "00",
			wantR:   rFull,
			wantS:   sFull[:31],
			wantV:   0,
			wantErr: true, // malformed hex must not be silently accepted
		},
		{
			// Documents the *unreachable* variant: a genuinely short R (62 chars) is
			// rejected by SignatureDetails.Validate() before the assembly runs.
			// This case passes on current code and is here to keep the reachability
			// statement above honest.
			name:    "R shorter than 64 hex chars is rejected by Validate",
			r:       hex.EncodeToString(rFull[:31]),
			s:       hex.EncodeToString(sFull),
			v:       "01",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signatures := map[string]requests.SignatureDetails{
				txHash: {R: tt.r, S: tt.s, V: tt.v},
			}

			sig, err := getSignatureForTxHash(txHash, signatures)

			if tt.wantErr {
				assert.Error(t, err, "malformed signature details must be rejected, not silently truncated")
			} else {
				require.NoError(t, err)
			}

			if err != nil {
				// Rejected: nothing further to check.
				return
			}

			require.Len(t, sig, crypto2.SignatureLength)
			assert.Equal(t,
				hex.EncodeToString(wantSignature(tt.wantR, tt.wantS, tt.wantV)),
				hex.EncodeToString(sig),
				"R must be right-aligned in [0,32), S right-aligned in [32,64), V in [64]")
		})
	}
}

// TestGetSignatureForTxHash_AgreesWithPermitSignatureFor pins the two assemblers against
// each other. permitSignatureFor is the corrected sibling: it offsets S by len(sBytes) and
// surfaces the hex decode error. For identical input the r||s halves must match.
func TestGetSignatureForTxHash_AgreesWithPermitSignatureFor(t *testing.T) {
	rFull, sFull := rsBytes()

	const digest = "0xc8e7a34af766c4ba9dc9b3d49939806fbf41fa01250c5a26afa5659e87b2020b"

	details := requests.SignatureDetails{
		R: hex.EncodeToString(rFull),
		S: hex.EncodeToString(sFull[:31]) + "zz", // 64 chars, decodes to 31 bytes
		V: "01",
	}
	signatures := map[string]requests.SignatureDetails{digest: details}

	permitSig, permitErr := permitSignatureFor(digest, signatures)
	txSig, txErr := getSignatureForTxHash(digest, signatures)

	// permitSignatureFor rejects the malformed hex...
	require.Error(t, permitErr)
	require.Nil(t, permitSig)

	// ...so getSignatureForTxHash must reject it too instead of assembling garbage.
	assert.Error(t, txErr, "the two assemblers must agree on which signature details are usable")
	if txErr == nil {
		assert.Equal(t,
			hex.EncodeToString(wantSignature(rFull, sFull[:31], 1)),
			hex.EncodeToString(txSig),
			"r||s halves must be laid out identically by both assemblers")
	}
}

// ---------------------------------------------------------------------------
// Finding 2: silent permit-signature drop on route rebuild
//
// carryOverPermitSignature only moves the signature across when the two permit digests
// match. permit2.Resolver.Resolve() regenerates the deadline (now+30m) and re-reads the
// Permit2 nonce on every buildPath, so a rebuilt route always carries a different digest.
// The already-collected signature is then dropped with no error and no signal: the send
// proceeds with PermitDetails.Signature empty.
//
// Desired behaviour: either the signature survives the rebuild, or the caller is told the
// permit has to be re-signed. carryOverPermitSignature returns nothing, so it cannot do
// the latter - the assertion below is on the only observable outcome. A fix that instead
// returns an explicit error will change this function's signature and this test with it.
// ---------------------------------------------------------------------------

func permitPathForRebuild(nonce, deadline int64, signature []byte) *routes.Path {
	return &routes.Path{
		RouterInputParamsUuid: "uuid-1",
		ProcessorName:         pathProcessorCommon.ProcessorLiFiName,
		FromChain:             &params.Network{ChainID: 1},
		PermitDetails: &permit2.Details{
			Type:      permit2.TypePermit2,
			ChainID:   1,
			Owner:     common.HexToAddress("0xAbC0000000000000000000000000000000000001"),
			Token:     common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			Amount:    big.NewInt(1000000),
			Spender:   common.HexToAddress("0x1231DEB6f5749EF6cE6943a275A1D3E7486F4EaE"),
			Permit2:   common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3"),
			Nonce:     big.NewInt(nonce),
			Deadline:  big.NewInt(deadline),
			Signature: signature,
		},
	}
}

func TestCarryOverPermitSignature_RebuiltRouteKeepsSignature(t *testing.T) {
	signature := make([]byte, crypto2.SignatureLength)
	for i := range signature {
		signature[i] = byte(i + 1)
	}

	// Same swap, rebuilt: Resolve() handed back a fresh nonce and a fresh deadline.
	oldPath := permitPathForRebuild(7, 1700000000, signature)
	newPath := permitPathForRebuild(8, 1700001800, nil)

	oldDigest, err := oldPath.PermitDetails.Digest()
	require.NoError(t, err)
	newDigest, err := newPath.PermitDetails.Digest()
	require.NoError(t, err)
	// Premise of the finding: a rebuild always changes the digest.
	require.NotEqual(t, oldDigest, newDigest)

	carryOverPermitSignature(oldPath, newPath)

	assert.NotEmpty(t, newPath.PermitDetails.Signature,
		"the collected permit signature must not be silently dropped when the route is rebuilt")
}

// TestGetOrInitDetailsForPath_RebuiltRouteKeepsSignature exercises the same drop through
// the smallest reachable wrapper: the second signing phase re-evaluates the swap path and
// getOrInitDetailsForPath swaps the stored path for the rebuilt one.
func TestGetOrInitDetailsForPath_RebuiltRouteKeepsSignature(t *testing.T) {
	signature := make([]byte, crypto2.SignatureLength)
	for i := range signature {
		signature[i] = byte(i + 1)
	}

	tm := &TransactionManager{}

	oldPath := permitPathForRebuild(7, 1700000000, signature)
	require.Same(t, oldPath, tm.getOrInitDetailsForPath(oldPath).RouterPath)

	// Route re-evaluated: same path identity, freshly resolved permit.
	newPath := permitPathForRebuild(8, 1700001800, nil)
	details := tm.getOrInitDetailsForPath(newPath)

	require.NotNil(t, details.RouterPath.PermitDetails)
	assert.NotEmpty(t, details.RouterPath.PermitDetails.Signature,
		"re-evaluating the path must not lose the signature the user already produced")
}
