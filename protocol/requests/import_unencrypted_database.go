package requests

type ImportUnencryptedDatabase struct {
	AccountData  string `json:"accountData"`
	Password     string `json:"password"`
	DatabasePath string `json:"databasePath"`
}
