package mocksettings

import (
	json "encoding/json"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	types "github.com/status-im/status-go/eth-node/types"
	settings "github.com/status-im/status-go/multiaccounts/settings"
)

// GetSettingLastSynced mocks base method.
func (m *MockDatabaseSettingsManager) GetSettingLastSynced(setting settings.SettingField) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSettingLastSynced", setting)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSettingLastSynced indicates an expected call of GetSettingLastSynced.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetSettingLastSynced(setting interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSettingLastSynced", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetSettingLastSynced), setting)
}

// GetNotificationsEnabled mocks base method.
func (m *MockDatabaseSettingsManager) GetNotificationsEnabled() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNotificationsEnabled")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNotificationsEnabled indicates an expected call of GetNotificationsEnabled.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetNotificationsEnabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNotificationsEnabled", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetNotificationsEnabled))
}

// GetProfilePicturesVisibility mocks base method.
func (m *MockDatabaseSettingsManager) GetProfilePicturesVisibility() (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProfilePicturesVisibility")
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProfilePicturesVisibility indicates an expected call of GetProfilePicturesVisibility.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetProfilePicturesVisibility() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProfilePicturesVisibility", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetProfilePicturesVisibility))
}

// GetPublicKey mocks base method.
func (m *MockDatabaseSettingsManager) GetPublicKey() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPublicKey")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPublicKey indicates an expected call of GetPublicKey.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetPublicKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPublicKey", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetPublicKey))
}

// GetFleet mocks base method.
func (m *MockDatabaseSettingsManager) GetFleet() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFleet")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFleet indicates an expected call of GetFleet.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetFleet() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFleet", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetFleet))
}

// GetDappsAddress mocks base method.
func (m *MockDatabaseSettingsManager) GetDappsAddress() (types.Address, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDappsAddress")
	ret0, _ := ret[0].(types.Address)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDappsAddress indicates an expected call of GetDappsAddress.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetDappsAddress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDappsAddress", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetDappsAddress))
}

// GetPinnedMailservers mocks base method.
func (m *MockDatabaseSettingsManager) GetPinnedMailservers() (map[string]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPinnedMailservers")
	ret0, _ := ret[0].(map[string]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPinnedMailservers indicates an expected call of GetPinnedMailservers.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetPinnedMailservers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPinnedMailservers", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetPinnedMailservers))
}

// GetDefaultSyncPeriod mocks base method.
func (m *MockDatabaseSettingsManager) GetDefaultSyncPeriod() (uint32, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDefaultSyncPeriod")
	ret0, _ := ret[0].(uint32)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDefaultSyncPeriod indicates an expected call of GetDefaultSyncPeriod.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetDefaultSyncPeriod() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDefaultSyncPeriod", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetDefaultSyncPeriod))
}

// GetMessagesFromContactsOnly mocks base method.
func (m *MockDatabaseSettingsManager) GetMessagesFromContactsOnly() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMessagesFromContactsOnly")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMessagesFromContactsOnly indicates an expected call of GetMessagesFromContactsOnly.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetMessagesFromContactsOnly() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMessagesFromContactsOnly", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetMessagesFromContactsOnly))
}

// GetProfilePicturesShowTo mocks base method.
func (m *MockDatabaseSettingsManager) GetProfilePicturesShowTo() (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProfilePicturesShowTo")
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetProfilePicturesShowTo indicates an expected call of GetProfilePicturesShowTo.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetProfilePicturesShowTo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProfilePicturesShowTo", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetProfilePicturesShowTo))
}

// GetLatestDerivedPath mocks base method.
func (m *MockDatabaseSettingsManager) GetLatestDerivedPath() (uint, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLatestDerivedPath")
	ret0, _ := ret[0].(uint)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLatestDerivedPath indicates an expected call of GetLatestDerivedPath.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetLatestDerivedPath() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLatestDerivedPath", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetLatestDerivedPath))
}

// GetCurrentStatus mocks base method.
func (m *MockDatabaseSettingsManager) GetCurrentStatus(status interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCurrentStatus", status)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetCurrentStatus indicates an expected call of GetCurrentStatus.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetCurrentStatus(status interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCurrentStatus", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetCurrentStatus), status)
}

// GetWalletRootAddress mocks base method.
func (m *MockDatabaseSettingsManager) GetWalletRootAddress() (types.Address, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWalletRootAddress")
	ret0, _ := ret[0].(types.Address)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWalletRootAddress indicates an expected call of GetWalletRootAddress.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetWalletRootAddress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWalletRootAddress", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetWalletRootAddress))
}

// GifAPIKey mocks base method.
func (m *MockDatabaseSettingsManager) GifAPIKey() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GifAPIKey")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GifAPIKey indicates an expected call of GifAPIKey.
func (mr *MockDatabaseSettingsManagerMockRecorder) GifAPIKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GifAPIKey", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GifAPIKey))
}

// GifFavorites mocks base method.
func (m *MockDatabaseSettingsManager) GifFavorites() (json.RawMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GifFavorites")
	ret0, _ := ret[0].(json.RawMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GifFavorites indicates an expected call of GifFavorites.
func (mr *MockDatabaseSettingsManagerMockRecorder) GifFavorites() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GifFavorites", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GifFavorites))
}

// GifRecents mocks base method.
func (m *MockDatabaseSettingsManager) GifRecents() (json.RawMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GifRecents")
	ret0, _ := ret[0].(json.RawMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GifRecents indicates an expected call of GifRecents.
func (mr *MockDatabaseSettingsManagerMockRecorder) GifRecents() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GifRecents", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GifRecents))
}
