package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/xeipuuv/gojsonschema"

	defaulttokenlists "github.com/status-im/status-go/services/wallet/token/token-lists/default-lists"
)

const templateText = `package defaulttokenlists

import (
	"time"

	"github.com/status-im/status-go/services/wallet/token/token-lists/fetcher"
)

var {{ .TokenListName }} = fetcher.FetchedTokenList{
	TokenList: fetcher.TokenList{
		ID:        "{{ .TokenListIdentifier }}",
		SourceURL: "{{ .TokenListSource }}",
	},
	Fetched: time.Unix({{ .FetchedTimestamp }}, 0),
	JsonData: {{ .JsonData }},
}
`

type templateData struct {
	TokenListName       string
	TokenListIdentifier string
	TokenListSource     string
	FetchedTimestamp    int64
	JsonData            string
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
		fmt.Printf("Failed to fetch tokens: %v\n", err)
		return
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Printf("Failed to read tokens: %v\n", err)
		return
	}

	if source.Schema != "" {
		_, err = validateDocument(string(body), source.Schema)
		if err != nil {
			fmt.Printf("Failed to validate token list against schema: %v\n", err)
			return
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
		JsonData:            fmt.Sprintf("`%s`", body),
	}

	tmpl := template.Must(template.New("tokenList").Parse(templateText))

	// Create the output Go file
	file, err := os.Create(source.OutputFile)
	if err != nil {
		fmt.Printf("Failed to create go file: %v\n", err)
		return
	}
	defer file.Close()

	// Execute the template with the tokens data and write the result to the file
	err = tmpl.Execute(file, data)
	if err != nil {
		fmt.Printf("Failed to write file: %v\n", err)
		return
	}
}
