package pb

//go:generate protoc -I./../../pb/. -I. --go_opt=paths=source_relative --go_opt=Mwaku_lightpush.proto=github.com/waku-org/go-waku/waku/v2/protocol/lightpush/pb --go_opt=Mwaku_message.proto=github.com/waku-org/go-waku/waku/v2/protocol/pb  --go_out=. ./waku_lightpush.proto
