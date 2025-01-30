//go:build use_nwaku
// +build use_nwaku

package wakuv2

import (
	"github.com/waku-org/go-waku/waku/v2/protocol/pb"
	"github.com/waku-org/waku-go-bindings/waku/common"
)

func HexToPbHash(hexHash common.MessageHash) (pb.MessageHash, error) {
	bytesHash, err := hexHash.Bytes()
	if err != nil {
		return pb.MessageHash{}, err
	}

	pbHash := pb.ToMessageHash(bytesHash)
	return pbHash, nil
}
