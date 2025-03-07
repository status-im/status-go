package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/status-im/status-go/services/wallet/token"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"

	"github.com/xeipuuv/gojsonschema"
)

const templateText = `package token

import (
	"github.com/ethereum/go-ethereum/common"
	tokenTypes "github.com/status-im/status-go/services/wallet/token/types"
)

var {{ .VersionName }} = "{{ .Version }}"

var {{ .TimestampName }} = int64({{ .Timestamp }})

var {{ .AllTokensName }} = []*tokenTypes.Token{
{{ range $token := .Tokens }}
	{
		Address:   common.HexToAddress("{{ $token.Address }}"),
		Name:      "{{ $token.Name }}",
		Symbol:    "{{ $token.Symbol }}",
		Decimals:  {{ $token.Decimals }},
		ChainID:   {{ $token.ChainID }},
		PegSymbol: "{{ $token.PegSymbol }}",
	},{{ end }}
}
`

type templateData struct {
	AllTokensName string
	VersionName   string
	TimestampName string
	Version       string
	Timestamp     uint64
	Tokens        []*tokenTypes.Token
}

type version struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Patch int `json:"patch"`
}
type details struct {
	Version   version `json:"version"`
	Timestamp string  `json:"timestamp"`
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

func bytesToTokens(tokenListData []byte) ([]*tokenTypes.Token, error) {
	var objmap map[string]json.RawMessage
	err := json.Unmarshal(tokenListData, &objmap)
	if err != nil {
		return nil, err
	}
	var tokens []*tokenTypes.Token
	err = json.Unmarshal(objmap["tokens"], &tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func getVersionAndTimestamp(data []byte) (version string, timestamp uint64, err error) {
	var details details
	err = json.Unmarshal(data, &details)
	if err != nil {
		fmt.Printf("Failed to unmarshal version and timestamp: %v\n", err)
		return
	}

	time, err := time.Parse(time.RFC3339, details.Timestamp)
	if err != nil {
		fmt.Printf("Failed to parse timestamp: %v\n", err)
		return
	}

	version = fmt.Sprintf("%d.%d.%d", details.Version.Major, details.Version.Minor, details.Version.Patch)
	timestamp = uint64(time.Unix())
	return
}

func main() {
	client := &http.Client{Timeout: time.Minute}

	for key, source := range token.TokensSources {
		downloadTokens(client, key, source)
	}
}

func downloadTokens(client *http.Client, key string, source token.TokensSource) {
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

	tokens, err := bytesToTokens(body)
	if err != nil {
		fmt.Printf("Failed to parse token list: %v\n", err)
		return
	}

	version, timestamp, err := getVersionAndTimestamp(body)
	if err != nil {
		fmt.Printf("Failed to parse version and time: %v\n", err)
	}

	data := templateData{
		AllTokensName: fmt.Sprintf("%sTokens", key),
		VersionName:   fmt.Sprintf("%sVersion", key),
		TimestampName: fmt.Sprintf("%sTimestamp", key),
		Version:       version,
		Timestamp:     timestamp,
		Tokens:        tokens,
	}

	tmpl := template.Must(template.New("tokens").Parse(templateText))

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
