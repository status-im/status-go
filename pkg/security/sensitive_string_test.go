package security

import (
	"encoding/json"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/suite"
)

// SensitiveStringSuite defines a testify suite for testing SensitiveString
type SensitiveStringSuite struct {
	suite.Suite
}

// SensitiveStringSuite is the test suite for all SensitiveString behaviors.
func (s *SensitiveStringSuite) TestNewSensitiveString() {
	secretValue := gofakeit.LetterN(10)
	ss := NewSensitiveString(secretValue)
	s.Require().Equal(secretValue, ss.Reveal())
}

func (s *SensitiveStringSuite) TestStringRedaction() {
	secretValue := gofakeit.LetterN(10)
	ss := NewSensitiveString(secretValue)
	s.Require().Equal(RedactionPlaceholder, ss.String())
}

func (s *SensitiveStringSuite) TestEmptyStringRedaction() {
	ss := NewSensitiveString("")
	s.Require().Equal("", ss.String())
}

func (s *SensitiveStringSuite) TestMarshalJSON() {
	secretValue := gofakeit.LetterN(10)
	ss := NewSensitiveString(secretValue)

	data, err := json.Marshal(ss)
	s.Require().NoError(err)
	s.Require().JSONEq(`"`+RedactionPlaceholder+`"`, string(data))
}

func (s *SensitiveStringSuite) TestMarshalJSONPointer() {
	secretValue := gofakeit.LetterN(10)
	sVal := NewSensitiveString(secretValue)

	data, err := json.Marshal(&sVal)
	s.Require().NoError(err)
	s.Require().JSONEq(`"`+RedactionPlaceholder+`"`, string(data))
}

func (s *SensitiveStringSuite) TestUnmarshalJSON() {
	secretValue := gofakeit.LetterN(10)
	payload := `"` + secretValue + `"`
	var ss SensitiveString

	s.Require().NoError(json.Unmarshal([]byte(payload), &ss))
	s.Require().Equal(secretValue, ss.Reveal())
}

func (s *SensitiveStringSuite) TestUnmarshalJSONError() {
	// Can't unmarshal a non-string value
	var ss SensitiveString
	payload := `{"key":"value"}`
	s.Require().Error(json.Unmarshal([]byte(payload), &ss))
}

func (s *SensitiveStringSuite) TestCopySensitiveString() {
	secretValue := gofakeit.LetterN(10)
	ss := NewSensitiveString(secretValue)
	ssCopy := ss
	s.Require().Equal(secretValue, ssCopy.Reveal())
}

func (s *SensitiveStringSuite) TestTrimRight() {
	const secretValue = "¡¡¡Hello, Gophers!!!" // #nosec G101
	s1 := NewSensitiveString(secretValue)

	s.Require().Equal(
		s1.TrimRight("!"),
		NewSensitiveString("¡¡¡Hello, Gophers"),
	)
}

func (s *SensitiveStringSuite) TestContains() {
	const secretValue = "¡¡¡Hello, Gophers!!!" // #nosec G101
	s1 := NewSensitiveString(secretValue)

	s.Require().True(s1.Contains("Hello"))
	s.Require().False(s1.Contains("World"))
}

func (s *SensitiveStringSuite) TestAppend() {
	secretValue := gofakeit.LetterN(10)
	s1 := NewSensitiveString(secretValue)
	s2 := NewSensitiveString(secretValue)

	s.Require().Equal(s1.Append(s2), NewSensitiveString(secretValue+secretValue))
	s.Require().Equal(s1.Append(secretValue), NewSensitiveString(secretValue+secretValue))
	s.Require().Equal(s1.Append(secretValue, secretValue), NewSensitiveString(secretValue+secretValue+secretValue))
}

// Entry point for the suite
func TestSensitiveStringSuite(t *testing.T) {
	suite.Run(t, new(SensitiveStringSuite))
}
