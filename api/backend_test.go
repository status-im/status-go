package api

import (
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/status-im/status-go/node"
	"github.com/status-im/status-go/params"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendStartNodeConcurrently(t *testing.T) {
	backend := NewStatusBackend()
	config := params.NodeConfig{}
	count := 2
	resultCh := make(chan error)

	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func() {
			resultCh <- backend.StartNode(&config)
			wg.Done()
		}()
	}

	// close channel as otherwise for loop never finishes
	go func() { wg.Wait(); close(resultCh) }()

	var results []error
	for err := range resultCh {
		results = append(results, err)
	}

	require.Contains(t, results, nil)
	require.Contains(t, results, node.ErrNodeRunning)

	err := backend.StopNode()
	require.NoError(t, err)
}

func TestBackendRestartNodeConcurrently(t *testing.T) {
	backend := NewStatusBackend()
	config := params.NodeConfig{}
	count := 3

	err := backend.StartNode(&config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func(idx int) {
			assert.NoError(t, backend.RestartNode())
			wg.Done()
		}(i)
	}

	wg.Wait()
}

// TODO(adam): add concurrent tests for ResetChainData()

func TestBackendGettersConcurrently(t *testing.T) {
	backend := NewStatusBackend()
	config := params.NodeConfig{}

	err := backend.StartNode(&config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.StatusNode())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.AccountManager())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.personalAPI)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.Transactor())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.True(t, backend.IsNodeRunning())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.True(t, backend.IsNodeRunning())
		wg.Done()
	}()

	wg.Wait()
}

func TestBackendAccountsConcurrently(t *testing.T) {
	backend := NewStatusBackend()
	config := params.NodeConfig{}

	err := backend.StartNode(&config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var wgCreateAccounts sync.WaitGroup
	count := 3
	addressCh := make(chan [2]string, count) // use buffered channel to avoid blocking

	// create new accounts concurrently
	for i := 0; i < count; i++ {
		wgCreateAccounts.Add(1)
		go func(pass string) {
			address, _, _, err := backend.AccountManager().CreateAccount(pass)
			assert.NoError(t, err)
			addressCh <- [...]string{address, pass}
			wgCreateAccounts.Done()
		}("password-00" + string(i))
	}

	// close addressCh as otherwise for loop never finishes
	go func() { wgCreateAccounts.Wait(); close(addressCh) }()

	// select, reselect or logout concurrently
	var wg sync.WaitGroup

	for tuple := range addressCh {
		wg.Add(1)
		go func(tuple [2]string) {
			assert.NoError(t, backend.SelectAccount(tuple[0], tuple[1]))
			wg.Done()
		}(tuple)

		wg.Add(1)
		go func() {
			assert.NoError(t, backend.reSelectAccount())
			wg.Done()
		}()

		wg.Add(1)
		go func() {
			assert.NoError(t, backend.Logout())
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestBackendConnectionChangesConcurrently(t *testing.T) {
	connections := [...]string{wifi, cellular, unknown}
	backend := NewStatusBackend()
	count := 3

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			connIdx := rand.Intn(len(connections))
			backend.ConnectionChange(connections[connIdx], false)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestBackendConnectionChangesToOffline(t *testing.T) {
	b := NewStatusBackend()
	b.ConnectionChange(none, false)
	assert.True(t, b.connectionState.Offline)

	b.ConnectionChange(wifi, false)
	assert.False(t, b.connectionState.Offline)

	b.ConnectionChange("unknown-state", false)
	assert.False(t, b.connectionState.Offline)
}

func TestBackendCallRPCConcurrently(t *testing.T) {
	backend := NewStatusBackend()
	config := params.NodeConfig{}
	count := 3

	err := backend.StartNode(&config)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, backend.StopNode())
	}()

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			result := backend.CallRPC(fmt.Sprintf(
				`{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":%d}`,
				idx+1,
			))
			assert.NotContains(t, result, "error")
			wg.Done()
		}(i)

		wg.Add(1)
		go func(idx int) {
			result := backend.CallPrivateRPC(fmt.Sprintf(
				`{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":%d}`,
				idx+1,
			))
			assert.NotContains(t, result, "error")
			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestAppStateChange(t *testing.T) {
	backend := NewStatusBackend()

	var testCases = []struct {
		name          string
		fromState     appState
		toState       appState
		expectedState appState
	}{
		{
			name:          "success",
			fromState:     appStateInactive,
			toState:       appStateBackground,
			expectedState: appStateBackground,
		},
		{
			name:          "invalid state",
			fromState:     appStateInactive,
			toState:       "unexisting",
			expectedState: appStateInactive,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			backend.appState = tc.fromState
			backend.AppStateChange(tc.toState.String())
			assert.Equal(t, tc.expectedState.String(), backend.appState.String())
		})
	}
}

func TestPrepareTxArgs(t *testing.T) {
	var flagtests = []struct {
		description      string
		gas              int64
		gasPrice         int64
		expectedGas      *hexutil.Uint64
		expectedGasPrice *hexutil.Big
	}{
		{
			description:      "Empty gas and gas price",
			gas:              0,
			gasPrice:         0,
			expectedGas:      nil,
			expectedGasPrice: nil,
		},
		{
			description: "Non empty gas and gas price",
			gas:         1,
			gasPrice:    2,
			expectedGas: func() *hexutil.Uint64 {
				x := hexutil.Uint64(1)
				return &x
			}(),
			expectedGasPrice: (*hexutil.Big)(big.NewInt(2)),
		},
		{
			description: "Empty gas price",
			gas:         1,
			gasPrice:    0,
			expectedGas: func() *hexutil.Uint64 {
				x := hexutil.Uint64(1)
				return &x
			}(),
			expectedGasPrice: nil,
		},
		{
			description:      "Empty gas",
			gas:              0,
			gasPrice:         2,
			expectedGas:      nil,
			expectedGasPrice: (*hexutil.Big)(big.NewInt(2)),
		},
	}
	for _, tt := range flagtests {
		t.Run(tt.description, func(t *testing.T) {
			args := prepareTxArgs(tt.gas, tt.gasPrice)
			assert.Equal(t, tt.expectedGas, args.Gas)
			assert.Equal(t, tt.expectedGasPrice, args.GasPrice)
		})
	}
}

// TODO(adam): add concurrent tests for: SendTransaction
