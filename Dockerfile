# Build status-go in a Go builder container
FROM golang:1.24.7-bookworm AS builder

# Set environment variables to use Clang
ENV CC=clang
ENV CXX=clang++

RUN apt-get update \
    && apt-get install -y git bash make llvm clang protobuf-compiler build-essential pkg-config \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

ARG build_tags='gowaku_no_rln'
ARG build_flags=''
ARG build_target='cmd'

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
    make $build_target BUILD_TAGS="$build_tags" BUILD_FLAGS="$build_flags"

# Copy binaries to the second image
FROM debian:bookworm-slim

LABEL maintainer="support@status.im"
LABEL source="https://github.com/status-im/status-go"
LABEL description="status-go is an underlying part of Status - a browser, messenger, and gateway to a decentralized world."

RUN apt-get update \
 && apt-get install -y ca-certificates bash curl python3 \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /usr/status-user && chmod -R 777 /usr/status-user
RUN mkdir -p /static/configs

COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/status-backend /usr/local/bin/
COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/push-notification-server /usr/local/bin/
COPY --from=builder /go/src/github.com/status-im/status-go/tests-functional/waku_configs/* /static/configs/
COPY --from=builder /go/src/github.com/status-im/nim-sds/build/libsds.so /usr/local/lib/

ENV LD_LIBRARY_PATH=/usr/local/lib/

EXPOSE 8080 8545 30303 30303/udp 30304/udp

CMD ["status-backend", "--help"]