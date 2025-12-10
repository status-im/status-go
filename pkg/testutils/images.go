package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/images"
)

func SampleIdentityImages() []images.IdentityImage {
	return []images.IdentityImage{
		{
			Name:         images.SmallDimName,
			Payload:      gofakeit.ImageJpeg(80, 80),
			Width:        80,
			Height:       80,
			FileSize:     256,
			ResizeTarget: 80,
			Clock:        0,
		},
		{
			Name:         images.LargeDimName,
			Payload:      gofakeit.ImagePng(300, 300),
			Width:        240,
			Height:       300,
			FileSize:     1024,
			ResizeTarget: 240,
			Clock:        0,
		},
	}
}

func SaveFakeImage(t *testing.T, width int, height int) string {
	tempdir := t.TempDir()
	payload := gofakeit.ImagePng(width, height)
	imagePath := filepath.Join(tempdir, gofakeit.LetterN(5)+".jpg")
	err := os.WriteFile(imagePath, payload, 0644)
	require.NoError(t, err)
	return imagePath
}
