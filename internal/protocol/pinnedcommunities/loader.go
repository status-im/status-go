package pinnedcommunities

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/status-im/status-go/internal/protocol/pinnedcommunities/assets"
)

const RawPayloadHexSuffix = ".rawpayload.hex"

type Payload struct {
	CommunityID string
	RawPayload  []byte
	FileName    string
}

// LoadEmbedded returns pinned communities shipped inside the binary.
func LoadEmbedded() ([]Payload, error) {
	return loadFromFS(assets.FS, ".")
}

// LoadFromDir reads all *.rawpayload.hex files and returns decoded payloads.
// Files are returned in lexical filename order for deterministic processing.
func LoadFromDir(dir string) ([]Payload, error) {
	return loadFromFS(os.DirFS(dir), ".")
}

func loadFromFS(fsys fs.FS, dir string) ([]Payload, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("read pinned communities dir: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, RawPayloadHexSuffix) {
			continue
		}
		files = append(files, name)
	}

	sort.Strings(files)

	payloads := make([]Payload, 0, len(files))
	for _, name := range files {
		communityID := strings.TrimSuffix(name, RawPayloadHexSuffix)
		if communityID == "" {
			return nil, fmt.Errorf("invalid pinned community filename %q: empty community id", name)
		}

		rawHex, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read pinned community payload %q: %w", name, err)
		}

		hexString := strings.TrimSpace(string(rawHex))
		if hexString == "" {
			return nil, fmt.Errorf("invalid pinned community payload %q: empty content", name)
		}

		decoded, err := hex.DecodeString(hexString)
		if err != nil {
			return nil, fmt.Errorf("decode pinned community payload %q: %w", name, err)
		}

		payloads = append(payloads, Payload{
			CommunityID: communityID,
			RawPayload:  decoded,
			FileName:    name,
		})
	}

	return payloads, nil
}
