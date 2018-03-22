package transactions

import (
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/golang/mock/gomock"
	"github.com/status-im/status-go/geth/account"
)

// MockAccountManager is a mock of AccountManager interface
type MockAccountManager struct {
	ctrl     *gomock.Controller
	recorder *MockAccountManagerMockRecorder
}

// MockAccountManagerMockRecorder is the mock recorder for MockAccountManager
type MockAccountManagerMockRecorder struct {
	mock *MockAccountManager
}

// NewMockAccountManager creates a new mock instance
func NewMockAccountManager(ctrl *gomock.Controller) *MockAccountManager {
	mock := &MockAccountManager{ctrl: ctrl}
	mock.recorder = &MockAccountManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAccountManager) EXPECT() *MockAccountManagerMockRecorder {
	return m.recorder
}

// VerifyAccountPassword mocks base method
func (m *MockAccountManager) VerifyAccountPassword(keyStoreDir, address, password string) (*keystore.Key, error) {
	ret := m.ctrl.Call(m, "VerifyAccountPassword", keyStoreDir, address, password)
	ret0, _ := ret[0].(*keystore.Key)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// VerifyAccountPassword indicates an expected call of VerifyAccountPassword
func (mr *MockAccountManagerMockRecorder) VerifyAccountPassword(keyStoreDir, address, password interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyAccountPassword", reflect.TypeOf((*MockAccountManager)(nil).VerifyAccountPassword), keyStoreDir, address, password)
}

// SelectedAccount mocks base method
func (m *MockAccountManager) SelectedAccount() (*account.SelectedExtKey, error) {
	ret := m.ctrl.Call(m, "SelectedAccount")
	ret0, _ := ret[0].(*account.SelectedExtKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SelectedAccount indicates an expected call of SelectedAccount
func (mr *MockAccountManagerMockRecorder) SelectedAccount() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelectedAccount", reflect.TypeOf((*MockAccountManager)(nil).SelectedAccount))
}
