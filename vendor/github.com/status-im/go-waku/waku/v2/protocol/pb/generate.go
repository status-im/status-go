package pb

//go:generate protoc -I. --gofast_out=. ./waku_filter.proto
//go:generate protoc -I. --gofast_out=. ./waku_lightpush.proto
//go:generate protoc -I. --gofast_out=. ./waku_message.proto
//go:generate protoc -I. --gofast_out=. ./waku_store.proto
//go:generate protoc -I. --gofast_out=. ./waku_swap.proto
