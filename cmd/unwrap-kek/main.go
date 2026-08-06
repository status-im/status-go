// Command unwrap-kek is a developer utility to unwrap a profile's wrapped-DEK file.
//
// Usage:
//
//	go run ./cmd/unwrap-kek <path/to/<keyUID>-profile.kek> <password>
//
// The password may be given as:
// - the raw profile password
// - in its hashed form, as the already-hashed 0x… value
// - the keycard encryption public key (both raw and hashed forms are tried)

package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/status-im/status-go/internal/accounts-management/keystore/envelope"
)

const (
	mainSuffix    = "-profile.kek"
	pendingSuffix = "-profile.kek.pending"
)

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <path/to/<keyUID>-profile.kek[.pending]> <password>\n", filepath.Base(os.Args[0]))
		os.Exit(2)
	}
	filePath, password := os.Args[1], os.Args[2]

	rootDataDir := filepath.Dir(filePath)
	base := filepath.Base(filePath)

	var keyUID string
	pending := false
	switch {
	case strings.HasSuffix(base, pendingSuffix):
		keyUID = strings.TrimSuffix(base, pendingSuffix)
		pending = true
	case strings.HasSuffix(base, mainSuffix):
		keyUID = strings.TrimSuffix(base, mainSuffix)
	default:
		fatalf("%s is not a %s file", base, mainSuffix)
	}

	if _, err := os.Stat(filePath); err != nil {
		fatalf("%v", err)
	}

	unwrap := envelope.Unwrap
	if pending {
		unwrap = envelope.UnwrapPending
	}

	// Whatever is sent as the password, we try to unwrap the envelope with it.
	hashed := "0x" + strings.ToLower(hex.EncodeToString(ethcrypto.Keccak256([]byte(password))))
	candidates := []struct{ label, kek string }{
		{"provided value used as-is", password},
		{"keccak256-hashed password (client convention)", hashed},
	}

	for _, candidate := range candidates {
		dekHex, dbKdfIterations, err := unwrap(rootDataDir, keyUID, candidate.kek)
		if err != nil {
			continue
		}
		fmt.Printf("KEK accepted:   %s\n", candidate.label)
		fmt.Printf("keyUID:         %s\n", keyUID)
		fmt.Printf("DEK (secret):   %s\n", dekHex)
		fmt.Printf("db kdf_iter:    %d\n", dbKdfIterations)
		fmt.Println()
		fmt.Println("The DEK is the passphrase for the profile databases and keystore files.")
		fmt.Println("To open a database with the sqlcipher CLI:")
		fmt.Printf("  PRAGMA key = '%s';\n", dekHex)
		fmt.Printf("  PRAGMA kdf_iter = %d;\n", dbKdfIterations)
		fmt.Println("  PRAGMA cipher_page_size = 8192;")
		fmt.Println("  PRAGMA cipher_hmac_algorithm = HMAC_SHA1;")
		fmt.Println("  PRAGMA cipher_kdf_algorithm = PBKDF2_HMAC_SHA1;")
		return
	}

	fatalf("the provided password does not unwrap this envelope (tried as-is and keccak256-hashed)")
}
