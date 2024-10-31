# Build status-go in a Go builder container
FROM golang:1.21-alpine3.18 as builder

# Set environment variables to use Clang
ENV CC=clang
ENV CXX=clang++

RUN apk add --no-cache git bash make llvm clang musl-dev linux-headers protobuf-dev~3.21

ARG build_tags
ARG build_flags
ARG build_target=statusgo

RUN mkdir -p /go/src/github.com/status-im/status-go
WORKDIR /go/src/github.com/status-im/status-go
ADD . .

RUN go install go.uber.org/mock/mockgen@v0.4.0
RUN go install github.com/kevinburke/go-bindata/v4/...@v4.0.2
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.1
RUN make $build_target BUILD_TAGS="$build_tags" BUILD_FLAGS="$build_flags" SHELL="/bin/sh"

# Copy the binary to the second image
FROM alpine:3.18

ARG build_target=statusgo

LABEL maintainer="support@status.im"
LABEL source="https://github.com/status-im/status-go"
LABEL description="status-go is an underlying part of Status - a browser, messenger, and gateway to a decentralized world."

RUN apk add --no-cache ca-certificates bash libgcc libstdc++ curl
RUN mkdir -p /static/keys
RUN mkdir -p /static/configs

COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/$build_target /usr/local/bin/
COPY --from=builder /go/src/github.com/status-im/status-go/static/keys/* /static/keys/
COPY --from=builder /go/src/github.com/status-im/status-go/tests-functional/config.json /static/configs/

RUN ln -s /usr/local/bin/$build_target /usr/local/bin/entrypoint

# 30304 is used for Discovery v5
EXPOSE 8080 8545 30303 30303/udp 30304/udp

ENTRYPOINT ["/usr/local/bin/entrypoint"]
CMD ["--help"]
