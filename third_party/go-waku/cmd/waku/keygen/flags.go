package keygen

import (
	cli "github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

// Options contain the settings used for generating a key file
var Options GenerateKeyOptions

var (
	// KeyFile is a flag that contains the path where the node key will be written
	KeyFile = altsrc.NewPathFlag(&cli.PathFlag{
		Name:        "key-file",
		Value:       "./nodekey",
		Usage:       "Path to a file containing the private key for the P2P node",
		Destination: &Options.KeyFile,
		EnvVars:     []string{"WAKUNODE2_KEY_FILE"},
	})
	// KeyPassword is a flag to set the password used to encrypt the file
	KeyPassword = altsrc.NewStringFlag(&cli.StringFlag{
		Name:        "key-password",
		Value:       "secret",
		Usage:       "Password used for the private key file",
		Destination: &Options.KeyPasswd,
		EnvVars:     []string{"WAKUNODE2_KEY_PASSWORD"},
	})
	// Overwrite is a flag used to overwrite an existing key file
	Overwrite = altsrc.NewBoolFlag(&cli.BoolFlag{
		Name:        "overwrite",
		Value:       false,
		Usage:       "Overwrite the nodekey file if it already exists",
		Destination: &Options.Overwrite,
	})
)
