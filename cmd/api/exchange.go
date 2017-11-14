package api

// NoArgs has to be used when a remote function needs
// no concrete argument for its work.
type NoArgs bool

// NoReply has to be used when a remote function provides
// no concrete reply beside a potential error.
type NoReply bool

// ConfigArgs is used to pass a configuration as argument.
type ConfigArgs struct {
	Config string
}

// AccountArgs is used to pass information about address
// and/or password as argument.
type AccountArgs struct {
	Address  string
	Password string
}

// AccountReply is used to return information about a created account.
type AccountReply struct {
	Address   string
	PublicKey string
	Mnemonic  string
}

// StringsReply is used to return a number of strings. Need
// to be wrapped.
type StringsReply struct {
	Strings []string
}
