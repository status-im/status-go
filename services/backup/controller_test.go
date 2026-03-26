package backup

import (
	"encoding/json"
	"encoding/xml"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"go.uber.org/mock/gomock"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/common"
	mock_backup_controller "github.com/status-im/status-go/services/backup/mock"
)

type Foo struct {
	Value        int
	PreciseValue float64
}

type Bar struct {
	Names   []string
	Surname string
}

type FooProvider struct {
	foo Foo
}

func (f FooProvider) ExportBackup() ([]byte, error) {
	return json.Marshal(f.foo)
}

var fooFromBackup Foo

func (b FooProvider) ImportBackup(data []byte) error {
	return json.Unmarshal(data, &fooFromBackup)
}

type BarProvider struct {
	bar Bar
}

func (b BarProvider) ExportBackup() ([]byte, error) {
	return xml.Marshal(b.bar)
}

var barFromBackup Bar

func (b BarProvider) ImportBackup(data []byte) error {
	return xml.Unmarshal(data, &barFromBackup)
}

func TestController(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	filename := t.TempDir() + "/test_backup.bak"

	ctrl := gomock.NewController(t)
	filenameProvider := mock_backup_controller.NewMockFilenameProvider(ctrl)
	filenameProvider.EXPECT().GetBackupFilename().Return(filename, nil).AnyTimes()

	controller, err := NewController(Config{
		FileNameProvider: filenameProvider,
		PrivateKey:       []byte("0123456789abcdef0123456789abcdef"),
	}, logger)
	require.NoError(t, err)

	foo := Foo{}
	bar := Bar{}
	err = gofakeit.Struct(&foo)
	require.NoError(t, err)
	err = gofakeit.Struct(&bar)
	require.NoError(t, err)

	fooProvider := FooProvider{
		foo: foo,
	}

	barProvider := BarProvider{
		bar: bar,
	}

	controller.Register("foo", fooProvider)
	controller.Register("bar", barProvider)

	filename, err = controller.PerformBackup()
	require.NoError(t, err)
	require.Equal(t, filename, filename)

	require.False(t, reflect.DeepEqual(barProvider.bar, barFromBackup))
	require.False(t, reflect.DeepEqual(fooProvider.foo, fooFromBackup))

	err = controller.LoadBackup(filename)
	require.NoError(t, err)

	require.True(t, reflect.DeepEqual(barProvider.bar, barFromBackup))
	require.True(t, reflect.DeepEqual(fooProvider.foo, fooFromBackup))
}

type countingProvider struct {
	hits *atomic.Int32
}

func (p countingProvider) ExportBackup() ([]byte, error) {
	p.hits.Add(1)
	return []byte(`{"ok":true}`), nil
}

func (p countingProvider) ImportBackup([]byte) error {
	return nil
}

func TestControllerLifecycleState(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	filenameProvider := mock_backup_controller.NewMockFilenameProvider(ctrl)
	filenameProvider.EXPECT().GetBackupFilename().Return(t.TempDir()+"/lifecycle_backup.bak", nil).AnyTimes()

	controller, err := NewController(Config{
		FileNameProvider: filenameProvider,
		PrivateKey:       []byte("0123456789abcdef0123456789abcdef"),
		BackupEnabled:    true,
		Interval:         time.Hour, // long interval so backup doesn't fire during test
	}, logger)
	require.NoError(t, err)

	require.Equal(t, common.ServiceStateStopped, controller.PausableState())

	controller.Start()
	require.Equal(t, common.ServiceStateRunning, controller.PausableState())

	controller.Stop()
	require.Equal(t, common.ServiceStateStopped, controller.PausableState())
}

func TestControllerStartPausesAndResumesByLifecycle(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	filenameProvider := mock_backup_controller.NewMockFilenameProvider(ctrl)
	filenameProvider.EXPECT().GetBackupFilename().Return(t.TempDir()+"/pause_resume_backup.bak", nil).AnyTimes()

	controller, err := NewController(Config{
		FileNameProvider: filenameProvider,
		PrivateKey:       []byte("0123456789abcdef0123456789abcdef"),
		BackupEnabled:    true,
		// Long enough that the first tick cannot fire before MarkPaused propagates.
		Interval: time.Second,
	}, logger)
	require.NoError(t, err)

	var hits atomic.Int32
	controller.Register("counting", countingProvider{hits: &hits})
	controller.Start()
	// Pause immediately after Start. Subscribe() delivers the current pause
	// state as a snapshot, so the PausableTicker goroutine will see "paused"
	// regardless of whether it subscribes before or after this call.
	controller.MarkPaused()
	defer controller.Stop()

	time.Sleep(150 * time.Millisecond)
	require.Equal(t, int32(0), hits.Load())

	controller.MarkResumed()
	require.Eventually(t, func() bool {
		return hits.Load() > 0
	}, 3*time.Second, 20*time.Millisecond)
}
