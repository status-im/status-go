package node

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/geth/common/geth"
	"github.com/status-im/status-go/geth/params"
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

	if *m.manager.state != started {
		t.FailNow()
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

	if *m.manager.state != stopped {
		t.FailNow()
	}
}

func TestStartNodeNewNodeStartError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	expectedErr := errors.New("error")
	m.newNode.EXPECT().Start().Times(1).Return(expectedErr)
	m.newNode.EXPECT().GetNode().AnyTimes().Return(nil)

	m.constr.EXPECT().Make().Return(m.newNode, nil)
	m.constr.EXPECT().SetConfig(m.config).AnyTimes()
	m.constr.EXPECT().Config().AnyTimes().Return(m.config)
	m.logger.EXPECT().Init(m.config.LogFile, m.config.LogLevel).Times(1).Return()

	err := m.manager.StartNode(m.constr)
	if !reflect.DeepEqual(err, ErrInvalidNodeManager) {
		t.Fatal(err, expectedErr)
	}

	if *m.manager.state != stopped {
		t.FailNow()
	}
}

func TestStartNodeRPCError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	m.newNode.EXPECT().Start().Times(1).Return(nil)
	m.newNode.EXPECT().GetNode().AnyTimes().Return(nil)
	m.managerNode.EXPECT().SetNode(m.newNode.GetNode()).Times(1)

	m.constr.EXPECT().Make().Return(m.newNode, nil)
	m.constr.EXPECT().SetConfig(m.config).AnyTimes()
	m.constr.EXPECT().Config().AnyTimes().Return(m.config)

	m.logger.EXPECT().Init(m.config.LogFile, m.config.LogLevel).Times(1).Return()
	m.rpc.EXPECT().Init(gomock.Any(), gomock.Any()).Times(1).Return(errors.New("error"))

	err := m.manager.StartNode(m.constr)
	if !reflect.DeepEqual(err, ErrRPCClient) {
		t.Fatal(err)
	}

	if *m.manager.state != stopped {
		t.FailNow()
	}
}

func TestStartNodeManyTimesError(t *testing.T) {
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
	if err != nil {
		t.Fatal(err)
	}
	if *m.manager.state != started {
		t.FailNow()
	}

	err = m.manager.StartNode(m.constr)
	if !reflect.DeepEqual(err, ErrNodeExists) {
		t.Fatal(err)
	}
	if *m.manager.state != started {
		t.FailNow()
	}
}

func TestStopNodeSuccess(t *testing.T) {
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
	_ = m.manager.StartNode(m.constr)

	m.managerNode.EXPECT().Stop().Times(1).Return(nil)
	m.managerNode.EXPECT().Wait().Times(1)

	err := m.manager.StopNode()
	if err != nil {
		t.Fatal(err)
	}

	if *m.manager.state != stopped {
		t.FailNow()
	}
}

func TestStopNodeError(t *testing.T) {
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
	_ = m.manager.StartNode(m.constr)

	errExpected := errors.New("expected error")
	m.managerNode.EXPECT().Stop().Times(1).Return(errExpected)

	err := m.manager.StopNode()
	if !reflect.DeepEqual(err, errExpected) {
		t.Fatal(err)
	}

	if *m.manager.state != started {
		t.FailNow()
	}
}

func TestStopNoNodeRunningError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	err := m.manager.StopNode()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}

	if *m.manager.state != pending {
		t.FailNow()
	}
}

func TestStopNodeManyTimesError(t *testing.T) {
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
	_ = m.manager.StartNode(m.constr)

	m.managerNode.EXPECT().Stop().Times(1).Return(nil)
	m.managerNode.EXPECT().Wait().Times(1)

	err := m.manager.StopNode()
	if err != nil {
		t.Fatal(err)
	}

	if *m.manager.state != stopped {
		t.FailNow()
	}

	err = m.manager.StopNode()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}
	if *m.manager.state != stopped {
		t.FailNow()
	}
}

func TestRestartNodeSuccess(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	m.newNode.EXPECT().Start().Times(2).Return(nil)
	m.newNode.EXPECT().GetNode().AnyTimes().Return(nil)
	m.managerNode.EXPECT().SetNode(m.newNode.GetNode()).Times(2)
	m.constr.EXPECT().Make().Times(2).Return(m.newNode, nil)
	m.constr.EXPECT().SetConfig(m.config).AnyTimes()
	m.constr.EXPECT().Config().AnyTimes().Return(m.config)
	m.logger.EXPECT().Init(m.config.LogFile, m.config.LogLevel).Times(1).Return()
	m.rpc.EXPECT().Init(gomock.Any(), gomock.Any()).Times(2).Return(nil)
	_ = m.manager.StartNode(m.constr)

	m.managerNode.EXPECT().Stop().Times(1).Return(nil)
	m.managerNode.EXPECT().Wait().Times(1)
	_ = m.manager.StartNode(m.constr)

	err := m.manager.RestartNode()
	if err != nil {
		t.Fatal(err)
	}

	if *m.manager.state != started {
		t.FailNow()
	}
}

func TestRestartNodeNoNodeError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	err := m.manager.RestartNode()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}

	if *m.manager.state == started {
		t.FailNow()
	}
}

func TestRestartNodeStopError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	m.newNode.EXPECT().Start().Times(1).Return(nil)
	m.newNode.EXPECT().GetNode().AnyTimes().Return(nil)
	m.managerNode.EXPECT().SetNode(m.newNode.GetNode()).Times(1)
	m.constr.EXPECT().Make().Times(1).Return(m.newNode, nil)
	m.constr.EXPECT().SetConfig(m.config).AnyTimes()
	m.constr.EXPECT().Config().AnyTimes().Return(m.config)
	m.logger.EXPECT().Init(m.config.LogFile, m.config.LogLevel).Times(1).Return()
	m.rpc.EXPECT().Init(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	_ = m.manager.StartNode(m.constr)

	errExpected := errors.New("expected error")
	m.managerNode.EXPECT().Stop().Times(1).Return(errExpected)
	_ = m.manager.StartNode(m.constr)

	err := m.manager.RestartNode()
	if !reflect.DeepEqual(err, errExpected) {
		t.Fatal(err)
	}

	if *m.manager.state != started {
		t.FailNow()
	}
}

func TestRestartNodeNewNodeError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	m.newNode.EXPECT().Start().Times(1).Return(nil)
	m.newNode.EXPECT().GetNode().AnyTimes().Return(nil)
	m.managerNode.EXPECT().SetNode(m.newNode.GetNode()).Times(1)
	m.constr.EXPECT().Make().Times(1).Return(m.newNode, nil)
	m.constr.EXPECT().SetConfig(m.config).AnyTimes()
	m.constr.EXPECT().Config().AnyTimes().Return(m.config)
	m.logger.EXPECT().Init(m.config.LogFile, m.config.LogLevel).Times(1).Return()
	m.rpc.EXPECT().Init(gomock.Any(), gomock.Any()).Times(1).Return(nil)

	m.managerNode.EXPECT().Stop().Times(1).Return(nil)
	m.managerNode.EXPECT().Wait().Times(1)
	_ = m.manager.StartNode(m.constr)

	errExpected := errors.New("expected error")
	m.constr.EXPECT().Make().Times(1).Return(nil, errExpected)

	err := m.manager.RestartNode()
	if !reflect.DeepEqual(err, errExpected) {
		t.Fatal(err)
	}

	if *m.manager.state != stopped {
		t.FailNow()
	}
}

func TestRestartNodeStartNodeError(t *testing.T) {
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
	_ = m.manager.StartNode(m.constr)

	errExpected := errors.New("expected error")
	m.managerNode.EXPECT().Stop().Times(1).Return(errExpected)

	err := m.manager.RestartNode()
	if !reflect.DeepEqual(err, errExpected) {
		t.Fatal(err)
	}

	if *m.manager.state != started {
		t.FailNow()
	}
}

func TestRPCClientSuccess(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	rpcClient := geth.RPCClient(nil)
	m.rpc.EXPECT().Client().Times(1).Return(rpcClient)

	rpc := m.manager.RPCClient()
	if !reflect.DeepEqual(rpc, rpcClient) {
		t.FailNow()
	}
}

func TestLightEthereumServiceSuccess(t *testing.T) {
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

	m.manager.node.(*geth.MockNode).EXPECT().Service(gomock.Any()).Times(1).Return(nil)

	_ = m.manager.StartNode(m.constr)

	l, err := m.manager.LightEthereumService()
	if err != nil {
		t.Fatal(err)
	}

	if l == nil {
		t.FailNow()
	}
}

func TestLightEthereumServiceNodeError(t *testing.T) {
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

	e := errors.New("error")
	m.manager.node.(*geth.MockNode).EXPECT().Service(gomock.Any()).Times(1).Return(e)

	_ = m.manager.StartNode(m.constr)

	_, err := m.manager.LightEthereumService()
	if !reflect.DeepEqual(err, ErrInvalidLightEthereumService) {
		t.Fatal(err)
	}
}

func TestLightEthereumServiceError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	_, err := m.manager.LightEthereumService()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}
}

func TestWhisperServiceSuccess(t *testing.T) {
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

	m.manager.node.(*geth.MockNode).EXPECT().Service(gomock.Any()).Times(1).Return(nil)

	_ = m.manager.StartNode(m.constr)

	w, err := m.manager.WhisperService()
	if err != nil {
		t.Fatal(err)
	}

	if w == nil {
		t.FailNow()
	}
}

func TestWhisperServiceNodeError(t *testing.T) {
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

	e := errors.New("error")
	m.manager.node.(*geth.MockNode).EXPECT().Service(gomock.Any()).Times(1).Return(e)

	_ = m.manager.StartNode(m.constr)

	_, err := m.manager.WhisperService()
	if !reflect.DeepEqual(err, ErrInvalidWhisperService) {
		t.Fatal(err)
	}
}

func TestWhisperServiceError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	_, err := m.manager.WhisperService()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}
}

func TestPublicWhisperAPISuccess(t *testing.T) {
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

	m.manager.node.(*geth.MockNode).EXPECT().Service(gomock.Any()).Times(1).Return(nil)

	_ = m.manager.StartNode(m.constr)

	w, err := m.manager.PublicWhisperAPI()
	if err != nil {
		t.Fatal(err)
	}

	if w == nil {
		t.FailNow()
	}
}

func TestPublicWhisperAPINodeError(t *testing.T) {
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

	e := errors.New("error")
	m.manager.node.(*geth.MockNode).EXPECT().Service(gomock.Any()).Times(1).Return(e)

	_ = m.manager.StartNode(m.constr)

	_, err := m.manager.PublicWhisperAPI()
	if !reflect.DeepEqual(err, ErrInvalidWhisperService) {
		t.Fatal(err)
	}
}

func TestPublicWhisperAPIError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	_, err := m.manager.PublicWhisperAPI()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}
}

func TestGetStatusBackendSuccess(t *testing.T) {
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

	m.manager.node.(*geth.MockNode).EXPECT().Service(gomock.Any()).Times(1).Return(nil)

	_ = m.manager.StartNode(m.constr)

	b, err := m.manager.GetStatusBackend()
	if err != nil {
		t.Fatal(err)
	}

	if b == nil {
		t.FailNow()
	}
}

func TestGetStatusBackendNodeError(t *testing.T) {
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

	e := errors.New("error")
	m.manager.node.(*geth.MockNode).EXPECT().Service(gomock.Any()).Times(1).Return(e)

	_ = m.manager.StartNode(m.constr)

	_, err := m.manager.GetStatusBackend()
	if !reflect.DeepEqual(err, ErrInvalidLightEthereumService) {
		t.Fatal(err)
	}
}

func TestGetStatusBackendError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	_, err := m.manager.GetStatusBackend()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}
}

func TestAccountManagerSuccess(t *testing.T) {
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

	accountManager := &accounts.Manager{}
	m.manager.node.(*geth.MockNode).EXPECT().AccountManager().Times(1).Return(accountManager)

	_ = m.manager.StartNode(m.constr)

	a, err := m.manager.AccountManager()
	if err != nil {
		t.Fatal(err)
	}

	if a == nil {
		t.FailNow()
	}
}

func TestAccountManagerNodeError(t *testing.T) {
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

	m.manager.node.(*geth.MockNode).EXPECT().AccountManager().Times(1).Return(nil)

	_ = m.manager.StartNode(m.constr)

	_, err := m.manager.AccountManager()
	if !reflect.DeepEqual(err, ErrInvalidAccountManager) {
		t.Fatal(err)
	}
}

func TestAccountManagerError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	_, err := m.manager.AccountManager()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}
}

func TestNodeSuccess(t *testing.T) {
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
	_ = m.manager.StartNode(m.constr)

	n, err := m.manager.Node()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(n, m.newNode) {
		t.FailNow()
	}
}

func TestNodeError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	_, err := m.manager.Node()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}

	if *m.manager.state != pending {
		t.FailNow()
	}
}

func TestNodeConfigSuccess(t *testing.T) {
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
	_ = m.manager.StartNode(m.constr)

	c, err := m.manager.NodeConfig()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(c, m.config) {
		t.FailNow()
	}
}

func TestNodeConfigError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	_, err := m.manager.NodeConfig()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
	}

	if *m.manager.state != pending {
		t.FailNow()
	}
}

func TestAccountKeyStoreNoBackendsError(t *testing.T) {
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

	accountManager := &accounts.Manager{}
	m.manager.node.(*geth.MockNode).EXPECT().AccountManager().Times(1).Return(accountManager)

	_ = m.manager.StartNode(m.constr)

	a, err := m.manager.AccountKeyStore()
	if !reflect.DeepEqual(err, ErrAccountKeyStoreMissing) {
		t.Fatal(err)
	}

	if a != nil {
		t.FailNow()
	}
}

func TestAccountKeyStoreNodeError(t *testing.T) {
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

	m.manager.node.(*geth.MockNode).EXPECT().AccountManager().Times(1).Return(nil)

	_ = m.manager.StartNode(m.constr)

	_, err := m.manager.AccountKeyStore()
	if !reflect.DeepEqual(err, ErrInvalidAccountManager) {
		t.Fatal(err)
	}
}

func TestAccountKeyStoreError(t *testing.T) {
	m := initMocks(t)
	defer m.ctrl.Finish()

	_, err := m.manager.AccountKeyStore()
	if !reflect.DeepEqual(err, ErrNoRunningNode) {
		t.Fatal(err)
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
