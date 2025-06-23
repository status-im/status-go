package backup

import (
	"encoding/json"
	"encoding/xml"
	"reflect"
	"testing"

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

func TestController(t *testing.T) {
	filename := t.TempDir() + "/test_backup.bak"
	controller, err := NewController(BackupConfig{
		FileNameGetter: func() string { return filename },
		PrivateKey:     []byte("0123456789abcdef0123456789abcdef"),
	})
	require.NoError(t, err)

	foo := Foo{
		Value:        123,
		PreciseValue: 456.789,
	}
	bar := Bar{
		Names:   []string{"Bob", "Tom"},
		Surname: "Smith",
	}

	var fooFromBackup Foo
	var barFromBackup Bar

	controller.Register("foo", func() ([]byte, error) { return json.Marshal(foo) }, func(data []byte) error { return json.Unmarshal(data, &fooFromBackup) })
	controller.Register("bar", func() ([]byte, error) { return xml.Marshal(bar) }, func(data []byte) error { return xml.Unmarshal(data, &barFromBackup) })

	err = controller.PerformBackup()
	require.NoError(t, err)

	require.False(t, reflect.DeepEqual(bar, barFromBackup))
	require.False(t, reflect.DeepEqual(foo, fooFromBackup))

	err = controller.LoadBackup(filename)
	require.NoError(t, err)

	require.True(t, reflect.DeepEqual(bar, barFromBackup))
	require.True(t, reflect.DeepEqual(foo, fooFromBackup))
}
