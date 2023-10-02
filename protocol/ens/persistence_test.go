package ens

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/appdatabase"
	"github.com/status-im/status-go/protocol/sqlite"
	"github.com/status-im/status-go/t/helpers"
)

func TestGetENSToBeVerified(t *testing.T) {
	pk := "1"
	name := "test.eth"
	updatedName := "test2.eth"

	db, err := helpers.SetupTestMemorySQLDB(appdatabase.DbInitializer{})
	require.NoError(t, err)
	err = sqlite.Migrate(db)
	require.NoError(t, err)

	persistence := NewPersistence(db)
	require.NotNil(t, persistence)

	record := VerificationRecord{Name: name, PublicKey: pk, Clock: 2}

	// We add a record, it should be nil
	response, err := persistence.AddRecord(record)
	require.NoError(t, err)
	require.Nil(t, response)

	// We add a record again, it should return the same record
	response, err = persistence.AddRecord(record)
	require.NoError(t, err)
	require.NotNil(t, response)

	require.False(t, response.Verified)
	require.Equal(t, record.Name, response.Name)
	require.Equal(t, record.PublicKey, response.PublicKey)

	// We add a record again, with a different clock value
	record.Clock++
	response, err = persistence.AddRecord(record)
	require.NoError(t, err)
	require.NotNil(t, response)

	require.False(t, response.Verified)
	require.Equal(t, record.Name, response.Name)
	require.Equal(t, record.PublicKey, response.PublicKey)

	// We add a record again, with a different name, but lower clock value
	record.Clock--
	record.Name = updatedName
	response, err = persistence.AddRecord(record)
	require.NoError(t, err)
	require.NotNil(t, response)

	require.False(t, response.Verified)
	require.Equal(t, name, response.Name)
	require.Equal(t, record.PublicKey, response.PublicKey)

	// We add a record again, with a different name and higher clock value
	record.Clock += 2
	record.Name = updatedName
	response, err = persistence.AddRecord(record)
	require.NoError(t, err)
	require.Nil(t, response)

	// update the record

	record.Verified = false
	record.VerificationRetries = 10
	record.NextRetry = 20
	record.VerifiedAt = 30

	err = persistence.UpdateRecords([]*VerificationRecord{&record})
	require.NoError(t, err)

	toBeVerified, err := persistence.GetENSToBeVerified(20)
	require.NoError(t, err)
	require.Len(t, toBeVerified, 1)
	require.False(t, toBeVerified[0].Verified)
	require.Equal(t, uint64(10), toBeVerified[0].VerificationRetries)
	require.Equal(t, uint64(20), toBeVerified[0].NextRetry)
	require.Equal(t, uint64(30), toBeVerified[0].VerifiedAt)
	require.Equal(t, updatedName, toBeVerified[0].Name)
	require.Equal(t, pk, toBeVerified[0].PublicKey)
}
