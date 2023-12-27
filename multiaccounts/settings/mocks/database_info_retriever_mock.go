// URLUnfurlingMode mocks base method.
package mocksettings

import (
	sql "database/sql"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	settings "github.com/status-im/status-go/multiaccounts/settings"
)

// GetDB mocks base method.
func (m *MockDatabaseSettingsManager) GetDB() *sql.DB {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDB")
	ret0, _ := ret[0].(*sql.DB)
	return ret0
}

// GetDB indicates an expected call of GetDB.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetDB() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDB", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetDB))
}

// GetSyncQueue mocks base method.
func (m *MockDatabaseSettingsManager) GetSyncQueue() chan settings.SyncSettingField {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSyncQueue")
	ret0, _ := ret[0].(chan settings.SyncSettingField)
	return ret0
}

// GetSyncQueue indicates an expected call of GetSyncQueue.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetSyncQueue() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSyncQueue", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetSyncQueue))
}

// GetChangesSubscriptions mocks base method.
func (m *MockDatabaseSettingsManager) GetChangesSubscriptions() []chan *settings.SyncSettingField {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetChangesSubscriptions")
	ret0, _ := ret[0].([]chan *settings.SyncSettingField)
	return ret0
}

// GetChangesSubscriptions indicates an expected call of GetChangesSubscriptions.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetChangesSubscriptions() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetChangesSubscriptions", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetChangesSubscriptions))
}

// GetNotifier mocks base method.
func (m *MockDatabaseSettingsManager) GetNotifier() settings.Notifier {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNotifier")
	ret0, _ := ret[0].(settings.Notifier)
	return ret0
}

// GetNotifier indicates an expected call of GetNotifier.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetNotifier() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNotifier", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetNotifier))
}

// GetSettings mocks base method.
func (m *MockDatabaseSettingsManager) GetSettings() (settings.Settings, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSettings")
	ret0, _ := ret[0].(settings.Settings)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSettings indicates an expected call of GetSettings.
func (mr *MockDatabaseSettingsManagerMockRecorder) GetSettings() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSettings", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).GetSettings))
}

// SubscribeToChanges mocks base method.
func (m *MockDatabaseSettingsManager) SubscribeToChanges() chan *settings.SyncSettingField {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeToChanges")
	ret0, _ := ret[0].(chan *settings.SyncSettingField)
	return ret0
}

// SubscribeToChanges indicates an expected call of SubscribeToChanges.
func (mr *MockDatabaseSettingsManagerMockRecorder) SubscribeToChanges() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeToChanges", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SubscribeToChanges))
}

// LastBackup mocks base method.
func (m *MockDatabaseSettingsManager) LastBackup() (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LastBackup")
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LastBackup indicates an expected call of LastBackup.
func (mr *MockDatabaseSettingsManagerMockRecorder) LastBackup() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LastBackup", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).LastBackup))
}

// Mnemonic mocks base method.
func (m *MockDatabaseSettingsManager) Mnemonic() (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Mnemonic")
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Mnemonic indicates an expected call of Mnemonic.
func (mr *MockDatabaseSettingsManagerMockRecorder) Mnemonic() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Mnemonic", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).Mnemonic))
}

// MnemonicRemoved mocks base method.
func (m *MockDatabaseSettingsManager) MnemonicRemoved() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MnemonicRemoved")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MnemonicRemoved indicates an expected call of MnemonicRemoved.
func (mr *MockDatabaseSettingsManagerMockRecorder) MnemonicRemoved() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MnemonicRemoved", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).MnemonicRemoved))
}

// MnemonicWasShown mocks base method.
func (m *MockDatabaseSettingsManager) MnemonicWasShown() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MnemonicWasShown")
	ret0, _ := ret[0].(error)
	return ret0
}

// MnemonicWasShown indicates an expected call of MnemonicWasShown.
func (mr *MockDatabaseSettingsManagerMockRecorder) MnemonicWasShown() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MnemonicWasShown", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).MnemonicWasShown))
}

// MutualContactEnabled mocks base method.
func (m *MockDatabaseSettingsManager) MutualContactEnabled() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MutualContactEnabled")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MutualContactEnabled indicates an expected call of MutualContactEnabled.
func (mr *MockDatabaseSettingsManagerMockRecorder) MutualContactEnabled() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MutualContactEnabled", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).MutualContactEnabled))
}

// ProfileMigrationNeeded mocks base method.
func (m *MockDatabaseSettingsManager) ProfileMigrationNeeded() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProfileMigrationNeeded")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ProfileMigrationNeeded indicates an expected call of ProfileMigrationNeeded.
func (mr *MockDatabaseSettingsManagerMockRecorder) ProfileMigrationNeeded() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProfileMigrationNeeded", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).ProfileMigrationNeeded))
}

// SetUseMailservers mocks base method.
func (m *MockDatabaseSettingsManager) SetUseMailservers(value bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetUseMailservers", value)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetUseMailservers indicates an expected call of SetUseMailservers.
func (mr *MockDatabaseSettingsManagerMockRecorder) SetUseMailservers(value interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetUseMailservers", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).SetUseMailservers), value)
}

// ShouldBroadcastUserStatus mocks base method.
func (m *MockDatabaseSettingsManager) ShouldBroadcastUserStatus() (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ShouldBroadcastUserStatus")
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ShouldBroadcastUserStatus indicates an expected call of ShouldBroadcastUserStatus.
func (mr *MockDatabaseSettingsManagerMockRecorder) ShouldBroadcastUserStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShouldBroadcastUserStatus", reflect.TypeOf((*MockDatabaseSettingsManager)(nil).ShouldBroadcastUserStatus))
}
