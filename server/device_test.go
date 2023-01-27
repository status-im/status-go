package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testCaseInput = []string{
		"some-computer-name.local",
		"Hello.local",
		"I'm an input.locally",
		"some-plain-input",
		"smol",
	}
)

func TestRemoveSuffix(t *testing.T) {
	tce := []string{
		"some-computer-name",
		"Hello",
		"I'm an input.locally",
		"some-plain-input",
		"smol",
	}

	for i, tci := range testCaseInput {
		require.Equal(t, tce[i], RemoveSuffix(tci, local))
	}
}

func TestParseHostname(t *testing.T) {
	tce := []string{
		"some computer name",
		"Hello",
		"I'm an input.locally",
		"some plain input",
		"smol",
	}

	for i, tci := range testCaseInput {
		require.Equal(t, tce[i], parseHostname(tci))
	}
}
