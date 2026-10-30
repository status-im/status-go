# Builds the Nim libraries status-go links, from one Nimble resolution.

NIMBLE ?= nimble

nimble.paths: status_go.nimble
	# nimble path lists every installed version, so keep only this resolution's.
	rm -rf nimbledeps
	env -u SDKROOT -u CPATH -u LIBRARY_PATH $(NIMBLE) setup --localdeps -y

.PHONY: nimble-deps
nimble-deps: nimble.paths ##@build Resolve the Nim dependencies

NIM_POLICY = --mm:refc --opt:size

NIM_MAC_ARCH = $(if $(filter x86_64 amd64,$(or $(ARCH),$(shell uname -m))),x86_64,arm64)
NIM_MAC_FLAGS = -arch $(NIM_MAC_ARCH) -isysroot $$(xcrun --show-sdk-path)
NIM_FLAGS_DESKTOP = -d:chronicles_line_numbers --warning:Deprecated:off --warning:UnusedImport:on -d:chronicles_log_level=TRACE
ifeq ($(detected_OS),Darwin)
    NIM_FLAGS_DESKTOP += --cpu:$(if $(filter x86_64,$(NIM_MAC_ARCH)),amd64,arm64) --passC:\"$(NIM_MAC_FLAGS)\" --passL:\"$(NIM_MAC_FLAGS)\"
endif

NIM_ANDROID_CPU = $(strip $(if $(filter arm64,$(ARCH)),arm64,\
	$(if $(filter arm,$(ARCH)),arm,\
	$(if $(filter amd64 x86 x86_64,$(ARCH)),amd64,\
	$(error Unsupported ARCH '$(ARCH)'. Please set ARCH to one of: arm64, arm, amd64, x86, x86_64)))))
NIM_ANDROID_TRIPLE = $(if $(filter arm64,$(NIM_ANDROID_CPU)),aarch64-linux-android,$(if $(filter arm,$(NIM_ANDROID_CPU)),armv7a-linux-androideabi,x86_64-linux-android))
NIM_ANDROID_CC = $(ANDROID_NDK_ROOT)/toolchains/llvm/prebuilt/$(or $(HOST_OS),linux)-x86_64/bin/$(NIM_ANDROID_TRIPLE)$(or $(ANDROID_API),28)-clang
NIM_FLAGS_ANDROID = --os:android --cpu:$(NIM_ANDROID_CPU) --cc:clang --clang.exe:$(NIM_ANDROID_CC) --clang.linkerexe:$(NIM_ANDROID_CC) -d:chronicles_log_level=ERROR

NIM_IOS_ARCH = $(or $(ARCH),arm64)
NIM_IOS_SDK = $(or $(IPHONE_SDK),iphoneos)
NIM_IOS_FLAGS = -arch $(NIM_IOS_ARCH) -isysroot $$(xcrun --sdk $(NIM_IOS_SDK) --show-sdk-path) -m$(if $(filter iphonesimulator,$(NIM_IOS_SDK)),ios-simulator,iphoneos)-version-min=$(or $(IOS_TARGET),13.0)
NIM_FLAGS_IOS = -d:useMalloc --cpu:$(if $(filter x86_64,$(NIM_IOS_ARCH)),amd64,arm64) --os:ios --cc:clang --passC:\"$(NIM_IOS_FLAGS)\" --passL:\"$(NIM_IOS_FLAGS)\"

NIM_FLAGS = $(NIM_FLAGS_DESKTOP)

# Runs a task from a library's .nims entry, so it builds from this resolution
# and not from its own lock. nimble shell puts the selected Nim on PATH.
# nimble.paths has every --path, so parent configs (status-app's) are skipped.
NIM_TASK = env -u SDKROOT -u CPATH -u LIBRARY_PATH \
	LIB_OUT_DIR="$(NIM_LIB_DIR)" $(if $(NIM_LIB_EXT),LIB_EXT=$(NIM_LIB_EXT)) \
	NIM_PARAMS="--skipParentCfg $(NIM_POLICY) $(NIM_FLAGS) $$NIM_PARAMS $$(tr '\n' ' ' < nimble.paths)" \
	$(NIMBLE) shell nim --skipParentCfg
