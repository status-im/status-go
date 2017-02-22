FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev && \

  # clone status-go
  mkdir -p /usr/lib/go/src/github.com/status-im && \
  git clone --depth 1 --branch develop https://github.com/status-im/status-go.git /usr/lib/go/src/github.com/status-im/status-go && \

  # clone go-ethereum (and install everything)
  git clone --depth 1 --branch status-master https://github.com/status-im/go-ethereum && \
  (cd go-ethereum && make geth) && \
  cp go-ethereum/build/bin/geth /geth && \
  apk del go git make gcc musl-dev && \
  rm -rf /go-ethereum && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/geth"]
