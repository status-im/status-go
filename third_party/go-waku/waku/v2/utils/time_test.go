package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestGetUnixEpochFrom(t *testing.T) {
	loc := time.UTC
	timestamp := GetUnixEpochFrom(time.Date(2019, 1, 1, 0, 0, 0, 0, loc))
	require.Equal(t, proto.Int64(int64(1546300800*time.Second)), timestamp)

	timestamp = GetUnixEpoch()
	require.NotNil(t, timestamp)
}
