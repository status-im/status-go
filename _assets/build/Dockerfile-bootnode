FROM golang:1.10-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ARG build_tags
ARG build_flags

RUN mkdir -p /go/src/github.com/status-im/status-go
ADD . /go/src/github.com/status-im/status-go
RUN cd /go/src/github.com/status-im/status-go && \
    make bootnode \
    BUILD_TAGS="$build_tags" \
    BUILD_FLAGS="$build_flags"

FROM alpine:latest

LABEL maintainer="support@status.im"
LABEL source="https://github.com/status-im/status-go"
LABEL description="bootnode is an Ethereum bootnode with patched Discovery V5."

RUN apk add --no-cache ca-certificates bash

COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/bootnode /usr/local/bin/
ENTRYPOINT /usr/local/bin/bootnode
