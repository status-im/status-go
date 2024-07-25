FROM golang:1.23rc2

WORKDIR /peerdiscovery
COPY . .
RUN go get github.com/schollz/progressbar/v2
RUN go build ./examples/ipv4/main.go

CMD ["/peerdiscovery/main"]
