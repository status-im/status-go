package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/pkg/backend"
	"github.com/status-im/status-go/pkg/backend/requests"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	logger := logutils.ZapLogger()
	rootDataDir := path.Join(dir, "data")

	fmt.Printf("Root data dir: %s\n", rootDataDir)

	b, err := backend.NewStatusBackend(
		rootDataDir,
		backend.WithLogger(logger),
	)

	if err != nil {
		fmt.Printf("Error creating b: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	accs, err := b.API().ListAccounts(ctx)
	if err != nil {
		fmt.Printf("Error listing accounts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Accounts: %+v\n", accs)

	request := &requests.CreateAccount{
		Password:          "",
		KdfIterations:     0,
		DeviceName:        "test-device",
		DisplayName:       "test-user",
		WakuV2LightClient: false,
	}
	acc, err := b.API().CreateAccount(ctx, request, nil)
	if err != nil {
		fmt.Printf("Error creating account: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Account created: %+v\n", acc)

	accs, err = b.API().ListAccounts(ctx)
	if err != nil {
		fmt.Printf("Error listing accounts: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Accounts: %+v\n", accs)
}
