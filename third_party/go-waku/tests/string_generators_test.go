package tests

import (
	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/utils"
	"go.uber.org/zap"
	"strconv"
	"testing"
)

func TestStringGenerators(t *testing.T) {

	log := utils.Logger()

	// Generate string and print out to console
	for i := 0; i < 1000; i++ {
		x, err := GenerateRandomASCIIString(4097)
		require.NoError(t, err)
		log.Info("Generated random ASCII string", zap.String(strconv.Itoa(i), x))

		x, err = GenerateRandomUTF8String(4097)
		require.NoError(t, err)
		log.Info("Generated random UTF8 string", zap.String(strconv.Itoa(i), x))

		x, err = GenerateRandomJSONString(4099)
		require.NoError(t, err)
		log.Info("Generated random JSON string", zap.String(strconv.Itoa(i), x))

		x, err = GenerateRandomBase64String(1025)
		require.NoError(t, err)
		log.Info("Generated random Base64 string", zap.String(strconv.Itoa(i), x))

		x, err = GenerateRandomURLEncodedString(2049)
		require.NoError(t, err)
		log.Info("Generated random URL encoded string", zap.String(strconv.Itoa(i), x))

		x, err = GenerateRandomSQLInsert(4096)
		require.NoError(t, err)
		log.Info("Generated random SQL insert string", zap.String(strconv.Itoa(i), x))
	}

}
