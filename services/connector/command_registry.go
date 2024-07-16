package connector

type CommandRegistry struct {
	commands map[string]RPCCommand
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]RPCCommand),
	}
}

func (r *CommandRegistry) Register(method string, command RPCCommand) {
	r.commands[method] = command
}

func (r *CommandRegistry) GetCommand(method string) (RPCCommand, bool) {
	command, exists := r.commands[method]
	return command, exists
}
