# Build status-go in a Go builder container
FROM golang:1.26.5-trixie AS builder

# Set environment variables to use Clang
ENV CC=clang
ENV CXX=clang++

RUN apt-get update \
    && apt-get install -y git bash make cmake llvm clang build-essential pkg-config curl xz-utils jq unzip \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install protoc from pre-built binaries
ARG PROTOC_VERSION=36.0
RUN set -eu && \
    DPKG_ARCH="$(dpkg --print-architecture)" && \
    case "$DPKG_ARCH" in \
    amd64) PROTOC_ARCH="x86_64" ;; \
    arm64) PROTOC_ARCH="aarch_64" ;; \
    *) echo "Unsupported architecture: $DPKG_ARCH" >&2; exit 1 ;; \
    esac && \
    curl -sSfL "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${PROTOC_ARCH}.zip" -o /tmp/protoc.zip && \
    unzip -q /tmp/protoc.zip -d /usr/local bin/protoc 'include/*' && \
    rm /tmp/protoc.zip && \
    protoc --version

# Install Nim from pre-built binaries
ARG NIM_VERSION=2.2.4
RUN set -eu && \
    DPKG_ARCH="$(dpkg --print-architecture)" && \
    case "$DPKG_ARCH" in \
    amd64) NIM_ARCH="x64" ;; \
    arm64) NIM_ARCH="arm64" ;; \
    *) echo "Unsupported architecture: $DPKG_ARCH" >&2; exit 1 ;; \
    esac && \
    NIM_URL=$(curl -sSf https://nim-lang.org/releases.json \
    | jq -r --arg ver "$NIM_VERSION" --arg arch "$NIM_ARCH" \
    '.[$ver]["linux_" + $arch].github_url // empty') && \
    if [ -z "$NIM_URL" ]; then \
    echo "ERROR: No download URL found for Nim $NIM_VERSION linux_$NIM_ARCH" >&2; \
    exit 1; \
    fi && \
    curl -sSfL "$NIM_URL" -o /tmp/nim.tar.xz && \
    mkdir -p /opt/nim && \
    tar -xJf /tmp/nim.tar.xz -C /opt/nim --strip-components=1 && \
    rm /tmp/nim.tar.xz && \
    /opt/nim/bin/nim --version

ENV PATH="/opt/nim/bin:${PATH}"

# The nimble bundled with Nim cannot resolve nim-sds through the bindings.
ARG NIMBLE_VERSION=0.24.1
RUN set -eu && \
    case "$(dpkg --print-architecture)" in \
    amd64) NIMBLE_ARCH="linux_x64" ;; \
    arm64) NIMBLE_ARCH="linux_aarch64" ;; \
    *) echo "ERROR: unsupported architecture" >&2; exit 1 ;; \
    esac; \
    curl -sSfL -o /tmp/nimble.tar.gz \
    "https://github.com/nim-lang/nimble/releases/download/v${NIMBLE_VERSION}/nimble-${NIMBLE_ARCH}.tar.gz" && \
    tar -xzf /tmp/nimble.tar.gz -C /opt/nim/bin && \
    rm /tmp/nimble.tar.gz && \
    chmod +x /opt/nim/bin/nimble && \
    nimble --version

ARG build_tags='gowaku_no_rln'
ARG build_flags=''
ARG build_target='cmd'
ARG use_logos_storage='false'

RUN mkdir -p /go/src/github.com/status-im/status-go
WORKDIR /go/src/github.com/status-im/status-go

ADD go.mod go.sum ./
RUN go mod download

ADD . .
ARG cache_id='local'
ARG enable_go_cache=true

RUN if [ "$enable_go_cache" = "true" ]; then \
    go env -w GOCACHE=/root/.cache/go-build; \
    fi
ARG use_torrent=false
RUN --mount=type=cache,target="/root/.cache/go-build",id=statusgo-build-$cache_id \
    make $build_target BUILD_TAGS="$build_tags" BUILD_FLAGS="$build_flags" USE_LOGOS_STORAGE="$use_logos_storage" USE_TORRENT="$use_torrent"

# Stage runtime shared libraries required by built binaries.
RUN mkdir -p /tmp/status-runtime-libs \
    && cp /go/src/github.com/status-im/status-go/build/libsds.so /tmp/status-runtime-libs/ \
    && if [ -f /go/src/github.com/status-im/logos-storage-nim/build/libstorage.so ]; then \
    cp /go/src/github.com/status-im/logos-storage-nim/build/libstorage.so /tmp/status-runtime-libs/; \
    fi

# Copy binaries to the second image
FROM debian:trixie-slim

LABEL maintainer="support@status.im"
LABEL source="https://github.com/status-im/status-go"
LABEL description="status-go is an underlying part of Status - a browser, messenger, and gateway to a decentralized world."

RUN apt-get update \
 && apt-get install -y ca-certificates bash curl python3 libgomp1 \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /usr/status-user && chmod -R 777 /usr/status-user
RUN mkdir -p /static/configs

COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/status-backend /usr/local/bin/
COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/push-notification-server /usr/local/bin/
COPY --from=builder /go/src/github.com/status-im/status-go/test/functional/waku_configs/* /static/configs/
COPY --from=builder /tmp/status-runtime-libs/ /usr/local/lib/

ENV LD_LIBRARY_PATH=/usr/local/lib/

EXPOSE 8080 8545 30303 30303/udp 30304/udp

CMD ["status-backend", "--help"]
