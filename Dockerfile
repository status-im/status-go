FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch feature/statusd-replaces-geth-on-cluster https://github.com/farazdagi/status-go && \
  (cd status-go && make) && \
  cp status-go/build/bin/statusd /statusd && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /status-go && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303
EXPOSE 3001

ENTRYPOINT ["/statusd"]
