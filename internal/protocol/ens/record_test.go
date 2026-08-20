package ens

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNextRetry(t *testing.T) {
	record := VerificationRecord{Name: "vitalik.eth"}
	record.VerifiedAt = 10
	record.CalculateNextRetry()

	var expectedNextRetry uint64 = 30 + 10
	require.Equal(t, expectedNextRetry, record.NextRetry)

	expectedNextRetry = 60 + 10
	record.VerificationRetries++
	record.CalculateNextRetry()
	require.Equal(t, expectedNextRetry, record.NextRetry)

	expectedNextRetry = 120 + 10
	record.VerificationRetries++
	record.CalculateNextRetry()
	require.Equal(t, expectedNextRetry, record.NextRetry)
}
