package requests

import "github.com/status-im/status-go/v10/multiaccounts"

type ImportUnencryptedDatabase struct {
	Account      multiaccounts.Account `json:"account"`
	Password     string                `json:"password"`
	DatabasePath string                `json:"databasePath"`
}
