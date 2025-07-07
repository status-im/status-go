# Build status-go in a Go builder container
FROM golang:1.22.12-bookworm AS builder

# Set environment variables to use Clang
ENV CC=clang
ENV CXX=clang++

RUN apt-get update \
    && apt-get install -y git bash make llvm clang protobuf-compiler build-essential pkg-config iproute2 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

ARG build_tags='gowaku_no_rln'
ARG build_flags=''
ARG build_target='cmd'

RUN mkdir -p /go/src/github.com/status-im/status-go
WORKDIR /go/src/github.com/status-im/status-go

ADD go.mod go.sum ./
RUN go mod download -x
RUN go install go.uber.org/mock/mockgen@v0.4.0
RUN go install github.com/kevinburke/go-bindata/v4/...@v4.0.2
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1

ADD . .
ARG cache_id='local'
ARG enable_go_cache=true

RUN if [ "$enable_go_cache" = "true" ]; then \
      go env -w GOCACHE=/root/.cache/go-build; \
    fi
RUN --mount=type=cache,target="/root/.cache/go-build",id=statusgo-build-$cache_id \
    --mount=type=cache,target="/root/.cache/go-generate-fast",id=statusgo-build-$cache_id \
    make $build_target BUILD_TAGS="$build_tags" BUILD_FLAGS="$build_flags" \
      GO_GENERATE_FAST_RECACHE=$([ "$enable_go_cache" = "true" ] && echo false || echo true) \
      GO_GENERATE_FAST_DIR="/root/.cache/go-generate-fast"

# Copy binaries to the second image
FROM debian:bookworm-slim

LABEL maintainer="support@status.im"
LABEL source="https://github.com/status-im/status-go"
LABEL description="status-go is an underlying part of Status - a browser, messenger, and gateway to a decentralized world."

RUN apt-get update \
 && apt-get install -y ca-certificates bash curl \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /usr/status-user && chmod -R 777 /usr/status-user
RUN mkdir -p /static/keys
RUN mkdir -p /static/configs

COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/status-backend /usr/local/bin/
COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/push-notification-server /usr/local/bin/
COPY --from=builder /go/src/github.com/status-im/status-go/static/keys/* /static/keys/
COPY --from=builder /go/src/github.com/status-im/status-go/tests-functional/config.json /static/configs/
COPY --from=builder /go/src/github.com/status-im/status-go/tests-functional/waku_configs/* /static/configs/

# 30304 is used for Discovery v5
EXPOSE 8080 8545 30303 30303/udp 30304/udp

ENTRYPOINT ["status-backend"]
CMD ["--help"]
