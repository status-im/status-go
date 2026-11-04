SHELL := bash # the shell used internally by Make

.PHONY: all build run

ifeq ($(OS),Windows_NT)     # is Windows_NT on XP, 2000, 7, Vista, 10...
 detected_OS := Windows
else
 detected_OS := $(strip $(shell uname))
endif

UNAME_P := $(shell uname -p)
ifneq ($(filter arm%,$(UNAME_P)),)
		detected_arch := ARM
endif

all: build

ifeq ($(detected_OS),Linux)
 PLATFORM_FLAGS_TEST_C ?= -ldl
else ifeq ($(detected_OS),Darwin)
 PLATFORM_FLAGS_TEST_C ?= -Wl,-headerpad_max_install_names -framework CoreFoundation -framework Security -lresolv
 ifeq ($(detected_arch),ARM)
 PLATFORM_FLAGS_TEST_C +=-lresolv -framework CoreServices
 endif
endif

build:
	cd ../../ && $(MAKE) static-library # Building library
	rm -rf build/main && \
	echo "Compiling 'main.c'"
	+ mkdir -p build
	$(CC) \
		-I../../build/lib/ \
		main.c base64.c \
		../../build/lib/libgowaku.a \
		-lm \
		-pthread \
		$(PLATFORM_FLAGS_TEST_C) \
		-o build/main


run:
	echo "Executing './build/main.c'"
ifeq ($(detected_OS),macOS)
	./build/main
else ifeq ($(detected_OS),Windows)
	PATH="$(PATH_TEST)" \
	./build/main
else
	./build/main
endif


