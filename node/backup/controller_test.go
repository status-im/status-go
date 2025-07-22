package backup

import (
	"encoding/json"
	"encoding/xml"
	"reflect"
	"testing"

	"github.com/brianvoe/gofakeit/v6"

	"go.uber.org/zap"

	"github.com/stretchr/testify/require"
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
	controller, err := NewController(BackupConfig{
		FileNameGetter: func() (string, error) { return filename, nil },
		PrivateKey:     []byte("0123456789abcdef0123456789abcdef"),
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
