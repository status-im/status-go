package images

import (
	"testing"
)

func TestExtractInitials(t *testing.T) {
	testCases := []struct {
		fullName         string
		amountInitials   int
		expectedInitials string
	}{
		{"John Doe", 1, "J"},
		{"John Doe", 2, "JD"},
		{"John    Doe", 2, "JD"},
		{"Jane ", 2, "J"},
		{"Xxxx", 2, "X"},
		{"", 2, ""},
	}

	for _, tc := range testCases {
		actualInitials := ExtractInitials(tc.fullName, tc.amountInitials)
		if actualInitials != tc.expectedInitials {
			t.Errorf("Unexpected result for %q with %d initials, expected %q but got %q", tc.fullName, tc.amountInitials, tc.expectedInitials, actualInitials)
		}
	}
}
