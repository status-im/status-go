package security

import (
	"encoding/json"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

func TestNewSensitiveString(t *testing.T) {
	secretValue := gofakeit.LetterN(10)
	s := NewSensitiveString(secretValue)
	require.Equal(t, secretValue, s.Reveal())
}

func TestStringRedaction(t *testing.T) {
	secretValue := gofakeit.LetterN(10)
	s := NewSensitiveString(secretValue)
	require.Equal(t, redactionPlaceholder, s.String())
}

func TestEmptyStringRedaction(t *testing.T) {
	s := NewSensitiveString("")
	require.Equal(t, "", s.String())
}

func TestMarshalJSON(t *testing.T) {
	secretValue := gofakeit.LetterN(10)
	s := NewSensitiveString(secretValue)
	data, err := json.Marshal(s)
	require.NoError(t, err)
	require.JSONEq(t, `"`+redactionPlaceholder+`"`, string(data))
}

func TestUnmarshalJSON(t *testing.T) {
	secretValue := gofakeit.LetterN(10)
	data := `"` + secretValue + `"`
	var s SensitiveString
	err := json.Unmarshal([]byte(data), &s)
	require.NoError(t, err)
	require.Equal(t, secretValue, s.Reveal())
}

func TestUnamarshalJSONError(t *testing.T) {
	// Can't unmarshal a non-string value
	var s SensitiveString
	data := `{"key": "value"}`
	err := json.Unmarshal([]byte(data), &s)
	require.Error(t, err)
}

func TestSetValue(t *testing.T) {
	initialValue := gofakeit.LetterN(10)
	newValue := gofakeit.LetterN(10)
	s := NewSensitiveString(initialValue)
	s.SetValue(newValue)
	require.Equal(t, newValue, s.Reveal())
}
