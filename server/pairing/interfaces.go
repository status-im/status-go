package pairing

type ForSenders interface {
	_forSenders()
}

type ForReceivers interface {
	_forReceivers()
}

type ForServers interface {
	_forServers()
}

type ForClients interface {
	_forClients()
}

func (sc *SenderConfig) _forSenders()     {}
func (rc *ReceiverConfig) _forReceivers() {}
func (sc *ServerConfig) _forServers()     {}
func (sc *ClientConfig) _forClients()     {}
