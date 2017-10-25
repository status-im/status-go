package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/status-im/status-go/geth/common/cipher"
	"github.com/status-im/status-go/geth/params"
	"gopkg.in/urfave/cli.v1"
)

/* Examples:
go run main.go --input "../keys/wnodekey" --output="../keys/wnodekey.cr" --key="1234567891234567" encrypt
go run main.go --input "../keys/wnodekey.cr" --output="../keys/wnodekey.crd" --key="1234567891234567" decrypt
*/
func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	app = cli.NewApp()

	keyFlag = cli.StringFlag{
		Name:   "key",
		EnvVar: "STATUS_KEY",
		Usage:  "AES-128 or AES-256",
	}

	nonceFlag = cli.StringFlag{
		Name:   "nonce",
		EnvVar: "STATUS_NONCE",
		Value:  params.DataDir,
	}

	inputFileFlag = cli.StringFlag{
		Name:   "input",
		EnvVar: "INPUT_FILE",
		Usage:  "path to input file",
	}

	outputFileFlag = cli.StringFlag{
		Name:   "output",
		EnvVar: "OUTPUT_FILE",
		Usage:  "path to output file",
	}
)

var (
	encryptCommand = cli.Command{
		Action: EncryptCommandHandler,
		Name:   "encrypt",
		Usage:  "Print app version",
	}
	decryptCommand = cli.Command{
		Action: DecryptCommandHandler,
		Name:   "decrypt",
		Usage:  "Print app version",
	}
)

// EncryptCommandHandler handles encrypt command.
func EncryptCommandHandler(ctx *cli.Context) error {
	text, err := ioutil.ReadFile(ctx.GlobalString(inputFileFlag.Name))
	if err != nil {
		return err
	}

	cipherText, err := cipher.Encrypt(
		ctx.GlobalString(keyFlag.Name),
		ctx.GlobalString(nonceFlag.Name),
		text)
	if err != nil {
		return err
	}

	outputFile := ctx.GlobalString(outputFileFlag.Name)
	if outputFile == "" {
		outputFile = "/dev/stdout"
	}

	err = ioutil.WriteFile(outputFile, cipherText, 0644)
	if err != nil {
		return err
	}

	return nil
}

// DecryptCommandHandler handles decrypt command.
func DecryptCommandHandler(ctx *cli.Context) error {
	text, err := ioutil.ReadFile(ctx.GlobalString(inputFileFlag.Name))
	if err != nil {
		return err
	}

	cipherText, err := cipher.Decrypt(
		ctx.GlobalString(keyFlag.Name),
		ctx.GlobalString(nonceFlag.Name),
		text)
	if err != nil {
		return err
	}

	outputFile := ctx.GlobalString(outputFileFlag.Name)
	if outputFile == "" {
		outputFile = "/dev/stdout"
	}

	err = ioutil.WriteFile(outputFile, cipherText, 0644)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	// setup the app
	app.Action = cli.ShowAppHelp
	app.HideVersion = true
	app.Commands = []cli.Command{
		encryptCommand,
		decryptCommand,
	}
	app.Flags = []cli.Flag{
		keyFlag,
		nonceFlag,
		inputFileFlag,
		outputFileFlag,
	}
}
