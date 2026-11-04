//go:build gowaku_no_rln
// +build gowaku_no_rln

package rlngenerate

import (
	"errors"

	cli "github.com/urfave/cli/v2"
)

// Command generates a key file used to generate the node's peerID, encrypted with an optional password
var Command = cli.Command{
	Name:  "generate-rln-credentials",
	Usage: "Generate credentials for usage with RLN",
	Action: func(cCtx *cli.Context) error {
		return errors.New("not available. Execute `make RLN=true` to add RLN support to go-waku")
	},
}
