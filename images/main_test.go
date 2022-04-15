package images

import (
	"image"
	"image/png"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateBannerImage_NoScaleUp(t *testing.T) {
	// Test image 256x256
	testImage := path + "status.png"
	identityImage, err := GenerateBannerImage(testImage, 50, 50, 150, 100)

	require.NoError(t, err)
	require.Exactly(t, identityImage.Name, BannerIdentityName)
	require.Positive(t, len(identityImage.Payload))
	// Ensure we don't scale it up. That will be done inefficiently in backend instead of frontend
	require.Exactly(t, identityImage.Width, 100)
	require.Exactly(t, identityImage.Height, 50)
	require.Exactly(t, identityImage.FileSize, len(identityImage.Payload))
	require.Exactly(t, identityImage.ResizeTarget, int(BannerDim))
}

func TestGenerateBannerImage_ShrinkOnly(t *testing.T) {
	// Generate test image bigger than BannerDim
	testImage := image.NewRGBA(image.Rect(0, 0, int(BannerDim)+10, int(BannerDim)+20))

	tmpTestFilePath := t.TempDir() + "/test.png"
	file, err := os.Create(tmpTestFilePath)
	require.NoError(t, err)
	defer file.Close()

	err = png.Encode(file, testImage)
	require.NoError(t, err)

	identityImage, error := GenerateBannerImage(tmpTestFilePath, 0, 0, int(BannerDim)+5, int(BannerDim)+10)

	require.NoError(t, error)
	require.Positive(t, len(identityImage.Payload))
	// Ensure we scale it down by the small side
	require.Exactly(t, identityImage.Width, int(BannerDim))
	require.Exactly(t, identityImage.Height, 805)
}
