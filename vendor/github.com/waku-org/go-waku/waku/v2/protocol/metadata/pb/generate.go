package pb

//go:generate protoc -I. --go_opt=paths=source_relative --go_opt=Mwaku_metadata.proto=github.com/waku-org/go-waku/waku/v2/protocol/metadata/pb --go_out=. ./waku_metadata.proto
