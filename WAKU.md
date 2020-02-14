# Waku spec support

*Last updated February 14, 2020*

status-nim client of Waku is spec compliant with [Waku spec v0.3](https://specs.vac.dev/waku.html) with the exception of:
- It doesn't support all the MUST packet codes (!)
- Currently nodes with higher version don't automatically disconnect if versions are different

It doesn't yet implement the following recommended features:
- It doesn't disconnect a peer if it receives a message before a Status message
- Partial support for rate limiting
- No support for DNS discovery to find Waku nodes
- No support for negotiation with peer supporting multiple version
- Exchange topic-interest packet code isn't implemented yet (WIP status-im/status-go#1853)

Additionally it makes the following choices:
- It doesn't enforce rate limit policy
