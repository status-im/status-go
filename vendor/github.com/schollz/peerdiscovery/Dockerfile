FROM golang:1.22rc2-bookworm

WORKDIR /peerdiscovery
COPY . .
RUN go get github.com/schollz/progressbar/v2
RUN go build ./examples/ipv4/main.go

CMD ["/peerdiscovery/main"]
