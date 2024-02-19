package wallet

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	path = "../../_assets/tests/"
)

func TestCryptoOnRamps_Get(t *testing.T) {
	s := httptest.NewServer(http.FileServer(http.Dir(path)))
	defer s.Close()

	cs := []*CryptoOnRampManager{
		{options: &CryptoOnRampOptions{dataSourceType: DataSourceStatic}},
		{options: &CryptoOnRampOptions{
			dataSourceType: DataSourceHTTP,
			dataSource:     s.URL + "/ramps.json", // nolint: goconst
		}},
	}

	for _, corm := range cs {
		require.Equal(t, 0, len(corm.ramps))

		rs, err := corm.Get()
		require.NoError(t, err)
		require.Greater(t, len(rs), 0)
	}
}

func TestCryptoOnRampManager_hasCacheExpired(t *testing.T) {
	s := httptest.NewServer(http.FileServer(http.Dir(path)))
	defer s.Close()

	corm := NewCryptoOnRampManager(&CryptoOnRampOptions{
		dataSourceType: DataSourceHTTP,
		dataSource:     s.URL + "/ramps.json", // nolint: goconst
	})
	nt := time.Time{}.Add(30 * time.Minute)

	require.False(t, corm.hasCacheExpired(nt))
	require.True(t, corm.hasCacheExpired(time.Now()))

	_, err := corm.Get()
	require.NoError(t, err)
	require.False(t, corm.hasCacheExpired(time.Now()))
	require.True(t, corm.hasCacheExpired(time.Now().Add(2*time.Hour)))
}
