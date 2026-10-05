//go:build !tkl

package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNimSelectionRequiresBuildTag(t *testing.T) {
	_, err := selectTokenListsManager(&Manager{}, []uint64{1}, time.Time{}, time.Hour, time.Minute, true)
	require.ErrorContains(t, err, "tkl tag")
}
