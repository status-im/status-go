package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/status-im/status-go/cmd/lint-panics/utils"
)

func TestMethods(t *testing.T) {
	t.Parallel()

	logger := utils.BuildLogger(zap.DebugLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	a, err := New(ctx, logger)
	require.NoError(t, err)

	analysistest.Run(t, analysistest.TestData(), a, "functions")
}
