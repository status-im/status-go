package main

import "C"
import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/status-im/status-go/geth/console"

	"gopkg.in/urfave/cli.v1"
)

var (
	consoleCommand = cli.Command{
		Action: consoleCommandHandler,
		Name:   "console",
		Usage:  "Starts an interactive console to call bindings",
		Flags:  []cli.Flag{},
	}

	bindings = map[string]interface{}{
		"GenerateConfig":        GenerateConfig,
		"StartNode":             StartNode,
		"StopNode":              StopNode,
		"ValidateNodeConfig":    ValidateNodeConfig,
		"ResetChainData":        ResetChainData,
		"CallRPC":               CallRPC,
		"CreateAccount":         CreateAccount,
		"CreateChildAccount":    CreateChildAccount,
		"RecoverAccount":        RecoverAccount,
		"VerifyAccountPassword": VerifyAccountPassword,
		"Login":                 Login,
		"Logout":                Logout,
		"CompleteTransaction":   CompleteTransaction,
		"CompleteTransactions":  CompleteTransactions,
		"DiscardTransaction":    DiscardTransaction,
		"DiscardTransactions":   DiscardTransactions,
		"InitJail":              InitJail,
		"Parse":                 Parse,
		"Call":                  Call,
		"StartNodeWithConfig": func(datadir *C.char, networkID C.int, devMode C.int) *C.char {
			config := GenerateConfig(datadir, networkID, devMode)
			return StartNode(config)
		},
	}

	typeCCharPtr = reflect.TypeOf((*C.char)(nil))
	typeCInt     = reflect.TypeOf((C.int)(0))
)

// consoleCommandHandler handles statusd console command.
func consoleCommandHandler(ctx *cli.Context) error {
	var autocomplete []string
	for k := range bindings {
		autocomplete = append(autocomplete, k)
	}

	c, err := console.New(console.Config{
		Autocomplete: autocomplete,
	})
	if err != nil {
		fmt.Printf("Failed to start console: %s\n", err)
		os.Exit(1)
	}

	c.Welcome()

	// struct used to notify that a command was handled.
	commandHandler := make(chan string)
	commands := c.Interactive(commandHandler)

LOOP:
	for {
		command, ok := <-commands
		if !ok {
			break LOOP
		}

		// Get binding by string.
		binding, ok := bindings[command]
		if !ok {
			fmt.Printf("Command '%s' not found\n", command)
			commandHandler <- ""
			continue
		}

		// We will use reflection to dynamically call the binding.
		bindingValue := reflect.ValueOf(binding)
		bindingType := bindingValue.Type()
		argc := bindingType.NumIn()

		fmt.Printf("Calling binding: %s\n", bindingType)

		params := make([]reflect.Value, argc)
		for i := 0; i < argc; i++ {
			commandHandler <- "... "
			command, ok := <-commands
			if !ok {
				break LOOP
			}

			switch bindingType.In(i) {
			case typeCCharPtr:
				params[i] = reflect.ValueOf(C.CString(command))

			case typeCInt:
				v, err := strconv.Atoi(command)
				if err != nil {
					fmt.Printf("failed to parse to int: %s\n", err)
					commandHandler <- ""
					continue LOOP
				}

				params[i] = reflect.ValueOf(C.int(v))
			}
		}

		// Print all results. Each binding returns a result of type *C.char
		// or no result at all.
		result := bindingValue.Call(params)
		for _, r := range result {
			fmt.Println(C.GoString(r.Interface().(*C.char)))
		}

		// Notify that the command processing finished.
		commandHandler <- ""
	}

	if err := c.Stop(); err != nil {
		fmt.Printf("Failed to gracefully stop the console: %s\n", err)
	}

	return nil
}
