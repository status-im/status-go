package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/xeipuuv/gojsonschema"

	defaulttokenlists "github.com/status-im/status-go/services/wallet/token/local-token-lists/default-lists"
)

const templateText = `package defaulttokenlists

import (
	"time"
)

func init() {
	{{ .TokenListName }}.ID = "{{ .TokenListIdentifier }}"
	{{ .TokenListName }}.SourceURL = "{{ .TokenListSource }}"
	{{ .TokenListName }}.Fetched = time.Unix({{ .FetchedTimestamp }}, 0)
	{{ .TokenListName }}.JsonData = {{ .JsonData }}
}
`

type templateData struct {
	TokenListName       string
	TokenListIdentifier string
	TokenListSource     string
	FetchedTimestamp    int64
	JsonData            string
}

func formatBytes(data []byte) string {
	var parts []string
	for _, b := range data {
		parts = append(parts, fmt.Sprintf("0x%02x", b))
	}
	return fmt.Sprintf("[]byte{%s}", strings.Join(parts, ", "))
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

func main() {
	client := &http.Client{Timeout: time.Minute}

	for key, source := range defaulttokenlists.TokensSources {
		downloadTokens(client, key, source)
	}
}

func downloadTokens(client *http.Client, key string, source defaulttokenlists.TokensSource) {
	response, err := client.Get(source.SourceURL)
	if err != nil {
		fmt.Printf("[%s] failed to fetch tokens: %v\n", key, err)
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("[%s] failed to read tokens: %v\n", key, err)
		return
	}

	// check if body is valid json
	var jsonData map[string]interface{}
	err = json.Unmarshal(body, &jsonData)
	if err != nil {
		fmt.Printf("ERR: [%s] failed to unmarshal body: %v\n", key, err)
		body = []byte{}
	}

	if source.Schema != "" && len(body) > 0 {
		_, err = validateDocument(string(body), source.Schema)
		if err != nil {
			fmt.Printf("ERR: [%s] failed to validate token list against schema: %v\n", key, err)
			body = []byte{}
		}
	}

	capitalizedFirstLetter := func(s string) string {
		if len(s) == 0 {
			return s
		}
		return fmt.Sprintf("%s%s", strings.ToUpper(string(s[0])), s[1:])
	}

	data := templateData{
		TokenListName:       capitalizedFirstLetter(fmt.Sprintf("%sTokenList", key)),
		TokenListIdentifier: key,
		TokenListSource:     source.SourceURL,
		FetchedTimestamp:    time.Now().Unix(),
		JsonData:            formatBytes(body),
	}

	tmpl := template.Must(template.New("tokenList").Parse(templateText))

	// Create the output Go file
	file, err := os.Create(source.OutputFile)
	if err != nil {
		fmt.Printf("ERR: [%s] failed to create go file: %v\n", key, err)
		return
	}
	defer file.Close()

	// Execute the template with the tokens data and write the result to the file
	err = tmpl.Execute(file, data)
	if err != nil {
		fmt.Printf("ERR: [%s] failed to write file: %v\n", key, err)
		return
	}

	if len(body) > 0 {
		fmt.Printf("INFO: [%s] downloaded tokens successfully\n", key)
	} else {
		fmt.Printf("WARN: [%s] stored with empty token list\n", key)
	}
}
