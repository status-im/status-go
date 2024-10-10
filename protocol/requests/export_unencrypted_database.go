package requests

type ExportUnencryptedDatabase struct {
    AccountData  string `json:"accountData"`
    Password     string `json:"password"`
    DatabasePath string `json:"databasePath"`
}
