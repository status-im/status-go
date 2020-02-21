# Waku spec support

*Last updated February 21, 2020*

status-go client of Waku is spec compliant with [Waku spec v0.4](https://specs.vac.dev/waku.html) with the exception of:
- Currently nodes with higher version don't automatically disconnect if versions are different

It doesn't yet implement the following recommended features:
- It doesn't apply a timeout to receive Status message
- Partial support for rate limiting
- No support for DNS discovery to find Waku nodes
- No support for negotiation with peer supporting multiple version

Additionally it makes the following choices:
- It doesn't enforce rate limit policy
