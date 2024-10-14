package requests

type InputConnectionStringForBootstrapping struct {
	ConnectionString string `json:"connectionString"`
	ConfigJSON       string `json:"configJSON"`
}
