package keygen

// GenerateKeyOptions contains all the settings that can be used when generating
// a keyfile with the generate-key command
type GenerateKeyOptions struct {
	KeyFile   string
	KeyPasswd string
	Overwrite bool
}
