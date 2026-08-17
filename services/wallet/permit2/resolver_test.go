package permit2

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	gomock "go.uber.org/mock/gomock"

	mock_ethclient "github.com/status-im/status-go/internal/rpc/chain/ethclient/mock/client/ethclient"
	mock_rpcclient "github.com/status-im/status-go/internal/rpc/mock/client"
	walletCommon "github.com/status-im/status-go/services/wallet/common"
)

const testChainID = walletCommon.EthereumMainnet

var (
	testNow      = time.Unix(1700000000, 0)
	testDeadline = big.NewInt(testNow.Add(deadlineWindow).Unix())
)

// callKey identifies a call as "<contract>.<4-byte selector>", which is all the resolver's
// call sites differ by.
func callKey(to common.Address, data []byte) string {
	return to.Hex() + "." + common.Bytes2Hex(data[:4])
}

// newResolver wires a resolver to a client answering eth_call from the given table. Calls
// the table doesn't cover revert, the way an absent method does on-chain.
func newResolver(t *testing.T, responses map[string][]byte) *Resolver {
	t.Helper()

	ctrl := gomock.NewController(t)
	client := mock_ethclient.NewMockEthClientInterface(ctrl)
	client.EXPECT().CallContract(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, call ethereum.CallMsg, _ *big.Int) ([]byte, error) {
			if response, ok := responses[callKey(*call.To, call.Data)]; ok {
				return response, nil
			}
			return nil, errors.New("execution reverted")
		}).AnyTimes()

	getter := mock_rpcclient.NewMockEthClientGetter(ctrl)
	getter.EXPECT().EthClient(testChainID).Return(client, nil).AnyTimes()

	resolver := NewResolver(getter)
	resolver.now = func() time.Time { return testNow }
	return resolver
}

func selectorKey(t *testing.T, to common.Address, parsed abi.ABI, method string, args ...interface{}) string {
	t.Helper()
	input, err := parsed.Pack(method, args...)
	require.NoError(t, err)
	return callKey(to, input)
}

func packOutput(t *testing.T, parsed abi.ABI, method string, value interface{}) []byte {
	t.Helper()
	packed, err := parsed.Methods[method].Outputs.Pack(value)
	require.NoError(t, err)
	return packed
}

func testParams() ResolveParams {
	return ResolveParams{
		ChainID:      testChainID,
		Owner:        testOwner,
		Token:        testToken,
		Amount:       big.NewInt(1000000),
		Permit2:      testPermit2,
		Permit2Proxy: testPermit2Proxy,
	}
}

// eip2612Responses makes the token look like a compliant EIP-2612 token whose domain uses
// the given version.
func eip2612Responses(t *testing.T, name, version string, nonce *big.Int) map[string][]byte {
	t.Helper()

	details := eip2612Details()
	details.ChainID = testChainID
	details.TokenName = name
	details.TokenVersion = version
	separator, err := domainSeparatorFor(details)
	require.NoError(t, err)

	responses := map[string][]byte{
		selectorKey(t, testToken, parsedERC20PermitABI, "DOMAIN_SEPARATOR"): separator.Bytes(),
		selectorKey(t, testToken, parsedERC20PermitABI, "name"): packOutput(t,
			parsedERC20PermitABI, "name", name),
		selectorKey(t, testToken, parsedERC20PermitABI, "nonces", testOwner): packOutput(t,
			parsedERC20PermitABI, "nonces", nonce),
	}
	if version != "" {
		responses[selectorKey(t, testToken, parsedERC20PermitABI, "version")] = packOutput(t,
			parsedERC20PermitABI, "version", version)
	}
	return responses
}

// permit2Responses makes the token look like a plain ERC-20: no EIP-2612, a Permit2 nonce
// from the proxy and the given allowance towards the Permit2 singleton.
func permit2Responses(t *testing.T, nextNonce, allowance *big.Int) map[string][]byte {
	t.Helper()
	return map[string][]byte{
		selectorKey(t, testPermit2Proxy, parsedPermit2ProxyABI, "nextNonce", testOwner): packOutput(t,
			parsedPermit2ProxyABI, "nextNonce", nextNonce),
		selectorKey(t, testToken, parsedERC20AllowanceABI, "allowance", testOwner, testPermit2): packOutput(t,
			parsedERC20AllowanceABI, "allowance", allowance),
	}
}

func TestResolve_NoPermitWhenAddressesOrAmountMissing(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ResolveParams)
	}{
		{"no permit2", func(p *ResolveParams) { p.Permit2 = common.Address{} }},
		{"no proxy", func(p *ResolveParams) { p.Permit2Proxy = common.Address{} }},
		{"nil amount", func(p *ResolveParams) { p.Amount = nil }},
		{"zero amount", func(p *ResolveParams) { p.Amount = big.NewInt(0) }},
		{"negative amount", func(p *ResolveParams) { p.Amount = big.NewInt(-1) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := testParams()
			tt.mutate(&params)

			// A nil client getter: an unusable request must not reach the chain.
			resolver := NewResolver(nil)
			plan, err := resolver.Resolve(context.Background(), params)

			require.NoError(t, err)
			require.Nil(t, plan)
		})
	}
}

func TestResolve_EIP2612TokenNeedsNoApproval(t *testing.T) {
	resolver := newResolver(t, eip2612Responses(t, "USD Coin", "2", big.NewInt(7)))

	plan, err := resolver.Resolve(context.Background(), testParams())

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.NotNil(t, plan.Details)
	require.Equal(t, TypeEIP2612, plan.Details.Type)
	require.Equal(t, "USD Coin", plan.Details.TokenName)
	require.Equal(t, "2", plan.Details.TokenVersion)
	require.Equal(t, testPermit2Proxy, plan.Details.Spender, "the proxy is the one calling permit()")
	require.Equal(t, big.NewInt(7), plan.Details.Nonce)
	require.Equal(t, testDeadline, plan.Details.Deadline)
	require.False(t, plan.NeedsApproval())
	require.Equal(t, common.Address{}, plan.ApprovalSpender)
}

// A token with no version() still has a domain; the resolver has to find which of the
// candidate versions the token actually uses instead of guessing one.
func TestResolve_EIP2612WithoutVersionMethod(t *testing.T) {
	responses := eip2612Responses(t, "Dai Stablecoin", "1", big.NewInt(0))
	delete(responses, selectorKey(t, testToken, parsedERC20PermitABI, "version"))

	plan, err := newResolver(t, responses).Resolve(context.Background(), testParams())

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Equal(t, TypeEIP2612, plan.Details.Type)
	require.Equal(t, "1", plan.Details.TokenVersion)
}

// The domain the token reports has to be the one we would sign against. A mismatch means
// version guessing failed, and signing anyway would produce a permit that reverts.
func TestResolve_EIP2612DomainMismatchFallsBackToPermit2(t *testing.T) {
	responses := eip2612Responses(t, "Some Token", "3", big.NewInt(0))
	// The candidates are "3" (reported), "1" and "2"; none of them can match this.
	responses[selectorKey(t, testToken, parsedERC20PermitABI, "DOMAIN_SEPARATOR")] = common.HexToHash("0xdead").Bytes()
	for key, value := range permit2Responses(t, big.NewInt(11), big.NewInt(0)) {
		responses[key] = value
	}

	plan, err := newResolver(t, responses).Resolve(context.Background(), testParams())

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Equal(t, TypePermit2, plan.Details.Type)
}

// DAI-style permit() takes an allowance flag rather than a value, so it can't be driven
// through the proxy even though the token has a domain separator.
func TestResolve_NonStandardPermitTypehashFallsBackToPermit2(t *testing.T) {
	responses := eip2612Responses(t, "Dai Stablecoin", "1", big.NewInt(0))
	responses[selectorKey(t, testToken, parsedERC20PermitABI, "PERMIT_TYPEHASH")] =
		common.HexToHash("0xea2aa0a1be11a07ed86d755c93467f4f82362b452371d1ba94d1715123511acb").Bytes()
	for key, value := range permit2Responses(t, big.NewInt(3), big.NewInt(0)) {
		responses[key] = value
	}

	plan, err := newResolver(t, responses).Resolve(context.Background(), testParams())

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Equal(t, TypePermit2, plan.Details.Type)
}

func TestResolve_StandardPermitTypehashKeepsEIP2612(t *testing.T) {
	responses := eip2612Responses(t, "USD Coin", "2", big.NewInt(1))
	responses[selectorKey(t, testToken, parsedERC20PermitABI, "PERMIT_TYPEHASH")] = standardPermitTypehash.Bytes()

	plan, err := newResolver(t, responses).Resolve(context.Background(), testParams())

	require.NoError(t, err)
	require.Equal(t, TypeEIP2612, plan.Details.Type)
}

func TestResolve_Permit2WithExistingAllowance(t *testing.T) {
	resolver := newResolver(t, permit2Responses(t, big.NewInt(42), big.NewInt(1000000)))

	plan, err := resolver.Resolve(context.Background(), testParams())

	require.NoError(t, err)
	require.NotNil(t, plan)
	require.Equal(t, TypePermit2, plan.Details.Type)
	require.Equal(t, testPermit2, plan.Details.Permit2, "the permit is signed against the singleton")
	require.Equal(t, testPermit2Proxy, plan.Details.Spender, "the proxy is what may pull the tokens")
	require.Equal(t, big.NewInt(42), plan.Details.Nonce)
	require.Equal(t, testDeadline, plan.Details.Deadline)
	require.Equal(t, big.NewInt(1000000), plan.Details.Amount)
	require.False(t, plan.NeedsApproval(), "an allowance covering the swap needs no approval tx")
}

func TestResolve_Permit2WithInsufficientAllowanceNeedsApproval(t *testing.T) {
	resolver := newResolver(t, permit2Responses(t, big.NewInt(1), big.NewInt(999999)))

	plan, err := resolver.Resolve(context.Background(), testParams())

	require.NoError(t, err)
	require.True(t, plan.NeedsApproval())
	require.Equal(t, testPermit2, plan.ApprovalSpender,
		"only the Permit2 singleton is ever approved, never the proxy")
}

// The caller's big.Int must not be reachable through the plan.
func TestResolve_CopiesAmount(t *testing.T) {
	resolver := newResolver(t, permit2Responses(t, big.NewInt(1), big.NewInt(0)))
	params := testParams()

	plan, err := resolver.Resolve(context.Background(), params)
	require.NoError(t, err)

	params.Amount.SetInt64(1)
	require.Equal(t, big.NewInt(1000000), plan.Details.Amount)
}

// Silently dropping to the approval flow would hide a broken RPC.
func TestResolve_PropagatesRPCFailures(t *testing.T) {
	tests := []struct {
		name    string
		missing string
	}{
		{"nextNonce unreadable", selectorKey(t, testPermit2Proxy, parsedPermit2ProxyABI, "nextNonce", testOwner)},
		{"allowance unreadable", selectorKey(t, testToken, parsedERC20AllowanceABI, "allowance", testOwner, testPermit2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses := permit2Responses(t, big.NewInt(1), big.NewInt(0))
			delete(responses, tt.missing)

			plan, err := newResolver(t, responses).Resolve(context.Background(), testParams())

			require.Error(t, err)
			require.Nil(t, plan)
		})
	}
}

func TestResolve_NoClientForChain(t *testing.T) {
	getter := mock_rpcclient.NewMockEthClientGetter(gomock.NewController(t))
	getter.EXPECT().EthClient(gomock.Any()).Return(nil, errors.New("no client")).AnyTimes()

	resolver := NewResolver(getter)
	resolver.now = func() time.Time { return testNow }

	plan, err := resolver.Resolve(context.Background(), testParams())

	require.Error(t, err)
	require.Nil(t, plan)
}

func TestPlan_NeedsApprovalNilSafe(t *testing.T) {
	var plan *Plan
	require.False(t, plan.NeedsApproval())
	require.False(t, (&Plan{}).NeedsApproval())
}
