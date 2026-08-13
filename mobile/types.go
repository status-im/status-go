package statusgo

// APIResponse generic response from API.
type APIResponse struct {
	Error string `json:"error"`
}

// APIKeyUIDResponse
type APIKeyUIDResponse struct {
	KeyUID string `json:"keyUID"`
}

// APIFieldError represents a set of errors
// related to a parameter.
type APIFieldError struct {
	Parameter string     `json:"parameter,omitempty"`
	Errors    []APIError `json:"errors"`
}

// APIError represents a single error.
type APIError struct {
	Message string `json:"message"`
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
