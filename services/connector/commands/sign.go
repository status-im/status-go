package commands

type SignCommand struct {
}

// TODO: Implement the signing
func (c *SignCommand) Execute(request RPCRequest) (string, error) {
	return "signed", nil
}
