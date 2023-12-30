package stickers

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/multiaccounts/accounts"
	mock_settings "github.com/status-im/status-go/multiaccounts/settings/mocks"
	"github.com/status-im/status-go/services/wallet/bigint"
)

func SetupAPI(t *testing.T) (*API, *mock_settings.MockDatabaseSettingsManager) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mock_settings.NewMockDatabaseSettingsManager(ctrl)

	require.NotNil(t, mockDB)

	accountDatabase := &accounts.Database{
		DatabaseSettingsManager: mockDB,
	}

	api := NewAPI(context.Background(), accountDatabase, nil, nil, nil, "test-store-dir", nil, nil)
	require.NotNil(t, api)

	return api, mockDB
}

func Test_WHEN_EmptyRecentStickers_And_EmptyStickerPacks_THEN_EmptyStickers_Returned(t *testing.T) {

	api, mockDB := SetupAPI(t)

	mockDB.EXPECT().GetInstalledStickerPacks().Return(nil, nil)

	actual, err := api.recentStickers()

	assert.NoError(t, err)
	assert.Equal(t, []Sticker{}, actual)
}

func Test_WHEN_EmptyStickerPacks_THEN_EmptyStickers_Returned(t *testing.T) {
	expectedStickers := []Sticker{
		{
			PackID: &bigint.BigInt{Int: big.NewInt(1)},
			URL:    "sticker1",
			Hash:   "0x1",
		},
		{
			PackID: &bigint.BigInt{Int: big.NewInt(2)},
			URL:    "sticker2",
			Hash:   "0x2",
		},
	}

	expectedStickerPacks := map[string]StickerPack{}

	api, mockDB := SetupAPI(t)

	expectedStickersJSON, err := json.Marshal(expectedStickers)
	require.NoError(t, err)

	expectedStickerPacksJSON, err := json.Marshal(expectedStickerPacks)
	require.NoError(t, err)

	mockDB.EXPECT().GetInstalledStickerPacks().Return((*json.RawMessage)(&expectedStickerPacksJSON), nil)
	mockDB.EXPECT().GetRecentStickers().Return((*json.RawMessage)(&expectedStickersJSON), nil)

	actual, err := api.recentStickers()

	require.NoError(t, err)
	require.Equal(t, 0, len(actual))
	require.Equal(t, []Sticker{}, actual)
}

func Test_WHEN_Stickers_In_Existing_SingleStickerPack_THEN_Stickers_Returned(t *testing.T) {
	expectedStickers := []Sticker{
		{
			PackID: &bigint.BigInt{Int: big.NewInt(1)},
			URL:    "sticker1",
			Hash:   "0x1",
		},
		{
			PackID: &bigint.BigInt{Int: big.NewInt(2)},
			URL:    "sticker2",
			Hash:   "0x2",
		},
	}

	expectedStickerPacks := map[string]StickerPack{
		"1": {
			ID:        &bigint.BigInt{Int: big.NewInt(1)},
			Name:      "test",
			Author:    "test",
			Owner:     [20]byte{},
			Price:     &bigint.BigInt{Int: big.NewInt(10)},
			Preview:   "",
			Thumbnail: "",
			Stickers:  expectedStickers,
			Status:    1,
		},
		"2": {
			ID:        &bigint.BigInt{Int: big.NewInt(2)},
			Name:      "test",
			Author:    "test",
			Owner:     [20]byte{},
			Price:     &bigint.BigInt{Int: big.NewInt(10)},
			Preview:   "",
			Thumbnail: "",
			Stickers:  expectedStickers,
			Status:    1,
		},
	}

	api, mockDB := SetupAPI(t)

	expectedStickersJSON, err := json.Marshal(expectedStickers)
	require.NoError(t, err)

	expectedStickerPacksJSON, err := json.Marshal(expectedStickerPacks)
	require.NoError(t, err)

	mockDB.EXPECT().GetInstalledStickerPacks().Return((*json.RawMessage)(&expectedStickerPacksJSON), nil)
	mockDB.EXPECT().GetRecentStickers().Return((*json.RawMessage)(&expectedStickersJSON), nil)

	actual, err := api.recentStickers()

	require.NoError(t, err)
	require.Equal(t, 2, len(actual))
	require.Equal(t, expectedStickers, actual)
}

func Test_WHEN_Stickers_In_Existing_In_MultipleStickerPacks_THEN_Stickers_Returned(t *testing.T) {
	expectedStickers := []Sticker{
		{
			PackID: &bigint.BigInt{Int: big.NewInt(1)},
			URL:    "sticker1",
			Hash:   "0x1",
		},
		{
			PackID: &bigint.BigInt{Int: big.NewInt(2)},
			URL:    "sticker2",
			Hash:   "0x2",
		},
	}

	expectedStickerPacks := map[string]StickerPack{
		"1": {
			ID:        &bigint.BigInt{Int: big.NewInt(1)},
			Name:      "test",
			Author:    "test",
			Owner:     [20]byte{},
			Price:     &bigint.BigInt{Int: big.NewInt(10)},
			Preview:   "",
			Thumbnail: "",
			Stickers:  expectedStickers,
			Status:    1,
		},
		"2": {
			ID:        &bigint.BigInt{Int: big.NewInt(2)},
			Name:      "test",
			Author:    "test",
			Owner:     [20]byte{},
			Price:     &bigint.BigInt{Int: big.NewInt(10)},
			Preview:   "",
			Thumbnail: "",
			Stickers:  expectedStickers,
			Status:    1,
		},
		"3": {
			ID:        &bigint.BigInt{Int: big.NewInt(3)},
			Name:      "test",
			Author:    "test",
			Owner:     [20]byte{},
			Price:     &bigint.BigInt{Int: big.NewInt(10)},
			Preview:   "",
			Thumbnail: "",
			Stickers:  expectedStickers,
			Status:    1,
		},
	}

	api, mockDB := SetupAPI(t)

	expectedStickersJSON, err := json.Marshal(expectedStickers)
	require.NoError(t, err)

	expectedStickerPacksJSON, err := json.Marshal(expectedStickerPacks)
	require.NoError(t, err)

	mockDB.EXPECT().GetInstalledStickerPacks().Return((*json.RawMessage)(&expectedStickerPacksJSON), nil)
	mockDB.EXPECT().GetRecentStickers().Return((*json.RawMessage)(&expectedStickersJSON), nil)

	actual, err := api.recentStickers()

	require.NoError(t, err)
	require.Equal(t, 2, len(actual))
	require.Equal(t, expectedStickers, actual)
}
