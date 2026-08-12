package permit2

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/internal/rpc"
)

const (
	signatureLength = 65

	// deadlineWindow is how long a signed permit stays usable. Matches LI.FI's reference
	// integration: long enough to survive a slow confirmation, short enough that an
	// abandoned signature stops being spendable quickly.
	deadlineWindow = 30 * time.Minute
)

var ErrInvalidSignatureLength = errors.New("permit signature must be 65 bytes")

// standardPermitTypehash is EIP-2612's Permit struct. Tokens reporting anything else are
// not usable here, DAI's holder/expiry/allowed variant being the usual case.
var standardPermitTypehash = crypto.Keccak256Hash([]byte(
	"Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"))

// domainVersionCandidates are tried when a token has no version() method. Both are
// checked against the token's own DOMAIN_SEPARATOR, so a wrong guess is detected rather
// than turned into an unusable signature.
var domainVersionCandidates = []string{"1", "2"}

// Plan is the outcome of deciding how a swap acquires the tokens it spends.
type Plan struct {
	// Details is nil when the swap must fall back to the regular approve-then-swap flow.
	Details *Details
	// ApprovalSpender is the contract the user still has to approve on-chain for this plan
	// to work, or the zero address when no approval is needed. Only ever the Permit2
	// singleton, a one-time cost per token.
	ApprovalSpender common.Address
}

// NeedsApproval reports whether the plan still requires an on-chain approval.
func (p *Plan) NeedsApproval() bool {
	return p != nil && p.ApprovalSpender != (common.Address{})
}

// ResolveParams describes the swap a permit is being resolved for.
type ResolveParams struct {
	ChainID      uint64
	Owner        common.Address
	Token        common.Address
	Amount       *big.Int
	Permit2      common.Address
	Permit2Proxy common.Address
}

// Resolver decides which permit variant a swap can use, probing the token and the
// user's existing allowances on-chain.
type Resolver struct {
	ethClientGetter rpc.EthClientGetter
	now             func() time.Time
}

func NewResolver(ethClientGetter rpc.EthClientGetter) *Resolver {
	return &Resolver{ethClientGetter: ethClientGetter, now: time.Now}
}

// Resolve picks the cheapest permit variant available for this swap:
//
//  1. the token implements EIP-2612          -> no approval, ever
//  2. Permit2 is already approved for it     -> no approval transaction
//  3. otherwise                              -> approve Permit2 once, then as (2)
//
// A nil Plan (with a nil error) means no permit variant applies and the caller should
// keep the existing approve-then-swap behaviour.
func (r *Resolver) Resolve(ctx context.Context, params ResolveParams) (*Plan, error) {
	if params.Permit2 == (common.Address{}) || params.Permit2Proxy == (common.Address{}) {
		return nil, nil
	}
	if params.Amount == nil || params.Amount.Sign() <= 0 {
		return nil, nil
	}

	deadline := big.NewInt(r.now().Add(deadlineWindow).Unix())

	details, err := r.resolveEIP2612(ctx, params, deadline)
	if err != nil {
		return nil, err
	}
	if details != nil {
		return &Plan{Details: details}, nil
	}

	nonce, err := r.nextPermit2Nonce(ctx, params)
	if err != nil {
		return nil, err
	}

	allowance, err := r.allowance(ctx, params.ChainID, params.Token, params.Owner, params.Permit2)
	if err != nil {
		return nil, err
	}

	plan := &Plan{
		Details: &Details{
			Type:     TypePermit2,
			ChainID:  params.ChainID,
			Owner:    params.Owner,
			Token:    params.Token,
			Amount:   new(big.Int).Set(params.Amount),
			Spender:  params.Permit2Proxy,
			Permit2:  params.Permit2,
			Nonce:    nonce,
			Deadline: deadline,
		},
	}
	if allowance.Cmp(params.Amount) < 0 {
		plan.ApprovalSpender = params.Permit2
	}
	return plan, nil
}

// resolveEIP2612 returns permit details when the token implements EIP-2612 in the form the
// proxy can drive, nil when it does not. Detection is positive-only: anything ambiguous
// falls through to the Permit2 path rather than producing a signature that would revert.
func (r *Resolver) resolveEIP2612(ctx context.Context, params ResolveParams, deadline *big.Int) (*Details, error) {
	onChainSeparator, err := r.domainSeparator(ctx, params.ChainID, params.Token)
	if err != nil || onChainSeparator == (common.Hash{}) {
		// No DOMAIN_SEPARATOR means no EIP-2612.
		return nil, nil //nolint:nilerr // absence is a valid answer, not a failure
	}

	if r.usesNonStandardPermit(ctx, params.ChainID, params.Token) {
		return nil, nil
	}

	name, err := r.stringCall(ctx, params.ChainID, params.Token, "name")
	if err != nil || name == "" {
		return nil, nil //nolint:nilerr
	}

	versions := domainVersionCandidates
	if version, err := r.stringCall(ctx, params.ChainID, params.Token, "version"); err == nil && version != "" {
		versions = append([]string{version}, versions...)
	}

	details := &Details{
		Type:      TypeEIP2612,
		ChainID:   params.ChainID,
		Owner:     params.Owner,
		Token:     params.Token,
		Amount:    new(big.Int).Set(params.Amount),
		Spender:   params.Permit2Proxy,
		Deadline:  deadline,
		TokenName: name,
	}

	// Verify the domain we would sign against actually matches the one the token
	// enforces. This turns version guessing from a heuristic into a check.
	matched := false
	for _, version := range versions {
		details.TokenVersion = version
		separator, err := domainSeparatorFor(details)
		if err != nil {
			return nil, err
		}
		if separator == onChainSeparator {
			matched = true
			break
		}
	}
	if !matched {
		return nil, nil
	}

	nonce, err := r.uint256Call(ctx, params.ChainID, params.Token, parsedERC20PermitABI, "nonces", params.Owner)
	if err != nil {
		return nil, nil //nolint:nilerr // a token without nonces() is not EIP-2612
	}
	details.Nonce = nonce

	return details, nil
}

// usesNonStandardPermit reports whether the token exposes a permit typehash other than
// EIP-2612's. DAI's is the common one: it takes an allowance flag instead of a value and
// so can't be driven through the proxy's ERC20Permit call.
//
// Tokens that don't expose PERMIT_TYPEHASH at all (OpenZeppelin keeps it private) read as
// standard here and are still guarded by the domain-separator check that follows.
func (r *Resolver) usesNonStandardPermit(ctx context.Context, chainID uint64, token common.Address) bool {
	result, err := r.call(ctx, chainID, token, parsedERC20PermitABI, "PERMIT_TYPEHASH")
	if err != nil || len(result) != 32 {
		return false
	}
	return common.BytesToHash(result) != standardPermitTypehash
}

func (r *Resolver) domainSeparator(ctx context.Context, chainID uint64, token common.Address) (common.Hash, error) {
	result, err := r.call(ctx, chainID, token, parsedERC20PermitABI, "DOMAIN_SEPARATOR")
	if err != nil || len(result) != 32 {
		return common.Hash{}, err
	}
	return common.BytesToHash(result), nil
}

// nextPermit2Nonce reads the next unused unordered nonce for the owner. Permit2 tracks
// nonces as a bitmap, so this is a lookup rather than a counter.
func (r *Resolver) nextPermit2Nonce(ctx context.Context, params ResolveParams) (*big.Int, error) {
	return r.uint256Call(ctx, params.ChainID, params.Permit2Proxy, parsedPermit2ProxyABI, "nextNonce", params.Owner)
}

func (r *Resolver) allowance(ctx context.Context, chainID uint64, token, owner, spender common.Address) (*big.Int, error) {
	return r.uint256Call(ctx, chainID, token, parsedERC20AllowanceABI, "allowance", owner, spender)
}

func (r *Resolver) call(ctx context.Context, chainID uint64, to common.Address, parsed abi.ABI, method string, args ...interface{}) ([]byte, error) {
	input, err := parsed.Pack(method, args...)
	if err != nil {
		return nil, err
	}

	ethClient, err := r.ethClientGetter.EthClient(chainID)
	if err != nil {
		return nil, err
	}

	return ethClient.CallContract(ctx, ethereum.CallMsg{To: &to, Data: input}, nil)
}

func (r *Resolver) uint256Call(ctx context.Context, chainID uint64, to common.Address, parsed abi.ABI, method string, args ...interface{}) (*big.Int, error) {
	result, err := r.call(ctx, chainID, to, parsed, method, args...)
	if err != nil {
		return nil, err
	}
	values, err := parsed.Unpack(method, result)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, errors.New("permit2: empty response for " + method)
	}
	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, errors.New("permit2: unexpected response type for " + method)
	}
	return value, nil
}

func (r *Resolver) stringCall(ctx context.Context, chainID uint64, to common.Address, method string) (string, error) {
	result, err := r.call(ctx, chainID, to, parsedERC20PermitABI, method)
	if err != nil {
		return "", err
	}
	values, err := parsedERC20PermitABI.Unpack(method, result)
	if err != nil {
		return "", err
	}
	if len(values) == 0 {
		return "", errors.New("permit2: empty response for " + method)
	}
	value, ok := values[0].(string)
	if !ok {
		return "", errors.New("permit2: unexpected response type for " + method)
	}
	return value, nil
}
