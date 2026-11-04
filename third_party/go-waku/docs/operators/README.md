# The go-waku guide for operators

*If you're eager to get started, check out our [quickstart guide](./quickstart.md).*

go-waku is a client implementation in Go of the [Waku v2 family of protocols](https://rfc.vac.dev/spec/10/) for peer-to-peer communication.
The protocols are designed to be secure, privacy-preserving, censorship-resistant and able to run in resource restricted environments.
Moreover, we've taken a modular approach so that node operators can choose which protocols they want to support
based on their own motivations and availability of resources.
We call this concept ["adaptive nodes"](https://rfc.vac.dev/spec/30/),
implying that a Waku v2 network can consist of heterogeneous nodes contributing at different levels to the network.

This guide provides step-by-step tutorials covering how to build and configure your own go-waku node,
connect to an existing Waku v2 network
and use existing tools for monitoring and maintaining a running node.

## Helpful resources

<!-- TODO -->

## Getting in touch or reporting an issue

For an inquiry, or if you would like to propose new features, feel free to [open a general issue](https://github.com/waku-org/go-waku/issues/new/).

For bug reports, please [tag your issue with the `bug` label](https://github.com/waku-org/go-waku/issues/new/).

If you believe the reported issue requires critical attention, please [use the `critical` label](https://github.com/waku-org/go-waku/issues/new?labels=critical,bug) to assist with triaging.

To get help, or participate in the conversation, join the [Vac Discord](https://discord.gg/KNj3ctuZvZ) server.