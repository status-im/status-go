lib
===

**DEPRECATED. Please see [`./mobile`](./mobile/REDME.md) instead. Currently, the exported bindings are not in sync with `mobile`. They might be missing or working differently.**

This package provides CGO bindings so that it can be used to create a static library and link it with other projects in different languages, namely [status-react](https://github.com/status-im/status-react). Even though, we switched to [gomobile](./mobile/REDME.md), this package can still be useful for generating a static library to link with C/C++ program.

Bindings are exported and described in `library.go`. 
