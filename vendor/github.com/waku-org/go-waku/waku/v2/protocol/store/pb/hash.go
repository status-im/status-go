package pb

import (
	pb "github.com/waku-org/go-waku/waku/v2/protocol/pb"
)

func (x *WakuMessageKeyValue) WakuMessageHash() pb.MessageHash {
	return pb.ToMessageHash(x.MessageHash)
}
