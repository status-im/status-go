//go:build gowaku_rln
// +build gowaku_rln

package node

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	r "github.com/status-im/go-rln/rln"
	"github.com/status-im/go-waku/waku/v2/protocol/rln"
)

func WithStaticRLNRelay(pubsubTopic string, contentTopic string, memberIndex r.MembershipIndex, spamHandler rln.SpamHandler) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRLN = true
		params.rlnRelayDynamic = false
		params.rlnRelayMemIndex = memberIndex
		params.rlnRelayPubsubTopic = pubsubTopic
		params.rlnRelayContentTopic = contentTopic
		params.rlnSpamHandler = spamHandler
		return nil
	}
}

func WithDynamicRLNRelay(pubsubTopic string, contentTopic string, memberIndex r.MembershipIndex, idKey *r.IDKey, idCommitment *r.IDCommitment, spamHandler rln.SpamHandler, ethClientAddress string, ethPrivateKey *ecdsa.PrivateKey, membershipContractAddress common.Address) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRLN = true
		params.rlnRelayDynamic = true
		params.rlnRelayMemIndex = memberIndex
		params.rlnRelayIDKey = idKey
		params.rlnRelayIDCommitment = idCommitment
		params.rlnRelayPubsubTopic = pubsubTopic
		params.rlnRelayContentTopic = contentTopic
		params.rlnSpamHandler = spamHandler
		params.rlnETHClientAddress = ethClientAddress
		params.rlnETHPrivateKey = ethPrivateKey
		params.rlnMembershipContractAddress = membershipContractAddress
		return nil
	}
}
