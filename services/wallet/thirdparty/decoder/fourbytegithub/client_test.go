package fourbytegithub

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, err := w.Write([]byte("transfer(address,uint256)"))
		if err != nil {
			return
		}
	}))
	defer srv.Close()

	fb := NewClient()
	fb.Client = srv.Client()
	fb.URL = srv.URL

	res, err := fb.Run("0xa9059cbb000000000000000000000000e0e40d81121d41a7d85d8d2462b475074f9df5ec0000000000000000000000000000000000000000000000000000000077359400")
	require.Nil(t, err)
	require.Equal(t, res.Signature, "transfer(address,uint256)")
	require.Equal(t, res.ID, "0xa9059cbb")
	require.Equal(t, res.Name, "transfer")
	require.Equal(t, res.Inputs, map[string]string{
		"0": "0x3030303030303030303030306530653430643831",
		"1": "22252012820881184517742036120632151212095838186768864961872069019727748752739",
	})
}
