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
requires "https://github.com/logos-messaging/nim-sds#beed22e6fd6bdfd8abf6f2d083bdd4ed95966256"

# The native token library; go.mod pins its Go wrapper at the same commit.
requires "https://github.com/status-im/nim-token-lists#801a9ec48a0082e7efef546d76f7c842e5a55a35"

# The untagged chronos commit nim-ffi needs; the libsds compile uses this tree.
requires "chronos#0de7b335d0ad5557ad5ba71a4b7662f7b201750e"

# liblogosdelivery: logos-messaging/logos-delivery#4683, which a caller builds
# with plain nim c.
requires "https://github.com/alexjba/logos-delivery#1838f6aa13aced8d88477f0498d571fd8c15487f"
# Its Leopard-RS (via nim-segmentation), at the revision logos-delivery locks:
# logos-storage/nim-leopard#34 configures it without its test executables.
requires "https://github.com/alexjba/nim-leopard#fa58e24ffef2fdbf79dee546a5da52a53c0b6d19"
