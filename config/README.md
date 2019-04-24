# Introduction

This document describes the available options in the JSON config for `status-go`.

The structure of the JSON config is defined in the [`params/config.go`](/params/config.go) file, which also contains detailed comments on meaning of each option. The `NodeConfig` struct defines the general configuration keys at the __root__ of the JSON file.

If the descriptions of any options are too vague feel free to [open an issue](https://github.com/status-im/status-go/issues/new).

Example config files can be viewed in the [`config/cli`](/config/cli) folder.

# Important Sections

The JSON config is separated into several sections. The most important ones are listed below.

## `NodeConfig`

The root of the JSON configuration.

An example of most important settings would include:
```json
{
    "NetworkID": 1,
    "DataDir": "/tmp/status-go-data",
    "NodeKey": "123qwe123qwe123qwe123",
    "Rendezvous": true,
    "NoDiscovery": false,
    "ListenAddr": "0.0.0.0:30303",
    "AdvertiseAddr": "12.34.56.78",
    "RegisterTopics": ["whispermail"]
}
```

If you'd want to enable JSON RPC port you'd need:
```json
{
    "HTTPEnabled": true,
    "HTTPHost": "0.0.0.0",
    "HTTPPort": 8545,
    "APIModules": "eth,net,web3,admin"
}
```

In order to adjust logging settings you'd need:
```json
{
    "LogFile": "/var/log/status-go.log",
    "LogLevel": "INFO",
    "LogMaxSize": 200,
    "LogMaxBackups": 5,
    "LogCompressRotated": true
}
```
Valid `LogLevel` settings are: `ERROR`, `WARN`, `INFO`, `DEBUG`, `TRACE`

## `WhisperConfig` 

If you want your node to relay Whisper(SHH) protocol messages you'll want to include this:
```json
{
    "WhisperConfig": {
        "Enabled": true,
        "EnableMailServer": true,
        "LightClient": false,
        "MailServerPassword": "status-offline-inbox"
    }
}
```
The `MailServerPassword` is used for symmetric encryption of history requests.

__NOTE:__ The default password used by Status App and [our mailservers](https://fleets.status.im/) is `status-offline-inbox`.

## `ClusterConfig`

This config manages what peers and bootstrap nodes your `status-go` instance connects when it starts.
```json
{
  "ClusterConfig": {
    "Enabled": true,
    "Fleet": "eth.beta",
    "BootNodes": [
      "enode://345ert345ert@23.45.67.89:30404"
    ],
    "TrustedMailServers": [
      "enode://qwe123qwe123@98.76.54.32:30504"
    ],
    "StaticNodes": [
      "enode://123qwe123qwe@12.34.56.78:30305"
    ],
    "RendezvousNodes": [
      "/ip4/87.65.43.21/tcp/30703/ethv4/16Uiu2HAm312312312312312312312312"
    ]
  }
}
```
`BootNodes` help the `status-go` instance find peers. They are more important to have than `StaticNodes` or `TrustedMailServers`, which are just statically added peers on start.

`RendezvousNodes` are an alternative protocol for doing the same peer discovery as `BootNodes` do.
