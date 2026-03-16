package connector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	state := setupTests(t)

	assert.NotNil(t, state.service)
}

func TestService_Start(t *testing.T) {
	state := setupTests(t)
	state.service.config.WSEnabled = true
	state.service.config.WSHost = "127.0.0.1"
	state.service.config.WSPort = 0

	err := state.service.Start()
	assert.NoError(t, err)
	assert.NotNil(t, state.service.wsServer)
	t.Cleanup(func() {
		assert.NoError(t, state.service.Stop())
	})
}

func TestService_Start_WSDisabled_DoesNotCreateWSServer(t *testing.T) {
	state := setupTests(t)
	state.service.config.WSEnabled = false

	err := state.service.Start()
	assert.NoError(t, err)
	assert.Nil(t, state.service.wsServer)
}

func TestService_Stop(t *testing.T) {
	state := setupTests(t)

	err := state.service.Stop()
	assert.NoError(t, err)
}

func TestService_APIs(t *testing.T) {
	state := setupTests(t)

	apis := state.api.s.APIs()

	assert.Len(t, apis, 1)
	assert.Equal(t, "connector", apis[0].Namespace)
	assert.Equal(t, "0.1.0", apis[0].Version)
	assert.NotNil(t, apis[0].Service)
}
