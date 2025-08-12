package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/multiaccounts"
	"github.com/status-im/status-go/walletdatabase"
)

const (
	password            = ""
	kdfIterationsNumber = 0
)

func main() {
	outDir := flag.String("out-dir", "build", "Output directory for generated DB files")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("failed to create output directory %s: %v", *outDir, err)
	}

	if err := generateAppDB(*outDir); err != nil {
		log.Fatalf("failed to generate app DB: %v", err)
	}
	fmt.Println("Generated app DB")

	if err := generateWalletDB(*outDir); err != nil {
		log.Fatalf("failed to generate wallet DB: %v", err)
	}
	fmt.Println("Generated wallet DB")

	if err := generateAccountsDB(*outDir); err != nil {
		log.Fatalf("failed to generate accounts DB: %v", err)
	}
	fmt.Println("Generated accounts DB")

	fmt.Printf("All DBs are generated under %s\n", *outDir)
}

func recreate(path string) error {
	_ = os.Remove(path)
	_ = os.Remove(path + "-wal")
	_ = os.Remove(path + "-shm")
	return nil
}

func generateAppDB(outDir string) error {
	path := filepath.Join(outDir, "app.db")
	if err := recreate(path); err != nil {
		return err
	}

	// Use the same initializer the app uses
	var init appdatabase.DbInitializer
	_, err := init.Initialize(path, password, kdfIterationsNumber)
	if err != nil {
		return err
	}
	return nil
}

func generateWalletDB(outDir string) error {
	path := filepath.Join(outDir, "wallet.db")
	if err := recreate(path); err != nil {
		return err
	}

	// Use the same initializer the app uses
	var init walletdatabase.DbInitializer
	_, err := init.Initialize(path, password, kdfIterationsNumber)
	if err != nil {
		return err
	}
	return nil
}

func generateAccountsDB(outDir string) error {
	path := filepath.Join(outDir, "accounts.db")
	if err := recreate(path); err != nil {
		return err
	}

	// Accounts DB uses its own initializer returning a wrapper type
	db, err := multiaccounts.InitializeDB(path)
	if err != nil {
		return err
	}
	defer db.Close()
	return nil
}
