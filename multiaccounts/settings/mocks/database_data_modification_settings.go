package mocksettings

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	settings "github.com/status-im/status-go/multiaccounts/settings"
	params "github.com/status-im/status-go/params"
)

// CreateSettings mocks base method.
func (m *MockDatabaseSettingsManager) CreateSettings(s settings.Settings, n params.NodeConfig) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSettings", s, n)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateSettings indicates an expected call of CreateSettings.
func (mr *MockDatabaseSettingsManagerMockRecorder) CreateSettings(s, n interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSettings", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).CreateSettings), s, n)
}

// SaveSetting mocks base method.
func (m *MockDatabaseSettingsManager) SaveSetting(setting string, value interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveSetting", setting, value)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveSetting indicates an expected call of SaveSetting.
func (mr *MockDatabaseSettingsManagerMockRecorder) SaveSetting(setting, value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveSetting", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SaveSetting), setting, value)
}

// SaveSettingField mocks base method.
func (m *MockDatabaseSettingsManager) SaveSettingField(sf settings.SettingField, value interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveSettingField", sf, value)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveSettingField indicates an expected call of SaveSettingField.
func (mr *MockDatabaseSettingsManagerMockRecorder) SaveSettingField(sf, value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveSettingField", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SaveSettingField), sf, value)
}

// DeleteMnemonic mocks base method.
func (m *MockDatabaseSettingsManager) DeleteMnemonic() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteMnemonic")
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteMnemonic indicates an expected call of DeleteMnemonic.
func (mr *MockDatabaseSettingsManagerMockRecorder) DeleteMnemonic() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteMnemonic", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).DeleteMnemonic))
}

// SaveSyncSetting mocks base method.
func (m *MockDatabaseSettingsManager) SaveSyncSetting(setting settings.SettingField, value interface{}, clock uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveSyncSetting", setting, value, clock)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveSyncSetting indicates an expected call of SaveSyncSetting.
func (mr *MockDatabaseSettingsManagerMockRecorder) SaveSyncSetting(setting, value, clock interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveSyncSetting", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SaveSyncSetting), setting, value, clock)
}

// SetSettingLastSynced mocks base method.
func (m *MockDatabaseSettingsManager) SetSettingLastSynced(setting settings.SettingField, clock uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetSettingLastSynced", setting, clock)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetSettingLastSynced indicates an expected call of SetSettingLastSynced.
func (mr *MockDatabaseSettingsManagerMockRecorder) SetSettingLastSynced(setting, clock interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSettingLastSynced", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SetSettingLastSynced), setting, clock)
}

// SetSettingsNotifier mocks base method.
func (m *MockDatabaseSettingsManager) SetSettingsNotifier(n settings.Notifier) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetSettingsNotifier", n)
}

// SetSettingsNotifier indicates an expected call of SetSettingsNotifier.
func (mr *MockDatabaseSettingsManagerMockRecorder) SetSettingsNotifier(n interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSettingsNotifier", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SetSettingsNotifier), n)
}

// SetLastBackup mocks base method.
func (m *MockDatabaseSettingsManager) SetLastBackup(time uint64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetLastBackup", time)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetLastBackup indicates an expected call of SetLastBackup.
func (mr *MockDatabaseSettingsManagerMockRecorder) SetLastBackup(time interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetLastBackup", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SetLastBackup), time)
}

// SetBackupFetched mocks base method.
func (m *MockDatabaseSettingsManager) SetBackupFetched(fetched bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetBackupFetched", fetched)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetBackupFetched indicates an expected call of SetBackupFetched.
func (mr *MockDatabaseSettingsManagerMockRecorder) SetBackupFetched(fetched interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetBackupFetched", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SetBackupFetched), fetched)
}

// SetPinnedMailservers mocks base method.
func (m *MockDatabaseSettingsManager) SetPinnedMailservers(mailservers map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetPinnedMailservers", mailservers)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetPinnedMailservers indicates an expected call of SetPinnedMailservers.
func (mr *MockDatabaseSettingsManagerMockRecorder) SetPinnedMailservers(mailservers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPinnedMailservers", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SetPinnedMailservers), mailservers)
}

// SetTokenGroupByCommunity mocks base method.
func (m *MockDatabaseSettingsManager) SetTokenGroupByCommunity(value bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetTokenGroupByCommunity", value)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetTokenGroupByCommunity indicates an expected call of SetTokenGroupByCommunity.
func (mr *MockDatabaseSettingsManagerMockRecorder) SetTokenGroupByCommunity(value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTokenGroupByCommunity", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SetTokenGroupByCommunity), value)
}

// GetTelemetryServerURL mocks base method.
func (m *MockDatabaseSettingsManager) GetTelemetryServerURL() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTelemetryServerURL")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTelemetryServerURL indicates an expected call of GetTelemetryServerURL.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetTelemetryServerURL() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTelemetryServerURL", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetTelemetryServerURL))
}

// GetTestNetworksEnabled mocks base method.
func (m *MockDatabaseSettingsManager) GetTestNetworksEnabled() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTestNetworksEnabled")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTestNetworksEnabled indicates an expected call of GetTestNetworksEnabled.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetTestNetworksEnabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTestNetworksEnabled", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetTestNetworksEnabled))
}

// GetTokenGroupByCommunity mocks base method.
func (m *MockDatabaseSettingsManager) GetTokenGroupByCommunity() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTokenGroupByCommunity")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTokenGroupByCommunity indicates an expected call of GetTokenGroupByCommunity.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetTokenGroupByCommunity() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTokenGroupByCommunity", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetTokenGroupByCommunity))
}

// URLUnfurlingMode mocks base method.
func (m *MockDatabaseSettingsManager) URLUnfurlingMode() (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "URLUnfurlingMode")
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// URLUnfurlingMode indicates an expected call of URLUnfurlingMode.
func (mr *MockDatabaseSettingsManagerMockRecorder) URLUnfurlingMode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "URLUnfurlingMode", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).URLUnfurlingMode))
}
