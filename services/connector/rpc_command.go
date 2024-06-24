package connector

type RPCCommand interface {
	Execute(inputJSON string) (string, error)
}
