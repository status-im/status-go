package sharedsecret

type Response struct {
	Secret          []byte
	InstallationIDs map[string]bool
}

type Persistence interface {
	Add(identity []byte, secret []byte, installationID string) error
	Get(identity []byte, installationIDs []string) (*Response, error)
	All() ([][][]byte, error)
}
