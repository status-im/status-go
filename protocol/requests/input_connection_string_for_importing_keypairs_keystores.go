package requests

type InputConnectionStringForImportingKeypairsKeystores struct {
	ConnectionString string `json:"connectionString"`
	ConfigJSON       string `json:"configJSON"`
}
