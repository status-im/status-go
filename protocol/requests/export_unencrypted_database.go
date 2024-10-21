package requests

import "github.com/status-im/status-go/multiaccounts"

type ExportUnencryptedDatabase struct {
	Account      multiaccounts.Account `json:"account"`
	Password     string                `json:"password"`
	DatabasePath string                `json:"databasePath"`
}
