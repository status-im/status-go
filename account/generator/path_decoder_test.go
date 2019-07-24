package generator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodePath(t *testing.T) {
	scenarios := []struct {
		path                  string
		expectedPath          []uint32
		expectedStartingPoint startingPoint
		err                   error
	}{
		{
			path:                  "",
			expectedPath:          []uint32{},
			expectedStartingPoint: startingPointCurrent,
		},
		{
			path:                  "1",
			expectedPath:          []uint32{1},
			expectedStartingPoint: startingPointCurrent,
		},
		{
			path:                  "..",
			expectedPath:          []uint32{},
			expectedStartingPoint: startingPointParent,
		},
		{
			path:                  "m",
			expectedPath:          []uint32{},
			expectedStartingPoint: startingPointMaster,
		},
		{
			path:                  "m/1",
			expectedPath:          []uint32{1},
			expectedStartingPoint: startingPointMaster,
		},
		{
			path:                  "m/1/2",
			expectedPath:          []uint32{1, 2},
			expectedStartingPoint: startingPointMaster,
		},
		{
			path:                  "m/1/2'/3",
			expectedPath:          []uint32{1, 2147483650, 3},
			expectedStartingPoint: startingPointMaster,
		},
		{
			path: "m/",
			err:  fmt.Errorf("error parsing derivation path m/; at position 2, expected number, got EOF"),
		},
		{
			path: "m/1//2",
			err:  fmt.Errorf("error parsing derivation path m/1//2; at position 5, expected number, got /"),
		},
		{
			path: "m/1'2",
			err:  fmt.Errorf("error parsing derivation path m/1'2; at position 5, expected /, got 2"),
		},
		{
			path: "m/'/2",
			err:  fmt.Errorf("error parsing derivation path m/'/2; at position 3, expected number, got '"),
		},
		{
			path: "m/2147483648",
			err:  fmt.Errorf("error parsing derivation path m/2147483648; at position 3, index must be lower than 2^31, got 2147483648"),
		},
	}

	for i, s := range scenarios {
		t.Run(fmt.Sprintf("scenario %d", i), func(t *testing.T) {
			startingP, path, err := decodePath(s.path)
			if s.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, s.expectedStartingPoint, startingP)
				assert.Equal(t, s.expectedPath, path)
			} else {
				assert.Equal(t, s.err, err)
			}
		})
	}
}
