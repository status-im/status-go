package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/images"
)

var (
	testJpegBytes = []byte{0xff, 0xd8, 0xff, 0xdb, 0x00, 0x84, 0x00, 0x50, 0x37, 0x3c, 0x46, 0x3c, 0x32, 0x50}
	testPngBytes  = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48}
)

func SampleIdentityImages() []images.IdentityImage {
	return []images.IdentityImage{
		{
			Name:         images.SmallDimName,
			Payload:      testJpegBytes,
			Width:        80,
			Height:       80,
			FileSize:     256,
			ResizeTarget: 80,
			Clock:        0,
		},
		{
			Name:         images.LargeDimName,
			Payload:      testPngBytes,
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
	err := os.WriteFile(imagePath, payload, 0600)
	require.NoError(t, err)
	return imagePath
}
