package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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

	for key, source := range tokenSources {
		if err := downloadTokens(client, key, source); err != nil {
			fmt.Fprintf(os.Stderr, "ERR: [%s] %v\n", key, err)
			failed = true
		}
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

	capitalizedFirstLetter := func(s string) string {
		if len(s) == 0 {
			return s
		}
		return fmt.Sprintf("%s%s", strings.ToUpper(string(s[0])), s[1:])
	}

	// The list is embedded from a sibling .json file so it stays in the binary's data section, off the heap.
	jsonPath := strings.TrimSuffix(source.OutputFile, filepath.Ext(source.OutputFile)) + ".json"
	if err := os.WriteFile(jsonPath, body, 0o600); err != nil {
		return fmt.Errorf("failed to write json file: %w", err)
	}

	tokenListName := capitalizedFirstLetter(fmt.Sprintf("%sTokenList", key))
	data := templateData{
		TokenListName:       tokenListName,
		TokenListIdentifier: key,
		TokenListSource:     source.SourceURL,
		FetchedTimestamp:    time.Now().Unix(),
		JSONFile:            filepath.Base(jsonPath),
		JSONVar:             strings.ToLower(tokenListName[:1]) + tokenListName[1:] + "JSON",
	}

	tmpl := template.Must(template.New("tokenList").Parse(templateText))

	if err = writeGeneratedFile(source.OutputFile, tmpl, data); err != nil {
		return err
	}

	fmt.Printf("INFO: [%s] downloaded tokens successfully\n", key)
	return nil
}

func writeGeneratedFile(path string, tmpl *template.Template, data templateData) error {
	tmpName, file, err := createTempOutput(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("failed to create go file: %w", err)
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

	if err = tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	if err = file.Close(); err != nil {
		closed = true
		return fmt.Errorf("failed to write file: %w", err)
	}
	closed = true

	mode, err := generatedFileMode(path)
	if err != nil {
		return err
	}
	if err = os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("failed to set go file mode: %w", err)
	}

	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("failed to replace go file: %w", err)
	}
	renamed = true
	return nil
}

func generatedFileMode(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.Mode().Perm(), nil
	}
	if !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to stat go file: %w", err)
	}
	return 0o644, nil
}
