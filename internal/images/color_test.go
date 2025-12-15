package images

import (
	"image/color"
	"testing"
)

func TestParseColor(t *testing.T) {
	// Test hex color string format
	hexColor := "#FF00FF"
	expectedResult := color.RGBA{R: 255, G: 0, B: 255, A: 255}
	result, err := ParseColor(hexColor)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if result != expectedResult {
		t.Errorf("unexpected result: %v (expected %v)", result, expectedResult)
	}

	// Test RGB color string format
	rgbColor := "rgb(255, 0, 255)"
	expectedResult = color.RGBA{R: 255, G: 0, B: 255, A: 255}
	result, err = ParseColor(rgbColor)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if result != expectedResult {
		t.Errorf("unexpected result: %v (expected %v)", result, expectedResult)
	}

	// Test RGBA color string format
	rgbaColor := "rgba(255, 0, 255, 1)"
	expectedResult = color.RGBA{R: 255, G: 0, B: 255, A: 255}
	result, err = ParseColor(rgbaColor)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if result != expectedResult {
		t.Errorf("unexpected result: %v (expected %v)", result, expectedResult)
	}

	// Test invalid color string format
	invalidColor := "blah"
	_, err = ParseColor(invalidColor)
	if err == nil {
		t.Errorf("expected error, but got none")
	}
}
