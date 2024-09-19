package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

	assert.NotNil(t, state.service)
	assert.Equal(t, state.rpcClient.GetNetworkManager(), state.service.nm)
}

func TestService_Start(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

	err := state.service.Start()
	assert.NoError(t, err)
}

func TestService_Stop(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

	err := state.service.Stop()
	assert.NoError(t, err)
}

func TestService_APIs(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

	apis := state.api.s.APIs()

	assert.Len(t, apis, 1)
	assert.Equal(t, "connector", apis[0].Namespace)
	assert.Equal(t, "0.1.0", apis[0].Version)
	assert.NotNil(t, apis[0].Service)
}

func TestService_Protocols(t *testing.T) {
	state, closeFn := setupTests(t)
	t.Cleanup(closeFn)

	protocols := state.service.Protocols()
	assert.Nil(t, protocols)
}
