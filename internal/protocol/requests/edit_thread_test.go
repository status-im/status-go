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
			maxLength := strings.Repeat(grapheme, common.MaxThreadNameLength)
			require.NoError(t, (&EditThread{ChatID: "chat", ThreadID: "thread", Name: &maxLength}).Validate())
			overlong := maxLength + grapheme
			require.ErrorIs(t, (&EditThread{ChatID: "chat", ThreadID: "thread", Name: &overlong}).Validate(), ErrEditThreadNameTooLong)
		})
	}

	tooLong := strings.Repeat("👩🏽‍💻", common.MaxThreadNameLength+1)
	require.ErrorIs(t, (&EditThread{ChatID: "chat", ThreadID: "thread", Name: &tooLong}).Validate(), ErrEditThreadNameTooLong)
}
