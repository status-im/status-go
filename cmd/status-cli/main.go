package main

import (
	"os"
	"time"

	"go.uber.org/zap"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/protocol"

	"github.com/urfave/cli/v2"
)

const LightFlag = "light"
const InteractiveFlag = "interactive"
const CountFlag = "count"
const NameFlag = "name"
const AddFlag = "add"
const PortFlag = "port"
const APIModulesFlag = "api-modules"
const TelemetryServerURLFlag = "telemetry-server-url"
const KeyUIDFlag = "key-uid"

const RetrieveInterval = 300 * time.Millisecond
const SendInterval = 1 * time.Second
const WaitingInterval = 5 * time.Second

var CommonFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    LightFlag,
		Aliases: []string{"l"},
		Usage:   "Enable light mode",
	},
	&cli.StringFlag{
		Name:    APIModulesFlag,
		Aliases: []string{"m"},
		Value:   "waku,wakuext,wakuv2,permissions,eth",
		Usage:   "API modules to enable",
	},
	&cli.StringFlag{
		Name:    TelemetryServerURLFlag,
		Aliases: []string{"t"},
		Usage:   "Telemetry server URL",
	},
}

var SimulateFlags = append([]cli.Flag{
	&cli.BoolFlag{
		Name:    InteractiveFlag,
		Aliases: []string{"i"},
		Usage:   "Use interactive mode to input the messages",
	},
	&cli.IntFlag{
		Name:    CountFlag,
		Aliases: []string{"c"},
		Value:   1,
		Usage:   "How many messages to sent from each user",
	},
}, CommonFlags...)

var ServeFlags = append([]cli.Flag{
	&cli.StringFlag{
		Name:    NameFlag,
		Aliases: []string{"n"},
		Value:   "Alice",
		Usage:   "Name of the user",
	},
	&cli.StringFlag{
		Name:    AddFlag,
		Aliases: []string{"a"},
		Usage:   "Add a friend with the public key",
	},
	&cli.IntFlag{
		Name:    PortFlag,
		Aliases: []string{"p"},
		Value:   8545,
		Usage:   "HTTP Server port to listen on",
	},
	&cli.BoolFlag{
		Name:    InteractiveFlag,
		Aliases: []string{"i"},
		Usage:   "Use interactive mode to input the messages",
	},
}, CommonFlags...)

var logger *zap.SugaredLogger

type StatusCLI struct {
	name      string
	messenger *protocol.Messenger
	backend   *api.GethStatusBackend
	logger    *zap.SugaredLogger
}

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "simulate",
				Usage: "Simulate the process of sending direct messages",
				Flags: SimulateFlags,
				Action: func(cCtx *cli.Context) error {
					return simulate(cCtx)
				},
			},
			{
				Name:    "serve",
				Aliases: []string{"s"},
				Usage:   "Start a server to send and receive messages",
				Flags:   ServeFlags,
				Action: func(cCtx *cli.Context) error {
					return serve(cCtx, false)
				},
			},
			{
				Name:    "serve-account",
				Aliases: []string{"sl"},
				Usage:   "Start a server with the lastest input name's account\n\n  E.g.: if last time you created an account with name 'Alice',\n  you can start the server with 'Alice' account by running 'servelast -n Alice'",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     NameFlag,
						Aliases:  []string{"n"},
						Usage:    "Name of the existing user",
						Required: true,
					},
					&cli.StringFlag{
						Name:    KeyUIDFlag,
						Aliases: []string{"kid"},
						Usage:   "Key ID of the existing user (if not provided the last account will be used)",
					},
					&cli.BoolFlag{
						Name:    InteractiveFlag,
						Aliases: []string{"i"},
						Usage:   "Use interactive mode to input the messages",
					},
					&cli.StringFlag{
						Name:    AddFlag,
						Aliases: []string{"a"},
						Usage:   "Add a friend with the public key",
					},
				},
				Action: func(cCtx *cli.Context) error {
					return serve(cCtx, true)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
