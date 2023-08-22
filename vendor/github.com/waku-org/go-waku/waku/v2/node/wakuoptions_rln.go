//go:build gowaku_rln
// +build gowaku_rln

package node

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln"
	r "github.com/waku-org/go-zerokit-rln/rln"
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

// WithDynamicRLNRelay enables the Waku V2 RLN protocol in onchain mode.
// Requires the `gowaku_rln` build constrain (or the env variable RLN=true if building go-waku)
func WithDynamicRLNRelay(pubsubTopic string, contentTopic string, keystorePath string, keystorePassword string, keystoreIndex uint, treePath string, membershipContract common.Address, membershipGroupIndex uint, spamHandler rln.SpamHandler, ethClientAddress string, ethPrivateKey *ecdsa.PrivateKey, registrationHandler rln.RegistrationHandler) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRLN = true
		params.rlnRelayDynamic = true
		params.keystorePassword = keystorePassword
		params.keystorePath = keystorePath
		params.keystoreIndex = keystoreIndex
		params.rlnRelayPubsubTopic = pubsubTopic
		params.rlnRelayContentTopic = contentTopic
		params.rlnSpamHandler = spamHandler
		params.rlnETHClientAddress = ethClientAddress
		params.rlnETHPrivateKey = ethPrivateKey
		params.rlnMembershipContractAddress = membershipContract
		params.rlnRegistrationHandler = registrationHandler
		params.rlnRelayMemIndex = membershipGroupIndex
		params.rlnTreePath = treePath
		return nil
	}
}
