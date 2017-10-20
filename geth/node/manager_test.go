package node

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/build/_workspace/deps/src/github.com/pkg/errors"
	"github.com/status-im/status-go/geth/common/geth"
	"github.com/status-im/status-go/geth/params"
	"reflect"
)

func TestStartNodeSuccess(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	m.newNode.EXPECT().Start().Times(1).Return(nil)
	m.newNode.EXPECT().GetNode().AnyTimes().Return(nil)
	m.managerNode.EXPECT().SetNode(m.newNode.GetNode()).Times(1)

	m.constr.EXPECT().Make().Return(m.newNode, nil)
	m.constr.EXPECT().SetConfig(m.config).AnyTimes()
	m.constr.EXPECT().Config().AnyTimes().Return(m.config)

	m.logger.EXPECT().Init(m.config.LogFile, m.config.LogLevel).Times(1).Return()
	m.rpc.EXPECT().Init(gomock.Any(), gomock.Any()).Times(1).Return(nil)

	err := m.manager.StartNode(m.constr)

	//todo: use testify checks
	if err != nil {
		t.Fatal(err)
	}
}

func TestStartNodeNewNodeError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	expectedErr := errors.New("error")
	m.constr.EXPECT().Make().Return(nil, expectedErr)
	m.constr.EXPECT().SetConfig(m.config).AnyTimes()
	m.constr.EXPECT().Config().AnyTimes().Return(m.config)
	m.logger.EXPECT().Init(m.config.LogFile, m.config.LogLevel).Times(1).Return()

	err := m.manager.StartNode(m.constr)

	//todo: use testify checks
	if !reflect.DeepEqual(err, expectedErr) {
		t.Fatal(err, expectedErr)
	}
}

func testConfig() *params.NodeConfig {
	return &params.NodeConfig{
		LogLevel: "ERROR",
		LogFile:  "test.txt",
		BootClusterConfig: &params.BootClusterConfig{
			Enabled:   true,
			BootNodes: []string{},
		},
	}
}

type mock struct {
	ctrl        *gomock.Controller
	config      *params.NodeConfig
	manager     *NodeManager
	constr      *geth.MockNodeConstructor
	logger      *Mocklogger
	rpc         *MockrpcAccess
	newNode     *geth.MockNode
	managerNode *geth.MockNode
}

func initMocks(t *testing.T) *mock {
	ctrl := gomock.NewController(t)
	config := testConfig()
	constr := geth.NewMockNodeConstructor(ctrl)
	logger := NewMocklogger(ctrl)
	rpc := NewMockrpcAccess(ctrl)
	newNode := geth.NewMockNode(ctrl)
	managerNode := geth.NewMockNode(ctrl)

	manager := NewNodeManager()
	manager.logger = logger
	manager.rpc = rpc
	manager.node = managerNode

	return &mock{
		ctrl:        ctrl,
		config:      config,
		manager:     manager,
		constr:      constr,
		logger:      logger,
		rpc:         rpc,
		newNode:     newNode,
		managerNode: managerNode,
	}
}
