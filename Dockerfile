# Build status-go in a Go builder container
# glibc≥2.39 (to match libstorage requirements)
FROM golang:1.24.7-trixie AS builder

# Set environment variables to use Clang
ENV CC=clang
ENV CXX=clang++

RUN apt-get update \
    && apt-get install -y git bash make llvm clang protobuf-compiler build-essential pkg-config curl xz-utils unzip \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Nim using choosenim
ARG NIM_VERSION=2.2.4
ENV CHOOSENIM_NO_ANALYTICS=1
RUN curl https://nim-lang.org/choosenim/init.sh -sSf | sh -s -- -y \
    && /root/.nimble/bin/choosenim ${NIM_VERSION}

ENV PATH="/root/.nimble/bin:${PATH}"

# Create system-wide symlinks for Nim binaries
RUN ln -sf /root/.choosenim/toolchains/nim-${NIM_VERSION}/bin/nim /usr/local/bin/nim \
    && ln -sf /root/.choosenim/toolchains/nim-${NIM_VERSION}/bin/nimble /usr/local/bin/nimble \
    && ln -sf /root/.choosenim/toolchains/nim-${NIM_VERSION}/bin/choosenim /usr/local/bin/choosenim \
    && chmod 755 /root/.choosenim/toolchains/nim-${NIM_VERSION}/bin/* \
    && nim --version

ARG build_tags='gowaku_no_rln'
ARG build_flags=''
ARG build_target='cmd'
ARG use_logos_storage='false'

RUN mkdir -p /go/src/github.com/status-im/status-go
WORKDIR /go/src/github.com/status-im/status-go

ADD go.mod go.sum ./
RUN go mod download
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1

ADD . .
ARG cache_id='local'
ARG enable_go_cache=true

RUN if [ "$enable_go_cache" = "true" ]; then \
    go env -w GOCACHE=/root/.cache/go-build; \
    fi
RUN --mount=type=cache,target="/root/.cache/go-build",id=statusgo-build-$cache_id \
    make $build_target BUILD_TAGS="$build_tags" BUILD_FLAGS="$build_flags" USE_LOGOS_STORAGE="$use_logos_storage"

# Stage runtime shared libraries required by built binaries.
RUN mkdir -p /tmp/status-runtime-libs \
    && cp /go/src/github.com/status-im/nim-sds/build/libsds.so /tmp/status-runtime-libs/ \
    && if [ -f /go/src/github.com/status-im/status-go/libs/libstorage.so ]; then cp /go/src/github.com/status-im/status-go/libs/libstorage.so /tmp/status-runtime-libs/; fi

# Copy binaries to the second image
# glibc≥2.39
FROM debian:trixie-slim

LABEL maintainer="support@status.im"
LABEL source="https://github.com/status-im/status-go"
LABEL description="status-go is an underlying part of Status - a browser, messenger, and gateway to a decentralized world."

RUN apt-get update \
    && apt-get install -y ca-certificates bash curl python3 libstdc++6 libgomp1 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /usr/status-user && chmod -R 777 /usr/status-user
RUN mkdir -p /static/configs

COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/status-backend /usr/local/bin/
COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/push-notification-server /usr/local/bin/
COPY --from=builder /go/src/github.com/status-im/status-go/tests-functional/waku_configs/* /static/configs/
COPY --from=builder /tmp/status-runtime-libs/ /usr/local/lib/

ENV LD_LIBRARY_PATH=/usr/local/lib/

RUN mkdir -p /go/src/github.com/status-im/status-go/libs
COPY --from=builder /go/src/github.com/status-im/status-go/libs/* /go/src/github.com/status-im/status-go/libs/

EXPOSE 8080 8545 30303 30303/udp 30304/udp

CMD ["status-backend", "--help"]
