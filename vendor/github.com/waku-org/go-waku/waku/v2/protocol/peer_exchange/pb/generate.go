package pb

//go:generate protoc -I. --go_opt=paths=source_relative --go_opt=Mwaku_peer_exchange.proto=github.com/waku-org/go-waku/waku/v2/protocol/peer_exchange/pb --go_out=. ./waku_peer_exchange.proto
