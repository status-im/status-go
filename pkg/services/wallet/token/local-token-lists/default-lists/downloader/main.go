package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/xeipuuv/gojsonschema"

	defaulttokenlists "github.com/status-im/status-go/pkg/services/wallet/token/local-token-lists/default-lists"
)

const templateText = `package defaulttokenlists

import (
	_ "embed" // for go:embed
	"time"
)

//go:embed {{ .JSONFile }}
var {{ .JSONVar }} []byte

func init() {
	{{ .TokenListName }}.ID = "{{ .TokenListIdentifier }}"
	{{ .TokenListName }}.SourceURL = "{{ .TokenListSource }}"
	{{ .TokenListName }}.Fetched = time.Unix({{ .FetchedTimestamp }}, 0)
	{{ .TokenListName }}.JsonData = {{ .JSONVar }}
}
`

type templateData struct {
	TokenListName       string
	TokenListIdentifier string
	TokenListSource     string
	FetchedTimestamp    int64
	JSONFile            string
	JSONVar             string
}

func validateDocument(doc string, schemaURL string) (bool, error) {
	schemaLoader := gojsonschema.NewReferenceLoader(schemaURL)
	docLoader := gojsonschema.NewStringLoader(doc)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return false, err
	}

	if !result.Valid() {
		return false, errors.New("Token list does not match schema")
	}

	return true, nil
}

var (
	osExit           = os.Exit
	createTempOutput = func(dir, pattern string) (string, io.WriteCloser, error) {
		file, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return "", nil, err
		}
		return file.Name(), file, nil
	}
	tokenSources = defaulttokenlists.TokensSources
)

func main() {
	client := &http.Client{Timeout: time.Minute}
	failed := false
	written := map[string]defaulttokenlists.TokensSource{}

	for key, source := range tokenSources {
		if err := downloadTokens(client, key, source); err != nil {
			fmt.Fprintf(os.Stderr, "ERR: [%s] %v\n", key, err)
			failed = true
			continue
		}
		written[key] = source
	}
	if err := writeManifests(written); err != nil {
		fmt.Fprintf(os.Stderr, "ERR: %v\n", err)
		failed = true
	}

	if failed {
		osExit(1)
	}
}

func downloadTokens(client *http.Client, key string, source defaulttokenlists.TokensSource) error {
	response, err := client.Get(source.SourceURL)
	if err != nil {
		return fmt.Errorf("failed to fetch tokens: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read tokens: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", response.Status)
	}

	var jsonData map[string]interface{}
	if err = json.Unmarshal(body, &jsonData); err != nil {
		return fmt.Errorf("failed to unmarshal body: %w", err)
	}

	if source.Schema != "" {
		if _, err = validateDocument(string(body), source.Schema); err != nil {
			return fmt.Errorf("failed to validate token list against schema: %w", err)
		}
	}

	if err = writeTokenList(key, source.SourceURL, source.OutputFile, body, time.Now()); err != nil {
		return err
	}

	fmt.Printf("INFO: [%s] downloaded tokens successfully\n", key)
	return nil
}

// writeTokenList writes the list as a sibling .json embedded by a generated .go file, so it stays in the
// binary's data section, off the heap. The .go goes last; if it fails, a .json that did not exist before is removed.
func writeTokenList(key, sourceURL, outputFile string, body []byte, fetched time.Time) error {
	jsonPath := strings.TrimSuffix(outputFile, filepath.Ext(outputFile)) + ".json"
	tokenListName := strings.ToUpper(key[:1]) + key[1:] + "TokenList"
	data := templateData{
		TokenListName:       tokenListName,
		TokenListIdentifier: key,
		TokenListSource:     sourceURL,
		FetchedTimestamp:    fetched.Unix(),
		JSONFile:            filepath.Base(jsonPath),
		JSONVar:             strings.ToLower(tokenListName[:1]) + tokenListName[1:] + "JSON",
	}
	var goSource bytes.Buffer
	if err := template.Must(template.New("tokenList").Parse(templateText)).Execute(&goSource, data); err != nil {
		return fmt.Errorf("failed to render go file: %w", err)
	}

	_, statErr := os.Stat(jsonPath)
	jsonExisted := statErr == nil
	if err := writeGeneratedFile(jsonPath, "json", body); err != nil {
		return err
	}
	if err := writeGeneratedFile(outputFile, "go", goSource.Bytes()); err != nil {
		if !jsonExisted {
			_ = os.Remove(jsonPath)
		}
		return err
	}
	return nil
}

// writeManifests regenerates SHA256SUMS in every directory holding one of the given lists.
func writeManifests(sources map[string]defaulttokenlists.TokensSource) error {
	dirs := map[string]bool{}
	for _, source := range sources {
		dirs[filepath.Dir(source.OutputFile)] = true
	}
	for dir := range dirs {
		if err := writeManifest(dir); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath.Join(dir, "SHA256SUMS"), err)
		}
	}
	return nil
}

// writeManifest records the sha256 of every embedded .json in dir, in `shasum -a 256` format.
func writeManifest(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	var manifest bytes.Buffer
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		fmt.Fprintf(&manifest, "%s  %s\n", hex.EncodeToString(sum[:]), filepath.Base(f))
	}
	return writeGeneratedFile(filepath.Join(dir, "SHA256SUMS"), "manifest", manifest.Bytes())
}

// writeGeneratedFile replaces path atomically (temp file + rename), keeping an existing file's mode.
func writeGeneratedFile(path, kind string, content []byte) error {
	tmpName, file, err := createTempOutput(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("failed to create %s file: %w", kind, err)
	}

	closed := false
	renamed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if !renamed {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = file.Write(content); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	if err = file.Close(); err != nil {
		closed = true
		return fmt.Errorf("failed to write file: %w", err)
	}
	closed = true

	mode, err := generatedFileMode(path, kind)
	if err != nil {
		return err
	}
	if err = os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("failed to set %s file mode: %w", kind, err)
	}

	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to replace %s file: %w", kind, err)
	}
	renamed = true
	return nil
}

func generatedFileMode(path, kind string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.Mode().Perm(), nil
	}
	if !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to stat %s file: %w", kind, err)
	}
	return 0o644, nil
}
