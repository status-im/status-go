package fetcher

import (
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/t/helpers"
	"github.com/status-im/status-go/walletdatabase"
)

func SetupTestWalletDB(t *testing.T) (*sql.DB, func()) {
	db, cleanup, err := helpers.SetupTestSQLDB(walletdatabase.DbInitializer{}, "wallet-tests")
	require.NoError(t, err)
	return db, func() { require.NoError(t, cleanup()) }
}

func GetTestServer() (server *httptest.Server, close func()) {
	mux := http.NewServeMux()
	server = httptest.NewServer(mux)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		resp := strings.ReplaceAll(listOfTokenListsJsonResponse, serverURLPlaceholder, server.URL)
		if _, err := w.Write([]byte(resp)); err != nil {
			log.Println(err.Error())
		}
	})

	mux.HandleFunc("/uniswap.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(uniswapTokenListJsonResponse)); err != nil {
			log.Println(err.Error())
		}
	})

	mux.HandleFunc("/aave.json", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(aaveTokenListJsonResponse)); err != nil {
			log.Println(err.Error())
		}
	})

	return server, server.Close
}
