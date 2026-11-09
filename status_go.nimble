### Package
version     = "0.0.1"
author      = "Status Research & Development GmbH"
description = "Status messaging and wallet backend, built as a C archive for status-app"
license     = "MPL-2.0"

# status-go has no Nim code: this package only resolves its Nim dependencies.
# The Makefile builds them.
srcDir = "internal/nimble/src"


### Dependencies
# Any newer Nim is fine.
requires "nim >= 2.2.6"

# nim-sds, whose C ABI go.mod's sds-go-bindings must match.
requires "https://github.com/logos-messaging/nim-sds#8f7fee08f0256c1788089e756182c8be0a621930"

# The native token library, v0.2.0; go.mod pins its Go wrapper at the same commit.
requires "https://github.com/status-im/nim-token-lists#4df3308a32b0f3da18ef0d37d26fa9f1195f8b9e"

# The untagged chronos commit nim-ffi needs; the libsds compile uses this tree.
requires "chronos#0de7b335d0ad5557ad5ba71a4b7662f7b201750e"
