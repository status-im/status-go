package onlinechecker

// OnlineChecker is used to determine if node has connectivity.
type OnlineChecker interface {
	IsOnline() bool
}

type DefaultOnlineChecker struct {
	online bool
}

func NewDefaultOnlineChecker(online bool) OnlineChecker {
	return &DefaultOnlineChecker{
		online: online,
	}
}

func (o *DefaultOnlineChecker) SetOnline(online bool) {
	o.online = online
}

func (o *DefaultOnlineChecker) IsOnline() bool {
	return o.online
}
