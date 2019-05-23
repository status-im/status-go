package signal

const (
	// EventChatsDidChange is triggered when there is new data in any of the subscriptions
	EventChatsDidChange = "status.chats.did-change"
)

func SendChatsDidChangeEvent(name string) {
	send(EventChatsDidChange, name)
}
