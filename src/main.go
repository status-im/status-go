package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

var app *cli.App

func init() {

	app = cli.NewApp()
	app.Name = "statusgo"
	app.Usage = "status specific geth functionality/bindings"

	app.Commands = []cli.Command{
		{
			Action: createAccount,
			Name:   "createaccount",
			Usage:  "statusgo newaccount --password=badpassword --keydir=/path/to/directory",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "password",
					Usage: "password for creating or unlocking an account",
				},
				cli.StringFlag{
					Name:  "keydir",
					Usage: "directory to be used for the geth account keystore",
				},
			},
		},
	}

}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
