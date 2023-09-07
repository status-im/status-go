//go:build gowaku_rln
// +build gowaku_rln

package node

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/waku-org/go-waku/waku/v2/protocol/rln"
	r "github.com/waku-org/go-zerokit-rln/rln"
)

// WithStaticRLNRelay enables the Waku V2 RLN protocol in offchain mode
// Requires the `gowaku_rln` build constrain (or the env variable RLN=true if building go-waku)
func WithStaticRLNRelay(memberIndex r.MembershipIndex, spamHandler rln.SpamHandler) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRLN = true
		params.rlnRelayDynamic = false
		params.rlnRelayMemIndex = memberIndex
		params.rlnSpamHandler = spamHandler
		return nil
	}
}

// WithDynamicRLNRelay enables the Waku V2 RLN protocol in onchain mode.
// Requires the `gowaku_rln` build constrain (or the env variable RLN=true if building go-waku)
func WithDynamicRLNRelay(keystorePath string, keystorePassword string, treePath string, membershipContract common.Address, membershipIndex uint, spamHandler rln.SpamHandler, ethClientAddress string) WakuNodeOption {
	return func(params *WakuNodeParameters) error {
		params.enableRLN = true
		params.rlnRelayDynamic = true
		params.keystorePassword = keystorePassword
		params.keystorePath = keystorePath
		params.rlnSpamHandler = spamHandler
		params.rlnETHClientAddress = ethClientAddress
		params.rlnMembershipContractAddress = membershipContract
		params.rlnRelayMemIndex = membershipIndex
		params.rlnTreePath = treePath
		return nil
	}
}
