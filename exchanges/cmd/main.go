package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/status-im/status-go/exchanges/cmd/scaffold"
)

const (
	templateFile   = "cmd/exchanges.txt" // must include the cmd prefix because this code is called from the Makefile
	outputGoFile   = "exchanges.go"
	outputJsonFile = "exchanges.json"

	baseUrl          = "https://sapi.coincarp.com/api"
	walletUrl        = baseUrl + "/v1/market/walletscreen/coin/wallet"
	walletAddressUrl = baseUrl + "/v1/market/walletscreen/coin/walletaddress"
	iconBaseUrl      = "https://s1.coincarp.com"

	ethereumCode     = "ethereum"
	mainnetChainType = ""
	initialPageSize  = 30

	maxRetries = 10

	requestWaitTime = 1000 * time.Millisecond
	requestTimeout  = 5 * time.Second

	zeroAddress = "0x0000000000000000000000000000000000000000"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	var fromJsonFile string

	flag.StringVar(&fromJsonFile, "from-json-file", "", "Path to JSON file to use instead of remote source")
	flag.Parse()

	var exchangesData []exchangeData
	if fromJsonFile == "" {
		log.Println("Fetching from external service...")
		exchangesData = getExchangesData()
	} else {
		log.Println("Fetching from JSON file...")
		exchangesData = loadExchangesDataFromJson(fromJsonFile)
	}

	log.Println("Generating files...")
	for _, gen := range generators {
		gen(exchangesData)
	}
}

func doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:96.0) Gecko/20100101 Firefox/96.0")

	// Ensure wait time between requests
	time.Sleep(requestWaitTime)

	client := http.Client{
		Timeout: requestTimeout,
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	statusCode := res.StatusCode
	if statusCode != http.StatusOK {
		err := fmt.Errorf("unsuccessful request: %s - %d %s", url, statusCode, http.StatusText(statusCode))
		log.Fatal(err)
	}

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	var dataInfo scaffold.DataInfo
	if err = json.Unmarshal(b, &dataInfo); err != nil {
		fmt.Println("unmarshall error: ", url)
		log.Fatal(err)
	}

	if dataInfo.Code != http.StatusOK {
		err := fmt.Errorf("inconsistent response: %s - %d %s", url, dataInfo.Code, dataInfo.Msg)
		return nil, err
	}

	return b, nil
}

func getExchangesData() []exchangeData {
	log.Println("Fetching exchanges list...")
	exchanges, err := getLatestExchangeList()
	if err != nil {
		log.Fatalf("could not get list of exchanges: %v", err)
	}

	exchangesData := make([]exchangeData, 0, 128)
	for _, exchange := range exchanges {
		log.Println("Fetching address list for exchange:", exchange.Name)
		addresses, err := getLatestExchangeAddresses(exchange.Code)
		if err != nil {
			log.Fatalf("could not get list of addresses: %v", err)
		}
		exchangeData := buildExchangeData(exchange, addresses)
		exchangesData = append(exchangesData, exchangeData)
	}

	if len(exchangesData) == 0 {
		log.Fatalf("could not build exchanges list")
	}

	return exchangesData
}

func loadExchangesDataFromJson(filePath string) []exchangeData {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	var data []exchangeData
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Fatalf("cannot unmarshal data: %v", err)
	}

	return data
}

func getLatestExchangeList() ([]*scaffold.Exchange, error) {
	page := 1
	pageSize := initialPageSize
	retries := 0
	exchanges := make([]*scaffold.Exchange, 0, 128)

	for {
		queryParams := url.Values{
			"code":       {ethereumCode},
			"chainType":  {mainnetChainType},
			"page":       {strconv.Itoa(page)},
			"pageSize":   {strconv.Itoa(pageSize)},
			"isexchange": {"false"},
			"lang":       {"en-US"},
		}

		url := walletUrl + "?" + queryParams.Encode()

		b, err := doRequest(url)
		if err != nil {
			fmt.Println("request error:", err)
			if retries < maxRetries {
				page = 1
				pageSize++
				retries++
				exchanges = nil
				fmt.Println("retry", retries)
				continue
			}
			log.Fatal(err)
		}

		var data scaffold.ExchangesData
		if err = json.Unmarshal(b, &data); err != nil {
			fmt.Println("unmarshall error: ", url)
			log.Fatal(err)
		}

		exchanges = append(exchanges, data.Entries.List...)

		if page >= data.Entries.TotalPages {
			break
		}

		page++
	}

	return exchanges, nil
}

func getLatestExchangeAddresses(exchangeCode string) ([]*scaffold.ExchangeAddress, error) {
	page := 1
	pageSize := initialPageSize
	retries := 0
	addresses := make([]*scaffold.ExchangeAddress, 0, 128)

	for {
		queryParams := url.Values{
			"code":         {ethereumCode},
			"exchangecode": {exchangeCode},
			"chainType":    {mainnetChainType},
			"page":         {strconv.Itoa(page)},
			"pageSize":     {strconv.Itoa(pageSize)},
			"lang":         {"en-US"},
		}

		url := walletAddressUrl + "?" + queryParams.Encode()

		b, err := doRequest(url)
		if err != nil {
			fmt.Println("request error:", err)
			if retries < maxRetries {
				page = 1
				pageSize++
				retries++
				addresses = nil
				fmt.Println("retry", retries)
				continue
			}
			log.Fatal(err)
		}

		var data scaffold.ExchangeAddressesData
		if err = json.Unmarshal(b, &data); err != nil {
			fmt.Println("unmarshall error: ", url)
			log.Fatal(err)
		}

		addresses = append(addresses, data.Entries.List...)

		if page >= data.Entries.TotalPages {
			break
		}

		page++
	}

	return addresses, nil
}

type exchangeData struct {
	Code      string           `json:"code"`
	Name      string           `json:"name"`
	Symbol    string           `json:"symbol"`
	Logo      string           `json:"logo"`
	Addresses []common.Address `json:"addresses"`
}

func buildExchangeData(exchange *scaffold.Exchange, addresses []*scaffold.ExchangeAddress) exchangeData {
	data := exchangeData{
		Code:      exchange.Code,
		Name:      exchange.Name,
		Symbol:    exchange.Symbol,
		Logo:      iconBaseUrl + exchange.Logo,
		Addresses: []common.Address{},
	}

	for _, exchangeAddress := range addresses {
		address := common.HexToAddress(exchangeAddress.Address)
		if address.Hex() == zeroAddress {
			continue
		}
		data.Addresses = append(data.Addresses, address)
	}

	return data
}

type generatorFunc func(exchangesData []exchangeData)

var generators = []generatorFunc{
	generateJsonFile,
	generateGoPackage,
}

func generateJsonFile(exchangesData []exchangeData) {
	file, err := json.MarshalIndent(exchangesData, "", " ")
	if err != nil {
		log.Fatalf("cannot marshal data: %v", err)
	}

	err = ioutil.WriteFile(outputJsonFile, file, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func toVariableName(input string) string {
	return "exchange_" + strings.ToLower(strings.Replace(input, "-", "_", -1))
}

func addressesJoin(addresses []common.Address) string {
	list := make([]string, 0, len(addresses))

	for _, address := range addresses {
		list = append(list, "common.HexToAddress(\""+address.String()+"\")")
	}

	return strings.Join(list, ", ")
}

func generateGoPackage(exchangesData []exchangeData) {
	tpl, err := ioutil.ReadFile(templateFile)
	if err != nil {
		log.Fatalf("cannot open template file: %v", err)
	}

	funcMap := template.FuncMap{
		"toVariableName": toVariableName,
		"addressesJoin":  addressesJoin,
	}

	t := template.Must(template.New("go").Funcs(funcMap).Parse(string(tpl)))
	buf := new(bytes.Buffer)
	err = t.Execute(buf, exchangesData)
	if err != nil {
		log.Fatal(err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatal(err)
	}
	buf = bytes.NewBuffer(formatted)

	to, err := os.Create(outputGoFile)
	if err != nil {
		log.Fatal(err)
	}
	defer to.Close()

	_, err = io.Copy(to, buf)
	if err != nil {
		log.Fatal(err)
	}
}
