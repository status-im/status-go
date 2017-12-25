# Build wnode-status in a Go builder container
FROM golang:1.9-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

RUN mkdir -p /go/src/github.com/status-im/status-go
ADD . /go/src/github.com/status-im/status-go
RUN cd /go/src/github.com/status-im/status-go && make statusgo && make wnode-status

# Copy the binary to the second image
FROM alpine:latest

RUN apk add --no-cache ca-certificates bash
COPY --from=builder /go/src/github.com/status-im/status-go/build/bin/* /usr/local/bin/

RUN mkdir -p /static/keys
COPY --from=builder /go/src/github.com/status-im/status-go/static/keys/* /static/keys/

EXPOSE 8545 30303 30303/udp
# Discovery v5
EXPOSE 30304/udp
