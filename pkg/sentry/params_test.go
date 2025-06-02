package sentry

import (
	"os"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

func setEnvVar(t *testing.T, varName, value string) {
	err := os.Setenv(varName, value)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = os.Unsetenv(varName)
		require.NoError(t, err)
	})
}

func TestEnvironment(t *testing.T) {
	t.Parallel()

	t.Run("returns production environment when production is true", func(t *testing.T) {
		varName := gofakeit.LetterN(10)
		setEnvVar(t, varName, "development")

		// Expect production although the env variable is set to development
		result := environment(true, varName)
		require.Equal(t, productionEnvironment, result)
	})

	t.Run("returns empty string when env is productionEnvironment", func(t *testing.T) {
		varName := gofakeit.LetterN(10)
		setEnvVar(t, varName, productionEnvironment)

		result := environment(false, varName)
		require.Equal(t, "", result)
	})

	t.Run("returns environment variable when production is false", func(t *testing.T) {
		varName := gofakeit.LetterN(10)
		expectedEnv := "development"
		setEnvVar(t, varName, expectedEnv)

		result := environment(false, varName)
		require.Equal(t, expectedEnv, result)
	})
}
