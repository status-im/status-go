### Package
version     = "0.0.1"
author      = "Status Research & Development GmbH"
description = "Status messaging and wallet backend, built as a C archive for status-app"
license     = "MPL-2.0"

# status-go has no Nim code: this package only resolves its Nim dependencies.
# The Makefile builds them.
srcDir = "internal/nimble/src"


### Dependencies
# Any newer Nim is fine; the bindings' lock picks the one that builds libsds.
requires "nim >= 2.2.6"

# Pins the nim-sds revision whose C ABI the bindings match.
requires "https://github.com/logos-messaging/sds-go-bindings#aba9548c"

# The untagged chronos commit nim-ffi needs; the libsds compile uses this tree.
requires "chronos#0de7b335d0ad5557ad5ba71a4b7662f7b201750e"
