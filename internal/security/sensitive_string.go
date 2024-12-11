package security

import (
	"encoding/json"
	"fmt"
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

func (s SensitiveString) NotEmpty() bool {
	return !s.Empty()
}

func (s SensitiveString) Plus(other SensitiveString) SensitiveString {
	return NewSensitiveString(s.value + other.value)
}

func (s SensitiveString) PlusString(other string) SensitiveString {
	return NewSensitiveString(s.value + other)
}
