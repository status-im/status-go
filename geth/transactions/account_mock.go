package transactions

import (
	"reflect"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
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

// CreateAccount mocks base method
func (m *MockAccountManager) CreateAccount(password string) (string, string, string, error) {
	ret := m.ctrl.Call(m, "CreateAccount", password)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// CreateAccount indicates an expected call of CreateAccount
func (mr *MockAccountManagerMockRecorder) CreateAccount(password interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateAccount", reflect.TypeOf((*MockAccountManager)(nil).CreateAccount), password)
}

// CreateChildAccount mocks base method
func (m *MockAccountManager) CreateChildAccount(parentAddress, password string) (string, string, error) {
	ret := m.ctrl.Call(m, "CreateChildAccount", parentAddress, password)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CreateChildAccount indicates an expected call of CreateChildAccount
func (mr *MockAccountManagerMockRecorder) CreateChildAccount(parentAddress, password interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateChildAccount", reflect.TypeOf((*MockAccountManager)(nil).CreateChildAccount), parentAddress, password)
}

// RecoverAccount mocks base method
func (m *MockAccountManager) RecoverAccount(password, mnemonic string) (string, string, error) {
	ret := m.ctrl.Call(m, "RecoverAccount", password, mnemonic)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// RecoverAccount indicates an expected call of RecoverAccount
func (mr *MockAccountManagerMockRecorder) RecoverAccount(password, mnemonic interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecoverAccount", reflect.TypeOf((*MockAccountManager)(nil).RecoverAccount), password, mnemonic)
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

// SelectAccount mocks base method
func (m *MockAccountManager) SelectAccount(address, password string) error {
	ret := m.ctrl.Call(m, "SelectAccount", address, password)
	ret0, _ := ret[0].(error)
	return ret0
}

// SelectAccount indicates an expected call of SelectAccount
func (mr *MockAccountManagerMockRecorder) SelectAccount(address, password interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelectAccount", reflect.TypeOf((*MockAccountManager)(nil).SelectAccount), address, password)
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

// Logout mocks base method
func (m *MockAccountManager) Logout() error {
	ret := m.ctrl.Call(m, "Logout")
	ret0, _ := ret[0].(error)
	return ret0
}

// Logout indicates an expected call of Logout
func (mr *MockAccountManagerMockRecorder) Logout() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Logout", reflect.TypeOf((*MockAccountManager)(nil).Logout))
}

// Accounts mocks base method
func (m *MockAccountManager) Accounts() ([]common.Address, error) {
	ret := m.ctrl.Call(m, "Accounts")
	ret0, _ := ret[0].([]common.Address)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Accounts indicates an expected call of Accounts
func (mr *MockAccountManagerMockRecorder) Accounts() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Accounts", reflect.TypeOf((*MockAccountManager)(nil).Accounts))
}

// AddressToDecryptedAccount mocks base method
func (m *MockAccountManager) AddressToDecryptedAccount(address, password string) (accounts.Account, *keystore.Key, error) {
	ret := m.ctrl.Call(m, "AddressToDecryptedAccount", address, password)
	ret0, _ := ret[0].(accounts.Account)
	ret1, _ := ret[1].(*keystore.Key)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// AddressToDecryptedAccount indicates an expected call of AddressToDecryptedAccount
func (mr *MockAccountManagerMockRecorder) AddressToDecryptedAccount(address, password interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddressToDecryptedAccount", reflect.TypeOf((*MockAccountManager)(nil).AddressToDecryptedAccount), address, password)
}
