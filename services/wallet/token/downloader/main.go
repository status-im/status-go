package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/status-im/status-go/services/wallet/token"

	"github.com/xeipuuv/gojsonschema"
)

const uniswapTokensURL = "https://gateway.ipfs.io/ipns/tokens.uniswap.org" // nolint:gosec
const tokenListSchemaURL = "https://uniswap.org/tokenlist.schema.json"     // nolint:gosec

const templateText = `
package token

import (
	"github.com/ethereum/go-ethereum/common"
)

var uniswapTokens = []*Token{
	{{ range $token := .Tokens }}
	{
		Address:   common.HexToAddress("{{ $token.Address }}"),
		Name:      "{{ $token.Name }}",
		Symbol:    "{{ $token.Symbol }}",
		Decimals:  {{ $token.Decimals }},
		ChainID:   {{ $token.ChainID }},
		PegSymbol: "{{ $token.PegSymbol }}",
	},
	{{ end }}
}
`

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

func bytesToTokens(tokenListData []byte) ([]*token.Token, error) {
	var objmap map[string]json.RawMessage
	err := json.Unmarshal(tokenListData, &objmap)
	if err != nil {
		return nil, err
	}
	var tokens []*token.Token
	err = json.Unmarshal(objmap["tokens"], &tokens)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func main() {
	client := &http.Client{Timeout: time.Minute}
	response, err := client.Get(uniswapTokensURL)
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

	_, err = validateDocument(string(body), tokenListSchemaURL)
	if err != nil {
		fmt.Printf("Failed to validate token list against schema: %v\n", err)
		return
	}

	tokens, err := bytesToTokens(body)
	if err != nil {
		fmt.Printf("Failed to parse token list: %v\n", err)
		return
	}

	tmpl := template.Must(template.New("tokens").Parse(templateText))

	// Create the output Go file
	file, err := os.Create("uniswap.go")
	if err != nil {
		fmt.Printf("Failed to create go file: %v\n", err)
		return
	}
	defer file.Close()

	// Execute the template with the tokens data and write the result to the file
	err = tmpl.Execute(file, struct{ Tokens []*token.Token }{Tokens: tokens})
	if err != nil {
		fmt.Printf("Failed to write file: %v\n", err)
		return
	}
}
