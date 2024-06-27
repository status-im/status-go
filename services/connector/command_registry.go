package connector

import "github.com/status-im/status-go/services/connector/commands"

type CommandRegistry struct {
	commands map[string]commands.RPCCommand
}

func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]commands.RPCCommand),
	}
}

func (r *CommandRegistry) Register(method string, command commands.RPCCommand) {
	r.commands[method] = command
}

func (r *CommandRegistry) GetCommand(method string) (commands.RPCCommand, bool) {
	command, exists := r.commands[method]
	return command, exists
}
