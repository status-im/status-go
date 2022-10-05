//go:build gowaku_rln
// +build gowaku_rln

package node

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/go-waku/waku/v2/protocol/rln"
	r "github.com/status-im/go-zerokit-rln/rln"
)

// WithStaticRLNRelay enables the Waku V2 RLN protocol in offchain mode
// Requires the `gowaku_rln` build constrain (or the env variable RLN=true if building go-waku)
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

// WithStaticRLNRelay enables the Waku V2 RLN protocol in onchain mode.
// Requires the `gowaku_rln` build constrain (or the env variable RLN=true if building go-waku)
func WithDynamicRLNRelay(pubsubTopic string, contentTopic string, memberIndex r.MembershipIndex, idKey *r.IDKey, idCommitment *r.IDCommitment, spamHandler rln.SpamHandler, ethClientAddress string, ethPrivateKey *ecdsa.PrivateKey, membershipContractAddress common.Address, registrationHandler rln.RegistrationHandler) WakuNodeOption {
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
		params.rlnRegistrationHandler = registrationHandler
		return nil
	}
}
