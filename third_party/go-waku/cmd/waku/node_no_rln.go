//go:build gowaku_no_rln
// +build gowaku_no_rln

package main

import (
	"github.com/waku-org/go-waku/waku/v2/node"
	"go.uber.org/zap"
)

func checkForRLN(logger *zap.Logger, options NodeOptions, nodeOpts *[]node.WakuNodeOption) error {
	// Do nothing
	return nil
}
