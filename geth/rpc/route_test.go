package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockGethrpcClient struct {
	mock.Mock
}

func (m *MockGethrpcClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	a := m.Called(ctx, result, method, args)
	return a.Error(0)
}

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) Handler(c context.Context, args ...interface{}) (interface{}, error) {
	a := m.Called(c, args)
	return nil, a.Error(0)
}

// localMethods are methods that should be executed locally.
var localNodeTestMethods = []string{"some_weirdo_method", "shh_newMessageFilter"}
var localHandlerTestMethods = []string{"eth_accounts"}

func TestRouteWithUpstream(t *testing.T) {
	// Arrange.

	for method, destination := range RoutingTable {
		switch destination {
		case localHandler:
			mockLocalNode := MockGethrpcClient{}
			mockUpstreamNode := MockGethrpcClient{}

			r := newRouter(&mockLocalNode, &mockUpstreamNode, true /*upstream enabled*/)

			mockHandler := MockHandler{}
			r.registerHandler(method, mockHandler.Handler)
			mockHandler.On("Handler", context.TODO(), []interface{}{0}).Return(nil).Once()

			// Act.
			err := r.callContext(context.TODO(), nil, method, 0)

			// Assert.
			require.True(t, err == nil, "callContext failed with method "+method)
			require.True(t, mockHandler.AssertCalled(t, "Handler", context.TODO(), []interface{}{0}))
			require.True(t, mockUpstreamNode.AssertNotCalled(t, "CallContext", context.TODO(), []interface{}{0}))
			require.True(t, mockLocalNode.AssertNotCalled(t, "CallContext", context.TODO(), []interface{}{0}))

		case localNode:
			mockLocalNode := MockGethrpcClient{}
			mockUpstreamNode := MockGethrpcClient{}

			r := newRouter(&mockLocalNode, &mockUpstreamNode, true /*upstream enabled*/)

			mockHandler := MockHandler{}
			r.registerHandler(method, mockHandler.Handler)

			mockLocalNode.On("CallContext", context.TODO(), nil, method, []interface{}{0}).Return(nil).Once()

			// Act.
			err := r.callContext(context.TODO(), nil, method, 0)

			// Assert.
			require.True(t, err == nil)
			require.True(t, mockLocalNode.AssertCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
			require.True(t, mockUpstreamNode.AssertNotCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
			require.True(t, mockHandler.AssertNotCalled(t, "Handler", context.TODO(), []interface{}{0}), "mock handler should not be called for method "+method)

		case upstreamNode:
			mockLocalNode := MockGethrpcClient{}
			mockUpstreamNode := MockGethrpcClient{}

			r := newRouter(&mockLocalNode, &mockUpstreamNode, true /*upstream enabled*/)

			mockHandler := MockHandler{}
			r.registerHandler(method, mockHandler.Handler)

			mockUpstreamNode.On("CallContext", context.TODO(), nil, method, []interface{}{0}).Return(nil).Once()

			// Act.
			err := r.callContext(context.TODO(), nil, method, 0)

			// Assert.
			require.True(t, err == nil)
			require.True(t, mockLocalNode.AssertNotCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
			require.True(t, mockUpstreamNode.AssertCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
			require.True(t, mockHandler.AssertNotCalled(t, "Handler", context.TODO(), []interface{}{0}), "mock handler should not be called for method "+method)
		}
	}

	for _, method := range localNodeTestMethods {
		mockLocalNode := MockGethrpcClient{}
		mockUpstreamNode := MockGethrpcClient{}

		r := newRouter(&mockLocalNode, &mockUpstreamNode, true /*upstream enabled*/)

		mockLocalNode.On("CallContext", context.TODO(), nil, method, []interface{}{0}).Return(nil).Once()

		// Act.
		err := r.callContext(context.TODO(), nil, method, 0)

		// Assert.
		require.True(t, err == nil)
		require.True(t, mockLocalNode.AssertCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
		require.True(t, mockUpstreamNode.AssertNotCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
	}

	for _, method := range localHandlerTestMethods {
		mockLocalNode := MockGethrpcClient{}
		mockUpstreamNode := MockGethrpcClient{}

		r := newRouter(&mockLocalNode, &mockUpstreamNode, true /*upstream enabled*/)

		mockHandler := MockHandler{}
		r.registerHandler(method, mockHandler.Handler)
		mockHandler.On("Handler", context.TODO(), []interface{}{0}).Return(nil).Once()

		// Act.
		err := r.callContext(context.TODO(), nil, method, 0)

		// Assert.
		require.True(t, err == nil)
		require.True(t, mockLocalNode.AssertNotCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
		require.True(t, mockUpstreamNode.AssertNotCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
		require.True(t, mockHandler.AssertCalled(t, "Handler", context.TODO(), []interface{}{0}))
	}
}

func TestRouteWithoutUpstream(t *testing.T) {
	// Arrange.
	mockLocalNode := MockGethrpcClient{}

	mockUpstreamNode := MockGethrpcClient{}

	r := newRouter(&mockLocalNode, &mockUpstreamNode, false /*upstream enabled*/)

	mockHandler := MockHandler{}

	for method, destination := range RoutingTable {
		switch destination {
		case localHandler:
			r.registerHandler(method, mockHandler.Handler)
			mockHandler.On("Handler", context.TODO(), []interface{}{0}).Return(nil).Once()

			// Act.
			err := r.callContext(context.TODO(), nil, method, 0)

			// Assert.
			require.True(t, err == nil, "callContext failed with method "+method)
			require.True(t, mockHandler.AssertCalled(t, "Handler", context.TODO(), []interface{}{0}))
			require.True(t, mockUpstreamNode.AssertNotCalled(t, "CallContext", context.TODO(), []interface{}{0}))
			require.True(t, mockLocalNode.AssertNotCalled(t, "CallContext", context.TODO(), []interface{}{0}))

		case localNode:
			mockLocalNode.On("CallContext", context.TODO(), nil, method, []interface{}{0}).Return(nil).Once()

			// Act.
			err := r.callContext(context.TODO(), nil, method, 0)

			// Assert.
			require.True(t, err == nil)
			require.True(t, mockLocalNode.AssertCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
			require.True(t, mockUpstreamNode.AssertNotCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))

		case upstreamNode:
			mockLocalNode.On("CallContext", context.TODO(), nil, method, []interface{}{0}).Return(nil).Once()

			// Act.
			err := r.callContext(context.TODO(), nil, method, 0)

			// Assert.
			require.True(t, err == nil)
			require.True(t, mockLocalNode.AssertCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
			require.True(t, mockUpstreamNode.AssertNotCalled(t, "CallContext", context.TODO(), nil, method, []interface{}{0}))
		}
	}
}
