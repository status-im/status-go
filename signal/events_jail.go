package signal

const (
	EventVmConsole  = "vm.console"
	EventJailSignal = "jail.signal"
)

// SendConsole is a signal sent when jail writes anything to console.
func SendConsole(args interface{}) {
	sendSignal(Envelope{
		Type:  EventVmConsole,
		Event: args,
	})
}

// SendJailSignal is nobody knows what.
// TODO(divan, adamb): investigate if this even needed.
func SendJailSignal(cellID, message string) {
	sendSignal(Envelope{
		Type: EventJailSignal,
		Event: struct {
			ChatID string `json:"chat_id"`
			Data   string `json:"data"`
		}{
			ChatID: cellID,
			Data:   message,
		},
	})
}
