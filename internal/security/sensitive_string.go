package security

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

const RedactionPlaceholder = "***"

// SensitiveString is a type for handling sensitive information securely.
// This helps to achieve the following goals:
//  1. Prevent accidental logging of sensitive information.
//  2. Provide controlled visibility (e.g., redacted output for String() or MarshalJSON()).
//  3. Enable controlled access to the sensitive value when needed.
type SensitiveString struct {
	value string
}

// NewSensitiveString creates a new SensitiveString
func NewSensitiveString(value string) SensitiveString {
	return SensitiveString{value: value}
}

func NewSensitiveStringPrintf(format string, args ...interface{}) SensitiveString {
	str := fmt.Sprintf(format, args...)
	return NewSensitiveString(str)
}

// String provides a redacted version of the sensitive string
func (s SensitiveString) String() string {
	if s.value == "" {
		return ""
	}
	return RedactionPlaceholder
}

// MarshalJSON ensures that sensitive strings are redacted when marshaled to JSON
// NOTE: It's important to define this method on the value receiver,
// otherwise `json.Marshal` will not call this method.
func (s SensitiveString) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements unmarshalling a sensitive string from JSON
// NOTE: It's important to define this method on the pointer receiver,
// otherwise `json.Marshal` will not call this method.
func (s *SensitiveString) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	s.value = value
	return nil
}

// Reveal exposes the sensitive value (use with caution)
func (s SensitiveString) Reveal() string {
	return s.value
}

// Empty checks if the value is empty
func (s SensitiveString) Empty() bool {
	return s.value == ""
}

func (s SensitiveString) TrimRight(cutset any) SensitiveString {
	return NewSensitiveString(strings.TrimRight(s.value, getValue(cutset)))
}

func (s SensitiveString) Contains(substr any) bool {
	return strings.Contains(s.value, getValue(substr))
}

func (s SensitiveString) Append(others ...any) SensitiveString {
	result := s.value
	for _, other := range others {
		result += getValue(other)
	}
	return NewSensitiveString(result)
}

// getValue extracts the string value from various types
func getValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case SensitiveString:
		return x.value
	default:
		panic(fmt.Sprintf("unsupported type: %T", v))
	}
}

// Value implements the driver.Valuer interface for SensitiveString
func (s SensitiveString) Value() (driver.Value, error) {
	return s.value, nil
}

// Scan implements the sql.Scanner interface for SensitiveString
func (s *SensitiveString) Scan(value interface{}) error {
	switch v := value.(type) {
	case nil:
		s.value = ""
	case string:
		s.value = v
	case []byte:
		s.value = string(v)
	default:
		return fmt.Errorf("cannot scan type %T into SensitiveString", value)
	}
	return nil
}
