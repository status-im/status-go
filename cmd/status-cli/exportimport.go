package main

import (
	"fmt"
	"os"

	"github.com/status-im/status-go/api"
	"github.com/status-im/status-go/logutils"
	statusgo "github.com/status-im/status-go/mobile"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func exportUnencryptedDB(cCtx *cli.Context) error {
	name := cCtx.String(NameFlag)
	rootDataDir := cCtx.String(RootDataDirFlag)
	keyUID := cCtx.String(KeyUIDFlag)
	isDebugLevel := cCtx.Bool(DebugLevel)
	cmdName := cCtx.Command.Name
	output := cCtx.String(OutputFlag)
	password := cCtx.String(PasswordFlag)
	if password != "" {
		password = statusgo.Sha3(password)
	}

	logger, err := getSLogger(isDebugLevel)
	if err != nil {
		zap.S().Fatalf("Error initializing logger: %v", err)
	}
	logger.Infof("Running %v command, with:\n%v", cmdName, flagsUsed(cCtx))
	logger = logger.Named(name)
	setupLogger(name)

	if fileExists(output) {
		return fmt.Errorf("Output file already exists %s", output)
	}

	backend := api.NewGethStatusBackend(logutils.ZapLogger())
	cli := StatusCLI{
		name:    name,
		backend: backend,
		logger:  logger,
	}
	defer cli.stop()

	backend.UpdateRootDataDir(rootDataDir)
	if err := backend.OpenAccounts(); err != nil {
		return fmt.Errorf("name '%v' might not have an account: trying to find: %v: %w", name, rootDataDir, err)
	}
	accs, err := backend.GetAccounts()
	if err != nil {
		return err
	}
	if len(accs) == 0 {
		logger.Error("no accounts found")
	}

	found := false
	for _, a := range accs {
		if a.KeyUID == keyUID {
			found = true
			break
		}
	}
	if !found {
		logger.Errorf("account not found for keyUID: %v", keyUID)
	}

	err = backend.ExportUnencryptedDatabase(
		multiaccounts.Account{
			Name:   "foo",
			KeyUID: keyUID,
		},
		password,
		output,
	)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("Exiting")
	return nil
}

func importUnencryptedDB(cCtx *cli.Context) error {
	name := cCtx.String(NameFlag)
	rootDataDir := cCtx.String(RootDataDirFlag)
	keyUID := cCtx.String(KeyUIDFlag)
	isDebugLevel := cCtx.Bool(DebugLevel)
	cmdName := cCtx.Command.Name
	input := cCtx.String(InputFlag)
	password := cCtx.String(PasswordFlag)
	if password != "" {
		password = statusgo.Sha3(password)
	}

	logger, err := getSLogger(isDebugLevel)
	if err != nil {
		zap.S().Fatalf("Error initializing logger: %v", err)
	}
	logger.Infof("Running %v command, with:\n%v", cmdName, flagsUsed(cCtx))
	logger = logger.Named(name)
	setupLogger(name)

	if !fileExists(input) {
		return fmt.Errorf("Input file does not exist %s", input)
	}

	backend := api.NewGethStatusBackend(logutils.ZapLogger())
	cli := StatusCLI{
		name:    name,
		backend: backend,
		logger:  logger,
	}
	defer cli.stop()

	backend.UpdateRootDataDir(rootDataDir)
	if err := backend.OpenAccounts(); err != nil {
		return fmt.Errorf("name '%v' might not have an account: trying to find: %v: %w", name, rootDataDir, err)
	}
	accs, err := backend.GetAccounts()
	if err != nil {
		return err
	}
	if len(accs) == 0 {
		logger.Error("no accounts found")
	}

	found := false
	for _, a := range accs {
		if a.KeyUID == keyUID {
			found = true
			break
		}
	}
	if !found {
		logger.Errorf("account not found for keyUID: %v", keyUID)
	}

	err = backend.ImportUnencryptedDatabase(
		multiaccounts.Account{
			Name:   "foo",
			KeyUID: keyUID,
		},
		password,
		input,
	)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Info("Exiting")
	return nil
}
