package sentry

import (
	"os"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
)

func TestWithDSN(t *testing.T) {
	t.Parallel()

	dsn := "https://examplePublicKey@o0.ingest.sentry.io/0"
	option := WithDSN(dsn)
	cfg := &sentry.ClientOptions{}
	option(cfg)
	require.Equal(t, dsn, cfg.Dsn)
}

func TestWithEnvironmentDSN(t *testing.T) {
	t.Parallel()

	expectedDSN := gofakeit.LetterN(10)
	envVar := gofakeit.LetterN(10)

	err := os.Setenv(envVar, expectedDSN)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.Unsetenv(envVar)
		require.NoError(t, err)
	})

	option := WithEnvironmentDSN(envVar)
	cfg := &sentry.ClientOptions{}
	option(cfg)
	require.Equal(t, expectedDSN, cfg.Dsn)
}

func TestWithContext(t *testing.T) {
	t.Parallel()

	name := "test-context"
	version := "v1.0.0"
	option := WithContext(name, version)
	cfg := &sentry.ClientOptions{}
	option(cfg)
	require.Equal(t, name, cfg.Tags["context.name"])
	require.Equal(t, version, cfg.Tags["context.version"])
}

func TestApplyOptions(t *testing.T) {
	t.Parallel()

	dsn := gofakeit.LetterN(10)
	name := gofakeit.LetterN(10)
	version := gofakeit.LetterN(5)

	options := []Option{
		WithDSN(dsn),
		WithContext(name, version),
	}
	cfg := &sentry.ClientOptions{}
	applyOptions(cfg, options...)

	require.Equal(t, dsn, cfg.Dsn)
	require.Equal(t, name, cfg.Tags["context.name"])
	require.Equal(t, version, cfg.Tags["context.version"])
}
