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
func (suite *SensitiveStringSuite) TestNewSensitiveString() {
	secretValue := gofakeit.LetterN(10)
	s := NewSensitiveString(secretValue)
	suite.Require().Equal(secretValue, s.Reveal())
}

func (suite *SensitiveStringSuite) TestStringRedaction() {
	secretValue := gofakeit.LetterN(10)
	s := NewSensitiveString(secretValue)
	suite.Require().Equal(RedactionPlaceholder, s.String())
}

func (suite *SensitiveStringSuite) TestEmptyStringRedaction() {
	s := NewSensitiveString("")
	suite.Require().Equal("", s.String())
}

func (suite *SensitiveStringSuite) TestMarshalJSON() {
	secretValue := gofakeit.LetterN(10)
	s := NewSensitiveString(secretValue)

	data, err := json.Marshal(s)
	suite.Require().NoError(err)
	suite.Require().JSONEq(`"`+RedactionPlaceholder+`"`, string(data))
}

func (suite *SensitiveStringSuite) TestMarshalJSONPointer() {
	secretValue := gofakeit.LetterN(10)
	sVal := NewSensitiveString(secretValue)

	data, err := json.Marshal(&sVal)
	suite.Require().NoError(err)
	suite.Require().JSONEq(`"`+RedactionPlaceholder+`"`, string(data))
}

func (suite *SensitiveStringSuite) TestUnmarshalJSON() {
	secretValue := gofakeit.LetterN(10)
	payload := `"` + secretValue + `"`
	var s SensitiveString

	suite.Require().NoError(json.Unmarshal([]byte(payload), &s))
	suite.Require().Equal(secretValue, s.Reveal())
}

func (suite *SensitiveStringSuite) TestUnmarshalJSONError() {
	// Can't unmarshal a non-string value
	var s SensitiveString
	payload := `{"key":"value"}`
	suite.Require().Error(json.Unmarshal([]byte(payload), &s))
}

func (suite *SensitiveStringSuite) TestCopySensitiveString() {
	secretValue := gofakeit.LetterN(10)
	s := NewSensitiveString(secretValue)
	sCopy := s
	suite.Require().Equal(secretValue, sCopy.Reveal())
}

func (suite *SensitiveStringSuite) TestPlus() {
	secretValue := gofakeit.LetterN(10)
	s1 := NewSensitiveString(secretValue)
	s2 := NewSensitiveString(secretValue)

	suite.Require().Equal(s1.Plus(s2), NewSensitiveString(secretValue+secretValue))
}

func (suite *SensitiveStringSuite) TestPlusString() {
	secretValue := gofakeit.LetterN(10)
	s1 := NewSensitiveString(secretValue)

	suite.Require().Equal(s1.PlusString(secretValue), NewSensitiveString(secretValue+secretValue))
}

func (suite *SensitiveStringSuite) TestTrimRight() {
	secretValue := "¡¡¡Hello, Gophers!!!" // #nosec G101
	s1 := NewSensitiveString(secretValue)

	suite.Require().Equal(
		s1.TrimRight("!"),
		NewSensitiveString("¡¡¡Hello, Gophers"),
	)
}

func (suite *SensitiveStringSuite) TestContains() {
	secretValue := "¡¡¡Hello, Gophers!!!" // #nosec G101
	s1 := NewSensitiveString(secretValue)

	suite.Require().True(s1.Contains("Hello"))
	suite.Require().False(s1.Contains("World"))
}

// Entry point for the suite
func TestSensitiveStringSuite(t *testing.T) {
	suite.Run(t, new(SensitiveStringSuite))
}
