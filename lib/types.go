package main

import (
	"bytes"
	"fmt"
	"strings"
)

// APIResponse generic response from API.
type APIResponse struct {
	Error string `json:"error"`
}

// APIDetailedResponse represents a generic response
// with possible errors.
type APIDetailedResponse struct {
	Status      bool            `json:"status"`
	Message     string          `json:"message,omitempty"`
	FieldErrors []APIFieldError `json:"field_errors,omitempty"`
}

// Error string representation of APIDetailedResponse.
func (r APIDetailedResponse) Error() string {
	buf := bytes.NewBufferString("")

	for _, err := range r.FieldErrors {
		buf.WriteString(err.Error() + "\n") // nolint: gas
	}

	return strings.TrimSpace(buf.String())
}

// APIFieldError represents a set of errors
// related to a parameter.
type APIFieldError struct {
	Parameter string     `json:"parameter,omitempty"`
	Errors    []APIError `json:"errors"`
}

// Error string representation of APIFieldError.
func (e APIFieldError) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}

	buf := bytes.NewBufferString(fmt.Sprintf("Parameter: %s\n", e.Parameter))

	for _, err := range e.Errors {
		buf.WriteString(err.Error() + "\n") // nolint: gas
	}

	return strings.TrimSpace(buf.String())
}

// APIError represents a single error.
type APIError struct {
	Message string `json:"message"`
}

// Error string representation of APIError.
func (e APIError) Error() string {
	return fmt.Sprintf("message=%s", e.Message)
}

// AccountInfo represents account's info.
type AccountInfo struct {
	Address  string `json:"address"`
	PubKey   string `json:"pubkey"`
	Mnemonic string `json:"mnemonic"`
	Error    string `json:"error"`
}

// NotifyResult is a JSON returned from notify message.
type NotifyResult struct {
	Status bool   `json:"status"`
	Error  string `json:"error,omitempty"`
}

// SignRequestResult is a JSON returned from transaction complete function (used in exposed method)
type SignRequestResult struct {
	ID    string `json:"id"`
	Hash  string `json:"hash"`
	Error string `json:"error"`
}

// SignRequestsResult is list of results from CompleteTransactions() (used in exposed method)
type SignRequestsResult struct {
	Results map[string]SignRequestResult `json:"results"`
}

// DiscardSignRequestResult is a JSON returned from transaction discard function
type DiscardSignRequestResult struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// DiscardSignRequestsResult is a list of results from DiscardTransactions()
type DiscardSignRequestsResult struct {
	Results map[string]DiscardSignRequestResult `json:"results"`
}
