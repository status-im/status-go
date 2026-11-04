package pb

//go:generate protoc -I. -I./../../waku-proto/ --go_opt=paths=source_relative --go_opt=Mstorev3.proto=github.com/waku-org/go-waku/waku/v2/protocol/store/pb --go_opt=Mwaku/message/v1/message.proto=github.com/waku-org/go-waku/waku/v2/protocol/pb --go_out=. ./storev3.proto
