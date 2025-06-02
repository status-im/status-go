package params

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadFleetsFromFile tests the loadFleetsFromFile function.
func TestLoadFleetsFromFile(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		// Arrange: Create a temporary valid JSON file.
		tempFile, err := os.CreateTemp("", "fleets*.json")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		// Write valid JSON data into the temp file.
		fleetName := gofakeit.LetterN(10)
		validFleets := FleetsMap{
			fleetName: {
				ClusterID: 123,
				WakuNodes: []string{"test.node"},
			},
		}
		err = json.NewEncoder(tempFile).Encode(validFleets)
		require.NoError(t, err)
		err = tempFile.Close()
		require.NoError(t, err)

		// Act: Call the function.
		result, err := loadFleetsFromFile(tempFile.Name())

		// Assert: Check the error.
		require.NoError(t, err)
		assert.Equal(t, validFleets, result)
	})

	t.Run("missing file", func(t *testing.T) {
		// Arrange: Use a non-existent file path.
		nonExistentFile := filepath.Join(os.TempDir(), "non_existent_file.json")

		// Act: Call the function.
		_, err := loadFleetsFromFile(nonExistentFile)

		// Assert: Check the error.
		require.Error(t, err)
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		// Arrange: Create a temporary file with invalid JSON.
		tempFile, err := os.CreateTemp("", "malformed*.json")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		_, err = tempFile.WriteString("{ invalid json }")
		require.NoError(t, err)
		tempFile.Close()

		// Act: Call the function.
		_, err = loadFleetsFromFile(tempFile.Name())

		// Assert: Check the error.
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid character") // Match JSON parsing error.
	})
}
