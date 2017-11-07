ottoext
=======

Originally based on [github.com/deoxxa/ottoext](https://github.com/deoxxa/ottoext)

[![GoDoc](https://godoc.org/github.com/status-im/status-go/geth/jail/ottoext?status.svg)](https://godoc.org/github.com/status-im/status-go/geth/jail/ottoext)

Overview
--------

This package contains some extensions for the otto JavaScript interpreter. The
most important extension is a generic event loop based on code from natto. The
other extensions are `setTimeout` and `setInterval` support, `Promise` support
(via `native-promise-only`, MIT license), and `fetch` support.

Examples
--------

Take a look at the test files to see how the extensions work.

License
-------

3-clause BSD. A copy is included with the source.
