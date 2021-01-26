package wallet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCryptoOnRamps_Get(t *testing.T) {
	cs := []*CryptoOnRampManager{
		{options: &CryptoOnRampOptions{dataSourceType: DataSourceStatic}},
		{options: &CryptoOnRampOptions{
			dataSourceType: DataSourceHTTP,
			dataSource:     cryptoOnRampsData,
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
	corm := NewCryptoOnRampManager(&CryptoOnRampOptions{
		dataSourceType: DataSourceHTTP,
		dataSource:     cryptoOnRampsData,
	})
	nt := time.Time{}.Add(30 * time.Minute)

	require.False(t, corm.hasCacheExpired(nt))
	require.True(t, corm.hasCacheExpired(time.Now()))

	_, err := corm.Get()
	require.NoError(t, err)
	require.False(t, corm.hasCacheExpired(time.Now()))
	require.True(t, corm.hasCacheExpired(time.Now().Add(2*time.Hour)))
}
