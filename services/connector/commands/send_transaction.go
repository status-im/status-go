package commands

type SendTransactionCommand struct {
}

// TODO: Implement the sending transaction
func (c *SendTransactionCommand) Execute(request RPCRequest) (string, error) {
	return "transaction sent", nil
}
