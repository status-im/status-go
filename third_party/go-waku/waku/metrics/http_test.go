package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

func TestStartAndStopMetricsServer(t *testing.T) {
	server := NewMetricsServer("0.0.0.0", 9876, utils.Logger())

	go func() {
		time.Sleep(1 * time.Second)
		err := server.Stop(context.Background())
		require.NoError(t, err)
	}()

	server.Start()
}
