/*
Package rpc - JSON-RPC client with custom routing.

Package rpc implements status-go JSON-RPC client and handles
requests to different implementations, namely the upstream and local nodes, and
special handlers in status-go.

Every JSON-RPC request coming from either JS code or any other part
of status-go should use this package to be handled and routed properly.

Routing rules are as follows:

- If a Handler has been registered, use it.
- If Upstream is disabled, everything is routed to local ethereum-go node
- Otherwise, some requests (see RoutingTable) are routed to upstream, others locally.

List of methods to be routed is currently available here: https://docs.google.com/spreadsheets/d/1N1nuzVN5tXoDmzkBLeC9_mwIlVH8DGF7YD2XwxA8BAE/edit#gid=0

Note, upon creation of a new client, it ok to be offline - client will keep trying to reconnect in background.

*/
package rpc

//go:generate autoreadme -f
