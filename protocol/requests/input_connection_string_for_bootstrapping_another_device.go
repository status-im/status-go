package requests

type InputConnectionStringForBootstrappingAnotherDevice struct {
	ConnectionString string `json:"connectionString"`
	ConfigJSON       string `json:"configJSON"`
}
