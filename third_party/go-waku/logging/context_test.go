package logging

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func Test_EmptyContext(t *testing.T) {
	logger := From(context.Background())
	require.Nil(t, logger)
}

func Test_With(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	ctx := With(context.Background(), logger)
	require.Equal(t, logger, From(ctx))
}
