package api

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"

	"github.com/status-im/status-go/geth/node"
	"github.com/status-im/status-go/geth/params"
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
		assert.NotNil(t, backend.JailManager())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.PersonalAPI())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.Transactor())
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		assert.NotNil(t, backend.PendingSignRequests())
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
			assert.NoError(t, backend.ReSelectAccount())
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
	connections := [...]string{"wifi", "cellular"}
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

// TODO(adam): add concurrent tests for: SendTransaction, ApproveSignRequest, DiscardSignRequest
