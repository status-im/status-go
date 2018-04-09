package sdk

var (
	generateSymKeyFromPasswordFormat = `{"jsonrpc":"2.0","id":2950,"method":"shh_generateSymKeyFromPassword","params":["%s"]}`
	newMessageFilterFormat           = `{"jsonrpc":"2.0","id":2,"method":"shh_newMessageFilter","params":[{"allowP2P":true,"topics":["%s"],"type":"sym","symKeyID":"%s"}]}`
	getFilterMessagesFormat          = `{"jsonrpc":"2.0","id":2968,"method":"shh_getFilterMessages","params":["%s"]}`
	standardMessageFormat            = `{"jsonrpc":"2.0","id":633,"method":"shh_post","params":[{"sig":"%s","symKeyID":"%s","payload":"%s","topic":"%s","ttl":10,"powTarget":%g,"powTime":1}]}`
	messagePayloadFormat             = `["~#c4",["%s","text/plain","~:public-group-user-message",%d,%d]]`
	web3ShaFormat                    = `{"jsonrpc":"2.0","method":"web3_sha3","params":["%s"],"id":%d}`
)
