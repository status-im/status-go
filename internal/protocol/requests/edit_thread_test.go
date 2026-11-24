package requests

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/status-im/status-go/internal/protocol/common"
)

func TestEditThreadValidate(t *testing.T) {
	name := "  Updated\nthread name "
	require.NoError(t, (&EditThread{ChatID: "chat", ThreadID: "thread", Name: &name}).Validate())

	testCases := []struct {
		name    string
		request *EditThread
		err     error
	}{
		{"nil", nil, ErrEditThreadInvalidRequest},
		{"missing-name", &EditThread{ChatID: "chat", ThreadID: "thread"}, ErrEditThreadInvalidRequest},
		{"missing-chat", &EditThread{ThreadID: "thread", Name: &name}, ErrEditThreadInvalidChatID},
		{"missing-thread", &EditThread{ChatID: "chat", Name: &name}, ErrEditThreadInvalidID},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.ErrorIs(t, testCase.request.Validate(), testCase.err)
		})
	}

	blank := " \n\t "
	require.ErrorIs(t, (&EditThread{ChatID: "chat", ThreadID: "thread", Name: &blank}).Validate(), ErrEditThreadInvalidName)

	for _, grapheme := range []string{"a", "e\u0301", "👩🏽‍💻"} {
		t.Run("grapheme-boundary-"+grapheme, func(t *testing.T) {
			maxLength := strings.Repeat(grapheme, common.MaxThreadNameLengthForRenaming)
			require.NoError(t, (&EditThread{ChatID: "chat", ThreadID: "thread", Name: &maxLength}).Validate())
			overlong := maxLength + grapheme
			require.ErrorIs(t, (&EditThread{ChatID: "chat", ThreadID: "thread", Name: &overlong}).Validate(), ErrEditThreadNameTooLongForRenaming)
		})
	}

	longName := strings.Repeat("👩🏽‍💻", common.MaxThreadNameLength+1)
	require.NoError(t, (&EditThread{ChatID: "chat", ThreadID: "thread", Name: &longName}).Validate())
}

func TestNormalizeThreadNameLimits(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		normalize func(string, bool) (string, bool)
		limit     int
	}{
		{"creation", common.NormalizeThreadName, common.MaxThreadNameLength},
		{"renaming", common.NormalizeThreadNameForRenaming, common.MaxThreadNameLengthForRenaming},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			name, overflow := testCase.normalize(" \n\t ", false)
			require.Empty(t, name)
			require.False(t, overflow)

			name, overflow = testCase.normalize("  Thread\n\t name  ", false)
			require.Equal(t, "Thread name", name)
			require.False(t, overflow)

			for _, grapheme := range []string{"a", "e\u0301", "👩🏽‍💻"} {
				boundary := strings.Repeat(grapheme, testCase.limit)
				name, overflow = testCase.normalize(" \n"+boundary+"\t ", false)
				require.Equal(t, boundary, name)
				require.False(t, overflow)

				name, overflow = testCase.normalize(boundary+grapheme, false)
				require.Equal(t, boundary+grapheme, name)
				require.True(t, overflow)

				name, overflow = testCase.normalize(boundary+grapheme, true)
				require.Equal(t, boundary, name)
				require.True(t, overflow)
			}
		})
	}
}
