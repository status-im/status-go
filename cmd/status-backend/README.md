# Description

Welcome to `status-backend`. This is a tool for debugging and testing `status-go`.
In contrast to existing `statusd` and `status-cli`, the `status-backend` exposes full status-go API through HTTP.

This allows to communicate with status-go through HTTP the same way as `status-desktop` and `status-mobile` do, including: 
- create account
- restore account
- login
- logout
- start messenger
- start wallet 
- subscribe to status-go signals
- etc.

# status-go API

## Public methods in `./mobile/status.go`

Only specific function signatures are currently supported:
   - `func(string) string` - 1 argument, 1 return 
   - `func() string` - 0 argument, 1 return

### Unsupported methods

Attempt to call any other functions will return `501: Not Implemented` HTTP code.  
For example, [`VerifyAccountPassword`](https://github.com/status-im/status-go/blob/669256095e16d953ca1af4954b90ca2ae65caa2f/mobile/status.go#L275-L277) has 3 arguments: 
```go
func VerifyAccountPassword(keyStoreDir, address, password string) string {
    return logAndCallString(verifyAccountPassword, keyStoreDir, address, password)
}
```

Later, as needed, a V2 of these functions will be introduced. V2 will have a single JSON argument composing all args in 1.  
For example, https://github.com/status-im/status-go/pull/5865 fixes some of these.

### Deprecated methods

Deprecated methods will have `Deprecation: true` HTTP header.

## Signals in `./signal`

Each signal has [this structure](https://github.com/status-im/status-go/blob/c9b777a2186364b8f394ad65bdb18b128ceffa70/signal/signals.go#L30-L33):
```go
// Envelope is a general signal sent upward from node to RN app
type Envelope struct {
    Type  string      `json:"type"`
    Event interface{} `json:"event"`
}
```

List of possible events can be found in `./signal/event_*.go` files. 

For example, `node.login` event is defined [here](https://github.com/status-im/status-go/blob/6bcf5f1289f9160168574290cbd6f90dede3f8f6/signal/events_node.go#L27-L28):
```go
const (
    // EventLoggedIn is once node was injected with user account and ready to be used.
    EventLoggedIn = "node.login"
)
```

And the structure of this event is [defined in the same file](https://github.com/status-im/status-go/blob/6bcf5f1289f9160168574290cbd6f90dede3f8f6/signal/events_node.go#L36-L42):
```go
// NodeLoginEvent returns the result of the login event
type NodeLoginEvent struct {
	Error        string                 `json:"error,omitempty"`
	Settings     *settings.Settings     `json:"settings,omitempty"`
	Account      *multiaccounts.Account `json:"account,omitempty"`
	EnsUsernames json.RawMessage        `json:"ensUsernames,omitempty"`
}
```


So the signal for `node.login` event will look like this (with corresponding data):
```json
{
  "type": "node.login",
  "event": {
    "error": "",
    "settings": {},
    "account": {},
    "endUsernames": {} 
  }
}
```

## Services in `./services/**/api.go`

Services are registered in go-ethereum JSON-RPC server. To call such method, send request to `statusgo/CallRPC` endpoint.

For example:
```http request
### Send Contact Request
POST http://localhost:12345/statusgo/CallRPC

{
    "jsonrpc": "2.0",
    "method": "wakuext_sendContactRequest",
    "params": [
        {
            "id": "0x048f0b885010783429c2298b916e24b3c01f165e55fe8f98fce63df0a55ade80089f512943d4fde5f8c7211f1a87b267a85cbcb3932eb2e4f88aa4ca3918f97541",
            "message": "Hi, Alice!"
        }
    ]
}
```

### Notes

1. In this case, there's no limitation to the number of arguments, comparing to `mobile/status.go`, so ll method are supported.
2. Deprecated methods won't have a corresponding `Deprecated: true`

# Usage

Start the app with the address to listen to:
```shell
status-backend --address localhost:12345
```

Or just use the root repo Makefile command:
```shell
make run-status-backend PORT=12345
```

Access the exposed API with any HTTP client you prefer:
- From your IDE:
    - [JetBrains](https://www.jetbrains.com/help/idea/http-client-in-product-code-editor.html)
    - [VS Code](https://marketplace.visualstudio.com/items?itemName=humao.rest-client)
- From UI client:
    - [Postman](https://learning.postman.com/docs/getting-started/first-steps/sending-the-first-request/)
    - [Insomnia](https://docs.insomnia.rest/insomnia/send-your-first-request)
- From command line:
    - [Curl](https://curl.se/docs/httpscripting.html)
- From your script:
    - [Python](https://pypi.org/project/requests/)
    - [Go](https://pkg.go.dev/net/http)

# Simple flows

In most cases to start testing you'll need some boilerplate. Below are the simple call flows for common cases.

## Create account and login

1. `InitializeApplication`
2. `CreateAccountAndLogin`
3. `wakuext_startMessenger`
4. `wallet_startWallet`
5. `settings_getSettings` (temporary workaround, otherwise settings don't get saved into DB)

## Login into account

1. `InitializeApplication`
2. `LoginAccount`
3. `wakuext_startMessenger`
4. `wallet_startWallet`