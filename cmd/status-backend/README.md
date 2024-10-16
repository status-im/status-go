# 📝Description

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

# 📍status-go API

> [!NOTE]  
> Unfortunately, for now there is no convenient docs like OpenAPI

## 1️⃣ Public methods in `./mobile/status.go`

### Description

Any **public** functions in `./mobile/status.go` with the one of the following signatures:
   - `func(string) string` - 1 argument, 1 return 
   - `func() string` - 0 argument, 1 return

### Address

Endpoints address: `http://<address>/statusgo/<function-name>`.  

Here, `statusgo` is the name of the package in `/mobile/status.go`. We might create more APIs next to it in the future.

### Response

Responses have JSON body with a single `error` field.  
If `error` is empty, no error occurred.

The structure of response is defined in [`APIResponse`](https://github.com/status-im/status-go/blob/91c6949cd25449d5459581a21f2c8b929290ced0/mobile/types.go#L9-L12):
```go
// APIResponse generic response from API.
type APIResponse struct {
    Error string `json:"error"`
}
```

### Parameters

Parameters are passed as request JSON body.  
Specific parameters for each endpoint should be checked in the source code.

### How to read the source code?

As example, let's look at [`CreateAccountAndLogin`](https://github.com/status-im/status-go/blob/fc36a7e980fc8f73dd36d7e3db29675ddff1d5dc/mobile/status.go#L415-L417).
```go
func CreateAccountAndLogin(requestJSON string) string {
	return callWithResponse(createAccountAndLogin, requestJSON)
}
```

1. The function has 1 JSON string argument and 1 JSON string return value.
2. The function wraps a private `createAccountAndLogin` with `callWithResponse`.  
`callWithResponse` does some internal magic like logging the request and response.   
So we should check [the private function](https://github.com/status-im/status-go/blob/fc36a7e980fc8f73dd36d7e3db29675ddff1d5dc/mobile/status.go#L419) for real arguments and return values:
    ```go
    func createAccountAndLogin(requestJSON string) string {
        var request requests.CreateAccount
        err := json.Unmarshal([]byte(requestJSON), &request)
        // ...
    }
    ```
3. We can see that the parameters are unmarshalled to [`requests.CreateAccount`](https://github.com/status-im/status-go/blob/f660be0daa7cc742a07f6a139ea5ac9966f3ebe0/protocol/requests/create_account.go#L35):
    ```go
    type CreateAccount struct {
        RootDataDir   string `json:"rootDataDir"`
        KdfIterations int    `json:"kdfIterations"`
        // ...
    ```
4. We can also see that each the response is formed as [`makeJSONResponse(err)`](https://github.com/status-im/status-go/blob/fc36a7e980fc8f73dd36d7e3db29675ddff1d5dc/mobile/status.go#L422-L424):
    ```go
    if err != nil {
        return makeJSONResponse(err)
    }
    ```
   ... which wraps the error to [`APIResponse`](https://github.com/status-im/status-go/blob/91c6949cd25449d5459581a21f2c8b929290ced0/mobile/types.go#L9-L12):
    ```go
    // APIResponse generic response from API.
    type APIResponse struct {
        Error string `json:"error"`
    }
    ```

### Unsupported methods

Attempt to call any functions with unsupported signatures will return `501: Not Implemented` HTTP code.  
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

## 2️⃣ Signals in `./signal`

> [!NOTE]  
> Unfortunately, there is no description of when any expected signals will appear.  
> For now, you have to check the source code.

### Address

Signals are available at `ws://<address>/signals`.

Connect to it as the first thing when running `status-backend`.

### Available signals

List of possible events can be found in `./signal/event_*.go` files.

For example, `node.login` event is defined [here](https://github.com/status-im/status-go/blob/6bcf5f1289f9160168574290cbd6f90dede3f8f6/signal/events_node.go#L27-L28):
```go
const (
    // EventLoggedIn is once node was injected with user account and ready to be used.
    EventLoggedIn = "node.login"
)
```

### Signals structure

Each signal has [this structure](https://github.com/status-im/status-go/blob/c9b777a2186364b8f394ad65bdb18b128ceffa70/signal/signals.go#L30-L33):
```go
// Envelope is a general signal sent upward from node to RN app
type Envelope struct {
    Type  string      `json:"type"`
    Event interface{} `json:"event"`
}
```

Here, `type` is the name of the event, e.g. `node.login`.     
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

## 3️⃣ Services in `./services/**/api.go`

Services are registered in the [go-ethereum JSON-RPC](https://geth.ethereum.org/docs/interacting-with-geth/rpc) server.   
All `./services/**/api.go` are registered as services in geth. 

Each method name has form `<namespace>_<method>`. In most cases namespace is the directory name, but it can be ensured in the `APIs` method of each service. For example, for [wallet service](https://github.com/status-im/status-go/blob/1d173734a608de2d71480d6ad39f4559f11a75e2/services/wallet/service.go#L288-L298):
```go
// APIs returns list of available RPC APIs.
func (s *Service) APIs() []gethrpc.API {
    return []gethrpc.API{
        {
            Namespace: "wallet",
            Version:   "0.1.0",
            Service:   NewAPI(s),
            Public:    true,
        },
    }
}
```

### Address

These methods are available through `/statusgo/CallRPC` endpoint defined [here](https://github.com/status-im/status-go/blob/fc36a7e980fc8f73dd36d7e3db29675ddff1d5dc/mobile/status.go#L249-L260).

This is the way desktop and mobile clients call these methods. You don't have to run a separate geth HTTP server for this.

### Arguments

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

Please reference to the source code for the list of methods and its arguments.

### Notes

1. In this case, there's no limitation to the number of arguments, comparing to `mobile/status.go`, so ll method are supported.
2. Deprecated methods won't have a corresponding `Deprecated: true`

# 🏃‍♂️Usage

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

# 👌 Simple flows

In most cases to start testing you'll need some boilerplate. Below are the simple call flows for common cases.

## Create account and login

1. Subscribe to `/signals`
2. Call  `/statusgo/InitializeApplication`
3. Create an account
   1. Call `/statusgo/CreateAccountAndLogin`
   2. Wait for `node.login` signal 
      - If `error` is empty, no error occurred
      - If `error` is not empty, stop the operation
4. Start required services 
   1. Call  `/statusgo/CallRPC` with `{ "method": "wakuext_startMessenger", "params": [] }`
   2. Call  `/statusgo/CallRPC` with `{ "method": "wallet_startWallet", "params": [] }`
5. Apply temporary workarounds:
   1. Call  `/statusgo/CallRPC` with `{ "method": "settings_getSettings", "params": [] }`  
   _(otherwise settings don't get saved into DB)_

## Login into account

1. Subscribe to `/signals`
2. Call  `/statusgo/InitializeApplication`
3. Create an account
   1. Call `/statusgo/LoginAccount`
   2. Wait for `node.login` signal
      - If `error` is empty, no error occurred
      - If `error` is not empty, stop the operation
4. Start required services
   1. Call  `/statusgo/CallRPC` with `{ "method": "wakuext_startMessenger", "params": [] }`
   2. Call  `/statusgo/CallRPC` with `{ "method": "wallet_startWallet", "params": [] }`
