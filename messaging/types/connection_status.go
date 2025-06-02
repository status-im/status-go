package types

type ConnectionStatus struct {
	IsOnline bool `json:"isOnline"`
}

type ConnectionStatusSubscription interface {
	C() <-chan ConnectionStatus
	Unsubscribe()
}
