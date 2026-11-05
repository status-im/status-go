# Builds the Nim libraries status-go links, from one Nimble resolution.

# A consumer passes its Nim build configuration: the compiler (NIM), the root
# of its resolution (NIMBLE_ROOT), the target (ARCH, ANDROID_API,
# ANDROID_NDK_ROOT, IPHONE_SDK, IOS_TARGET) and its policy (NIM_PARAMS).
NIMBLE ?= nimble
NIMBLE_ROOT ?= $(CURDIR)
# A resolved package's directory, from nimble.paths: `nimble path` lists every
# installed version. Windows paths come with (escaped) backslashes and CRLF.
NIM_PKG_DIR = $$(sed -n -E 's@[\\]+@/@g; s@^--path:"(.*/pkgs2/$(1)-[^/"]*)"[[:space:]]*$$@\1@p' "$(NIMBLE_ROOT)/nimble.paths" | head -n 1)

ifeq ($(abspath $(NIMBLE_ROOT)),$(CURDIR))
nimble.paths: status_go.nimble
	# nimble path lists every installed version, so keep only this resolution's.
	rm -rf nimbledeps
	env -u SDKROOT -u CPATH -u LIBRARY_PATH $(NIMBLE) setup --localdeps -y
else
ifeq ($(origin NIM),undefined)
    $(error NIMBLE_ROOT is set, so pass its compiler as NIM)
endif
# The consumer resolves its own root.
.PHONY: nimble.paths
nimble.paths:
	@test -f "$(NIMBLE_ROOT)/nimble.paths" || { echo "Error: $(NIMBLE_ROOT)/nimble.paths not found" >&2; exit 1; }
endif
# The Nim nimble installed for this resolution, else the one on PATH.
NIM ?= $(or $(firstword $(wildcard $(CURDIR)/nimbledeps/pkgs2/nim-*/bin/nim $(CURDIR)/nimbledeps/pkgs2/nim-*/bin/nim.exe)),nim)

.PHONY: nimble-deps
nimble-deps: nimble.paths ##@build Resolve the Nim dependencies

# The default policy. A consumer's NIM_PARAMS replaces it.
NIM_POLICY_DESKTOP = --mm:refc --opt:size -d:chronicles_line_numbers --warning:Deprecated:off --warning:UnusedImport:on -d:chronicles_log_level=TRACE
NIM_POLICY_ANDROID = --mm:refc --opt:size -d:chronicles_log_level=ERROR
NIM_POLICY_IOS = --mm:refc --opt:size -d:useMalloc
# libtkl runs on ORC with threads only (abi/libtkl.nim checks).
NIM_POLICY_TKL = --mm:orc -d:useMalloc --threads:on -d:release

NIM_MAC_ARCH = $(if $(filter x86_64 amd64,$(or $(ARCH),$(shell uname -m))),x86_64,arm64)
NIM_MAC_FLAGS = -arch $(NIM_MAC_ARCH) -isysroot $$(xcrun --show-sdk-path)
ifeq ($(detected_OS),Darwin)
    NIM_FLAGS_DESKTOP = --cpu:$(if $(filter x86_64,$(NIM_MAC_ARCH)),amd64,arm64) --passC:\"$(NIM_MAC_FLAGS)\" --passL:\"$(NIM_MAC_FLAGS)\"
endif

NIM_ANDROID_CPU = $(strip $(if $(filter arm64,$(ARCH)),arm64,\
	$(if $(filter arm,$(ARCH)),arm,\
	$(if $(filter amd64 x86 x86_64,$(ARCH)),amd64,\
	$(error Unsupported ARCH '$(ARCH)'. Please set ARCH to one of: arm64, arm, amd64, x86, x86_64)))))
NIM_ANDROID_TRIPLE = $(if $(filter arm64,$(NIM_ANDROID_CPU)),aarch64-linux-android,$(if $(filter arm,$(NIM_ANDROID_CPU)),armv7a-linux-androideabi,x86_64-linux-android))
NIM_ANDROID_BIN = $(ANDROID_NDK_ROOT)/toolchains/llvm/prebuilt/$(or $(HOST_OS),linux)-x86_64/bin
NIM_ANDROID_CC = $(NIM_ANDROID_BIN)/$(NIM_ANDROID_TRIPLE)$(or $(ANDROID_API),28)-clang
NIM_FLAGS_ANDROID = --os:android --cpu:$(NIM_ANDROID_CPU) --cc:clang --clang.exe:$(NIM_ANDROID_CC) --clang.linkerexe:$(NIM_ANDROID_CC)

NIM_IOS_ARCH = $(or $(ARCH),arm64)
NIM_IOS_SDK = $(or $(IPHONE_SDK),iphoneos)
NIM_IOS_FLAGS = -arch $(NIM_IOS_ARCH) -isysroot $$(xcrun --sdk $(NIM_IOS_SDK) --show-sdk-path) -m$(if $(filter iphonesimulator,$(NIM_IOS_SDK)),ios-simulator,iphoneos)-version-min=$(or $(IOS_TARGET),13.0)
NIM_FLAGS_IOS = --cpu:$(if $(filter x86_64,$(NIM_IOS_ARCH)),amd64,arm64) --os:ios --cc:clang --passC:\"$(NIM_IOS_FLAGS)\" --passL:\"$(NIM_IOS_FLAGS)\"

NIM_FLAGS = $(NIM_FLAGS_DESKTOP)
NIM_POLICY = $(NIM_POLICY_DESKTOP)

# Builds a library's entry with plain nim c from this resolution, so it builds
# from the resolution and not from its own lock. nimble shell puts the selected
# Nim on PATH. nimble.paths has every --path, so parent configs (status-app's)
# are skipped; the library's own config next to its entry is still read.
# eval: NIM_FLAGS and nimble.paths quote their arguments.
# $(call NIM_COMPILE,<entry .nim>,<lib|staticlib>,<output file in NIM_LIB_DIR>[,<flags>])
NIM_COMPILE = eval "env -u SDKROOT -u CPATH -u LIBRARY_PATH $(NIM) c --skipParentCfg \
	$(NIM_FLAGS) $${NIM_PARAMS:-$(NIM_POLICY)} $$(tr '\r\n' '  ' < "$(NIMBLE_ROOT)/nimble.paths") \
	--app:$(2) $(4) --nimcache:\"$(NIM_LIB_DIR)/nimcache\" -o:\"$(NIM_LIB_DIR)/$(3)\" \"$(1)\""

# Seals a static library built into a fresh NIM_LIB_DIR/nimcache: only its C
# API (<prefix>*) stays global, so its Nim runtime can't clash with another's.
# A cross build seals with the target's tools (NIM_SEAL_TOOLS), e.g. the NDK's.
# $(call NIM_SEAL,<C API prefix>,<sealed archive>)
NIM_SEAL_TOOLS_ANDROID = NIM_SEAL_LD="$(NIM_ANDROID_BIN)/ld.lld" NIM_SEAL_OBJCOPY="$(NIM_ANDROID_BIN)/llvm-objcopy" NIM_SEAL_AR="$(NIM_ANDROID_BIN)/llvm-ar"
NIM_SEAL = $(NIM_SEAL_TOOLS) bash scripts/seal_nim_lib.sh $(1) "$(NIM_LIB_DIR)/nimcache" "$(2)"
