# Toolchain stage: Go, clang, protoc, Nim, nimble. CI publishes it with --target toolchain.
FROM golang:1.26.5-trixie AS toolchain

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

# Keep the existing Nim toolchain for other dependencies. Token lists use the
# compiler tested by the pinned library's CI.
ARG TKL_NIM_VERSION=2.2.10
RUN set -eu && \
    case "$(dpkg --print-architecture)" in \
    amd64) TKL_NIM_ARCH=x64 ;; \
    arm64) TKL_NIM_ARCH=arm64 ;; \
    *) echo "Unsupported token compiler architecture" >&2; exit 1 ;; \
    esac && \
    TKL_NIM_URL=$(curl -sSf https://nim-lang.org/releases.json \
    | jq -er --arg ver "$TKL_NIM_VERSION" --arg arch "$TKL_NIM_ARCH" \
    '.[$ver]["linux_" + $arch].github_url') && \
    curl -sSfL "$TKL_NIM_URL" -o /tmp/tkl-nim.tar.xz && \
    mkdir -p /opt/nim-tkl && \
    tar -xJf /tmp/tkl-nim.tar.xz -C /opt/nim-tkl --strip-components=1 && \
    rm /tmp/tkl-nim.tar.xz && \
    /opt/nim-tkl/bin/nim --version
ENV TKL_NIM=/opt/nim-tkl/bin/nim

# Nimble builds nim-sds with the Nim installed below; the one above builds logos-storage.
# .github/actions/versions parses the FROM line and the ARG *_VERSION lines of this stage.
ARG NIMBLE_VERSION=0.26.1
RUN set -eu && \
    case "$(dpkg --print-architecture)" in \
    amd64) NIMBLE_ARCH="linux_x64" ;; \
    arm64) NIMBLE_ARCH="linux_aarch64" ;; \
    *) echo "ERROR: unsupported architecture" >&2; exit 1 ;; \
    esac; \
    mkdir -p /opt/nimble/bin && \
    curl -sSfL -o /tmp/nimble.tar.gz \
    "https://github.com/nim-lang/nimble/releases/download/v${NIMBLE_VERSION}/nimble-${NIMBLE_ARCH}.tar.gz" && \
    tar -xzf /tmp/nimble.tar.gz -C /opt/nimble/bin && \
    rm /tmp/nimble.tar.gz && \
    chmod +x /opt/nimble/bin/nimble

ENV PATH="/opt/nimble/bin:${PATH}"

# The Nim status_go.nimble requires for libsds, left where nimble looks first; arm64 has no published build.
ARG SDS_NIM_VERSION=2.2.6
ARG SDS_NIM_X64_SHA256=38b8407f87d78bd207390051e4c76f38a45d0a26983cb262017c899b56ad8d06
ARG SDS_NIM_SOURCE_SHA256=657b0e3d5def788148d2a87fa6123fa755b2d92cad31ef60fd261e451785528b
RUN set -eu && \
    mkdir -p /opt/nim-sds /root/.nimble/nimbinaries && \
    case "$(dpkg --print-architecture)" in \
    amd64) \
    SDS_NIM_URL=$(curl -sSf https://nim-lang.org/releases.json \
    | jq -er --arg ver "$SDS_NIM_VERSION" '.[$ver].linux_x64.github_url') && \
    curl -sSfL "$SDS_NIM_URL" -o /tmp/nim.tar.xz && \
    echo "${SDS_NIM_X64_SHA256}  /tmp/nim.tar.xz" | sha256sum -c - && \
    tar -xJf /tmp/nim.tar.xz -C /opt/nim-sds --strip-components=1 ;; \
    arm64) \
    curl -sSfL "https://nim-lang.org/download/nim-${SDS_NIM_VERSION}.tar.xz" -o /tmp/nim.tar.xz && \
    echo "${SDS_NIM_SOURCE_SHA256}  /tmp/nim.tar.xz" | sha256sum -c - && \
    tar -xJf /tmp/nim.tar.xz -C /opt/nim-sds --strip-components=1 && \
    cd /opt/nim-sds && \
    sh build.sh > /dev/null && \
    bin/nim c --noNimblePath --skipUserCfg --skipParentCfg --hints:off koch && \
    ./koch boot -d:release --skipUserCfg --skipParentCfg --hints:off && \
    rm -rf c_code nimcache compiler/nimcache koch ;; \
    *) echo "ERROR: unsupported architecture" >&2; exit 1 ;; \
    esac && \
    rm /tmp/nim.tar.xz && \
    /opt/nim-sds/bin/nim --version && \
    ln -s /opt/nim-sds "/root/.nimble/nimbinaries/nim-${SDS_NIM_VERSION}"
# nimble writes its metadata next to the compiler, as the calling user.
RUN chmod a+w /opt/nim-sds

# Test runner behind `make test-unit`
ARG GOTESTSUM_VERSION=1.13.0
RUN go install "gotest.tools/gotestsum@v${GOTESTSUM_VERSION}" && \
    go clean -cache -modcache && \
    gotestsum --version

# Build status-go in the toolchain container
FROM toolchain AS builder

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
